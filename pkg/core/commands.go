package core

import (
	"fmt"
	"maps"
	"time"
)

func saveCommit(repoRoot, message string, parents []string, files map[string]string) error {
	hash, err := makeCommit(repoRoot, message, parents, files)
	if err != nil {
		return err
	}
	if err := moveHEAD(repoRoot, hash); err != nil {
		return err
	}
	fmt.Printf("[%s] %s\n", hash[:7], message)
	return nil
}

func makeCommit(repoRoot, message string, parents []string, files map[string]string) (string, error) {
	return WriteCommit(repoRoot, Commit{
		Parents: parents, Message: message, Timestamp: time.Now(), Files: files,
	})
}

func EnsureCleanWorkspace(repoRoot string) error {
	staged, modified, _, err := GetStatus(repoRoot)
	if err != nil {
		return err
	}
	if len(staged)+len(modified) > 0 {
		return fmt.Errorf("working tree is not clean; commit your changes before proceeding")
	}
	return nil
}

func Checkout(repoRoot, target string) error {
	commit, branch, err := resolveCommit(repoRoot, target)
	if err != nil {
		return err
	}
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

	if err := updateWorktree(repoRoot, head.Files, commit.Files, commit.Files, false, func() error {
		if branch {
			return UpdateHEAD(repoRoot, target)
		}
		return writeMetadata(repoRoot, "HEAD", []byte(commit.ID+"\n"))
	}); err != nil {
		return err
	}
	if branch {
		fmt.Printf("Switched to branch %q\n", target)
	} else {
		fmt.Printf("HEAD detached at %s\n", commit.ID)
	}
	return nil
}

func Reset(repoRoot, mode, target string) error {
	if mode != "soft" && mode != "mixed" && mode != "hard" {
		return fmt.Errorf("unknown reset mode: %q", mode)
	}
	commit, _, err := resolveCommit(repoRoot, target)
	if err != nil {
		return err
	}
	merge, err := readMerge(repoRoot)
	if err != nil {
		return err
	}
	if merge.Target != "" && mode != "hard" {
		return fmt.Errorf("finish the merge or use a hard reset to abandon it")
	}

	updateHead := func() error {
		return moveHEAD(repoRoot, commit.ID)
	}
	switch mode {
	case "soft":
		err = updateHead()
	case "mixed":
		err = updateIndex(repoRoot, commit.Files, updateHead)
	case "hard":
		head, err := readHEAD(repoRoot)
		if err != nil {
			return err
		}
		idx, err := ReadIndex(repoRoot)
		if err != nil {
			return err
		}
		maps.Copy(head.Files, idx.Entries)
		for name := range merge.Conflicts {
			head.Files[name] = "conflict"
		}
		if err := updateWorktree(repoRoot, head.Files, commit.Files, commit.Files, true, func() error {
			return finishMerge(repoRoot, merge, updateHead)
		}); err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	fmt.Printf("HEAD is now at %s\n", commit.ID[:7])
	return nil
}

// History returns the first-parent history of the checked-out revision.
func History(repoRoot string) ([]Commit, error) {
	hash, err := GetHEADCommit(repoRoot)
	if err != nil {
		return nil, err
	}
	commits := []Commit{}
	for hash != "" {
		commit, err := ReadCommit(repoRoot, hash)
		if err != nil {
			return nil, err
		}
		commits = append(commits, *commit)
		if len(commit.Parents) == 0 {
			break
		}
		hash = commit.Parents[0]
	}
	return commits, nil
}

func Log(repoRoot string) error {
	commits, err := History(repoRoot)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		return fmt.Errorf("your current branch does not have any commits yet")
	}
	for _, commit := range commits {
		fmt.Printf("commit %s\nDate:   %s\n\n    %s\n\n", commit.ID, commit.Timestamp.Format(time.RFC1123Z), commit.Message)
	}
	return nil
}

func Status(repoRoot string) error {
	branch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		return err
	}
	staged, modified, untracked, err := GetStatus(repoRoot)
	if err != nil {
		return err
	}
	merge, conflicts, err := MergeStatus(repoRoot)
	if err != nil {
		return err
	}
	if branch != "" {
		fmt.Printf("On branch %s\n", branch)
	} else {
		hash, err := GetHEADCommit(repoRoot)
		if err != nil {
			return err
		}
		fmt.Printf("HEAD detached at %s\n", hash[:7])
	}

	if merge {
		fmt.Println("Merge in progress. Resolve and stage conflicts, then commit.")
	}
	printChanges("Unresolved conflicts", "conflict", conflicts)
	printChanges("Changes to be committed", "staged", staged)
	printChanges("Changes not staged for commit", "modified", modified)
	printChanges("Untracked files", "untracked", untracked)
	if !merge && len(staged)+len(modified)+len(untracked) == 0 {
		fmt.Println("\nnothing to commit, working tree clean")
	}
	return nil
}

func printChanges(heading, label string, paths []string) {
	if len(paths) == 0 {
		return
	}
	fmt.Printf("\n%s:\n", heading)
	for _, path := range paths {
		fmt.Printf("  (%s)  %s\n", label, path)
	}
}
