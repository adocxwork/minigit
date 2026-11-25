package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"unicode/utf8"
)

const mergeFile = "MERGE_HEAD"

type mergeState struct {
	Target    string            `json:"target"`
	Conflicts map[string]string `json:"conflicts"`
}

func readMerge(repoRoot string) (mergeState, error) {
	state := mergeState{Conflicts: map[string]string{}}
	data, err := readMetadata(repoRoot, mergeFile)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if text := strings.TrimSpace(string(data)); validObjectID(text) {
		// Older repositories stored only the second parent.
		state.Target = text
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("read merge state: %w", err)
	}
	if !validObjectID(state.Target) {
		return state, fmt.Errorf("invalid merge target")
	}
	for name, hash := range state.Conflicts {
		if !validWorkPath(name) || (hash != "" && !validObjectID(hash)) {
			return state, fmt.Errorf("invalid merge conflict entry")
		}
	}
	return state, nil
}

func writeMerge(repoRoot string, state mergeState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeMetadata(repoRoot, mergeFile, data)
}

func clearMerge(repoRoot string) error {
	meta, err := openMetadata(repoRoot)
	if err != nil {
		return err
	}
	defer meta.Close()
	err = meta.Remove(mergeFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func finishMerge(repoRoot string, state mergeState, finish func() error) error {
	if state.Target == "" {
		return finish()
	}
	if err := clearMerge(repoRoot); err != nil {
		return err
	}
	if err := finish(); err != nil {
		return errors.Join(err, writeMerge(repoRoot, state))
	}
	return nil
}

func MergeStatus(repoRoot string) (bool, []string, error) {
	state, err := readMerge(repoRoot)
	conflicts := append([]string{}, slices.Sorted(maps.Keys(state.Conflicts))...)
	return state.Target != "", conflicts, err
}

func requireNoMerge(repoRoot string) error {
	merging, _, err := MergeStatus(repoRoot)
	if err != nil {
		return err
	}
	if merging {
		return fmt.Errorf("finish the merge or use a hard reset to abandon it")
	}
	return nil
}

// commitGraph returns all revisions reachable from a tip, with parents intact.
func commitGraph(repoRoot, tip string) (map[string]*Commit, error) {
	graph := map[string]*Commit{}
	queue := []string{tip}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if graph[id] != nil {
			continue
		}
		commit, err := ReadCommit(repoRoot, id)
		if err != nil {
			return nil, err
		}
		graph[id] = commit
		queue = append(queue, commit.Parents...)
	}
	return graph, nil
}

func mergeBase(ours, theirs map[string]*Commit) (*Commit, error) {
	// A common revision with a common child cannot be a best merge base.
	// All intermediate revisions between two common revisions are common too.
	superseded := map[string]bool{}
	for id, commit := range ours {
		if theirs[id] != nil {
			for _, parent := range commit.Parents {
				superseded[parent] = true
			}
		}
	}
	var base *Commit
	for id, commit := range ours {
		if theirs[id] == nil || superseded[id] {
			continue
		}
		if base != nil {
			return nil, fmt.Errorf("multiple merge bases; merge this history manually")
		}
		base = commit
	}
	if base == nil {
		return nil, fmt.Errorf("branches have no common ancestor")
	}
	return base, nil
}

func Merge(repoRoot, targetBranch string) error {
	if err := requireNoMerge(repoRoot); err != nil {
		return err
	}
	if err := EnsureCleanWorkspace(repoRoot); err != nil {
		return err
	}
	head, err := readHEAD(repoRoot)
	if err != nil {
		return err
	}
	target, err := GetBranchCommit(repoRoot, targetBranch)
	if err != nil {
		return err
	}
	if head.ID == "" {
		return fmt.Errorf("create a commit before merging")
	}

	theirs, err := commitGraph(repoRoot, target)
	if err != nil {
		return err
	}
	if theirs[head.ID] != nil && head.ID != target {
		fmt.Println("Fast-forwarding...")
		return updateWorktree(repoRoot, head.Files, theirs[target].Files, theirs[target].Files, false, func() error {
			return moveHEAD(repoRoot, target)
		})
	}
	ours, err := commitGraph(repoRoot, head.ID)
	if err != nil {
		return err
	}
	if ours[target] != nil {
		fmt.Println("Already up to date.")
		return nil
	}

	base, err := mergeBase(ours, theirs)
	if err != nil {
		return err
	}
	return mergeSnapshots(repoRoot, targetBranch, head, theirs[target], base)
}

func mergeSnapshots(repoRoot, targetBranch string, head, target, base *Commit) error {
	meta, err := openMetadata(repoRoot)
	if err != nil {
		return err
	}
	defer meta.Close()

	merged := map[string]string{}
	conflicts := map[string]string{}
	all := map[string]string{}
	maps.Copy(all, head.Files)
	maps.Copy(all, target.Files)
	maps.Copy(all, base.Files)
	for _, name := range slices.Sorted(maps.Keys(all)) {
		ours, theirs, ancestor := head.Files[name], target.Files[name], base.Files[name]
		switch {
		case ours == theirs || theirs == ancestor:
			if ours != "" {
				merged[name] = ours
			}
		case ours == ancestor:
			if theirs != "" {
				merged[name] = theirs
			}
		default:
			if ours != "" {
				merged[name] = ours
			}
			conflicts[name] = theirs
		}
	}

	if len(conflicts) == 0 {
		message := fmt.Sprintf("Merge branch %q", targetBranch)
		id, err := makeCommit(repoRoot, message, []string{head.ID, target.ID}, merged)
		if err != nil {
			return err
		}
		if err := updateWorktree(repoRoot, head.Files, merged, merged, false, func() error {
			return moveHEAD(repoRoot, id)
		}); err != nil {
			return err
		}
		fmt.Printf("[%s] %s\n", id[:7], message)
		return nil
	}

	working := maps.Clone(merged)
	state := mergeState{Target: target.ID, Conflicts: map[string]string{}}
	for _, name := range slices.Sorted(maps.Keys(conflicts)) {
		hash, markerHash, err := conflictFile(meta, head.Files[name], conflicts[name], targetBranch)
		if err != nil {
			return err
		}
		working[name] = hash
		state.Conflicts[name] = markerHash
	}

	// Conflict files are in the working tree. The index retains the versions
	// from HEAD until the user stages a resolution.
	if err := updateWorktree(repoRoot, head.Files, working, merged, false, func() error {
		return writeMerge(repoRoot, state)
	}); err != nil {
		return err
	}
	return fmt.Errorf("merge conflicts in %s; resolve and stage them, then commit", strings.Join(slices.Sorted(maps.Keys(conflicts)), ", "))
}

func conflictFile(meta *os.Root, ours, theirs, target string) (string, string, error) {
	var current, incoming bytes.Buffer
	if ours != "" {
		if err := copyBlob(meta, ours, &current); err != nil {
			return "", "", err
		}
	}
	if theirs != "" {
		if err := copyBlob(meta, theirs, &incoming); err != nil {
			return "", "", err
		}
	}
	if !isText(current.Bytes()) || !isText(incoming.Bytes()) {
		if ours != "" {
			return ours, "", nil
		}
		return theirs, "", nil
	}
	for _, buffer := range []*bytes.Buffer{&current, &incoming} {
		if data := buffer.Bytes(); len(data) > 0 && data[len(data)-1] != '\n' {
			buffer.WriteByte('\n')
		}
	}
	markers := fmt.Sprintf("<<<<<<< HEAD\n%s=======\n%s>>>>>>> %s\n", &current, &incoming, target)
	hash, err := storeBlob(meta, strings.NewReader(markers))
	return hash, hash, err
}

func isText(data []byte) bool {
	return utf8.Valid(data) && bytes.IndexByte(data, 0) < 0
}
