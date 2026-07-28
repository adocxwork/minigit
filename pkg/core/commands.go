package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mgit/pkg/utils"
)

// Add stages a file or directory.
func Add(repoRoot, path string) error {
	idx, err := ReadIndex(repoRoot)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	err = filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the .mgit directory itself
		relToRoot, err := filepath.Rel(repoRoot, p)
		if err != nil {
			return err
		}
		if relToRoot == MgitDir || strings.HasPrefix(relToRoot, MgitDir+string(filepath.Separator)) {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		hash, err := WriteBlob(repoRoot, p)
		if err != nil {
			return fmt.Errorf("failed to write blob for %s: %w", p, err)
		}

		idx.Entries[relToRoot] = hash
		return nil
	})

	if err != nil {
		return err
	}

	return WriteIndex(repoRoot, idx)
}

// CreateCommit creates a new commit from the staging area.
func CreateCommit(repoRoot, message string, parents []string) error {
	mergeHeadPath := filepath.Join(repoRoot, MgitDir, "MERGE_HEAD")
	if utils.Exists(mergeHeadPath) {
		content, err := os.ReadFile(mergeHeadPath)
		if err == nil {
			mergeHash := strings.TrimSpace(string(content))
			found := false
			for _, p := range parents {
				if p == mergeHash {
					found = true
				}
			}
			if !found && mergeHash != "" {
				parents = append(parents, mergeHash)
			}
		}
	}

	idx, err := ReadIndex(repoRoot)
	if err != nil {
		return err
	}

	if len(idx.Entries) == 0 {
		return fmt.Errorf("nothing to commit, working tree clean")
	}

	commit := Commit{
		Parents:   parents,
		Message:   message,
		Timestamp: time.Now(),
		Files:     make(map[string]string),
	}

	for k, v := range idx.Entries {
		commit.Files[k] = v
	}

	commitHash, err := WriteCommit(repoRoot, commit)
	if err != nil {
		return err
	}

	branch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		return err
	}

	if branch != "" {
		if err := UpdateBranch(repoRoot, branch, commitHash); err != nil {
			return err
		}
	} else {
		// Detached HEAD, just update HEAD file
		headPath := filepath.Join(repoRoot, MgitDir, "HEAD")
		if err := os.WriteFile(headPath, []byte(commitHash+"\n"), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("[%s] %s\n", commitHash[:7], message)
	os.Remove(filepath.Join(repoRoot, MgitDir, "MERGE_HEAD"))
	return nil
}

func restoreFiles(repoRoot, oldCommitHash, newCommitHash string) error {
	var oldCommit *Commit
	if oldCommitHash != "" {
		oldCommit, _ = ReadCommit(repoRoot, oldCommitHash)
	}
	
	newCommit, err := ReadCommit(repoRoot, newCommitHash)
	if err != nil {
		return err
	}

	// 1. Remove files from the old commit
	if oldCommit != nil {
		for file := range oldCommit.Files {
			os.Remove(filepath.Join(repoRoot, file))
			dir := filepath.Dir(filepath.Join(repoRoot, file))
			for dir != repoRoot {
				os.Remove(dir)
				dir = filepath.Dir(dir)
			}
		}
	}

	// 2. Write files from new commit
	idx := &Index{Entries: make(map[string]string)}
	for relPath, blobHash := range newCommit.Files {
		absPath := filepath.Join(repoRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return err
		}

		blobPath := filepath.Join(repoRoot, MgitDir, "objects", blobHash)
		err = func() error {
			src, err := os.Open(blobPath)
			if err != nil {
				return err
			}
			defer src.Close()

			dst, err := os.Create(absPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			if _, err := io.Copy(dst, src); err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}

		idx.Entries[relPath] = blobHash
	}

	// 3. Update index
	return WriteIndex(repoRoot, idx)
}

// Checkout switches to a branch or commit, updating the working directory.
func Checkout(repoRoot, target string) error {
	// First, determine if target is a branch or a commit hash
	commitHash := target
	isBranch := false

	if utils.Exists(filepath.Join(repoRoot, MgitDir, "refs", "heads", target)) {
		isBranch = true
		hash, err := GetBranchCommit(repoRoot, target)
		if err != nil {
			return err
		}
		commitHash = hash
	} else {
		// Check if it's a valid commit object
		if !utils.Exists(filepath.Join(repoRoot, MgitDir, "objects", target)) {
			return fmt.Errorf("pathspec '%s' did not match any file(s) known to mgit", target)
		}
	}

	headCommitHash, _ := GetHEADCommit(repoRoot)
	if err := restoreFiles(repoRoot, headCommitHash, commitHash); err != nil {
		return err
	}

	// 4. Update HEAD
	if isBranch {
		if err := UpdateHEAD(repoRoot, target); err != nil {
			return err
		}
		fmt.Printf("Switched to branch '%s'\n", target)
	} else {
		headPath := filepath.Join(repoRoot, MgitDir, "HEAD")
		if err := os.WriteFile(headPath, []byte(commitHash+"\n"), 0644); err != nil {
			return err
		}
		fmt.Printf("Note: switching to '%s'.\nYou are in 'detached HEAD' state.\n", commitHash)
	}

	return nil
}

// GetStatus computes the repository status.
func GetStatus(repoRoot string) (staged, modified, untracked []string, err error) {
	staged = []string{}
	modified = []string{}
	untracked = []string{}
	
	idx, err := ReadIndex(repoRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	headCommitHash, _ := GetHEADCommit(repoRoot)
	headFiles := make(map[string]string)
	if headCommitHash != "" {
		commit, err := ReadCommit(repoRoot, headCommitHash)
		if err == nil {
			headFiles = commit.Files
		}
	}

	workDirFiles := make(map[string]string)
	err = filepath.Walk(repoRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relToRoot, err := filepath.Rel(repoRoot, p)
		if err != nil {
			return err
		}
		if relToRoot == MgitDir || strings.HasPrefix(relToRoot, MgitDir+string(filepath.Separator)) {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			hash, _ := utils.HashFile(p)
			workDirFiles[relToRoot] = hash
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	for file, wdHash := range workDirFiles {
		idxHash, inIdx := idx.Entries[file]
		if !inIdx {
			untracked = append(untracked, file)
		} else if idxHash != wdHash {
			modified = append(modified, file)
		}
	}

	for file, idxHash := range idx.Entries {
		headHash, inHead := headFiles[file]
		if !inHead || headHash != idxHash {
			staged = append(staged, file)
		}
	}

	sort.Strings(staged)
	sort.Strings(modified)
	sort.Strings(untracked)

	return staged, modified, untracked, nil
}

// Status prints the status of the repository.
func Status(repoRoot string) error {
	branch, _ := GetCurrentBranch(repoRoot)
	if branch != "" {
		fmt.Printf("On branch %s\n", branch)
	} else {
		headCommit, _ := GetHEADCommit(repoRoot)
		if headCommit != "" {
			fmt.Printf("HEAD detached at %s\n", headCommit[:7])
		} else {
			fmt.Println("No commits yet")
		}
	}

	staged, modified, untracked, err := GetStatus(repoRoot)
	if err != nil {
		return err
	}

	if len(staged) > 0 {
		fmt.Println("\nChanges to be committed:")
		for _, f := range staged {
			fmt.Printf("  (staged)  %s\n", f)
		}
	}

	if len(modified) > 0 {
		fmt.Println("\nChanges not staged for commit:")
		for _, f := range modified {
			fmt.Printf("  (modified) %s\n", f)
		}
	}

	if len(untracked) > 0 {
		fmt.Println("\nUntracked files:")
		for _, f := range untracked {
			fmt.Printf("  (untracked) %s\n", f)
		}
	}

	if len(staged) == 0 && len(modified) == 0 && len(untracked) == 0 {
		fmt.Println("\nnothing to commit, working tree clean")
	}

	return nil
}

// Log prints the commit history.
func Log(repoRoot string) error {
	commitHash, err := GetHEADCommit(repoRoot)
	if err != nil {
		return err
	}

	if commitHash == "" {
		return fmt.Errorf("your current branch does not have any commits yet")
	}

	for commitHash != "" {
		commit, err := ReadCommit(repoRoot, commitHash)
		if err != nil {
			return err
		}

		fmt.Printf("commit %s\n", commit.ID)
		fmt.Printf("Date:   %s\n", commit.Timestamp.Format(time.RFC1123Z))
		fmt.Printf("\n    %s\n\n", commit.Message)

		if len(commit.Parents) > 0 {
			commitHash = commit.Parents[0] // just follow first parent for simple log
		} else {
			commitHash = ""
		}
	}

	return nil
}

func getCommitAncestors(repoRoot string, commitHash string) (map[string]bool, error) {
	ancestors := make(map[string]bool)
	queue := []string{commitHash}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if ancestors[curr] {
			continue
		}
		ancestors[curr] = true
		commit, err := ReadCommit(repoRoot, curr)
		if err == nil {
			for _, p := range commit.Parents {
				if p != "" {
					queue = append(queue, p)
				}
			}
		}
	}
	return ancestors, nil
}

func findCommonAncestor(repoRoot, hash1, hash2 string) (string, error) {
	ancestors1, err := getCommitAncestors(repoRoot, hash1)
	if err != nil {
		return "", err
	}

	queue := []string{hash2}
	visited := make(map[string]bool)
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr] {
			continue
		}
		visited[curr] = true

		if ancestors1[curr] {
			return curr, nil
		}

		commit, err := ReadCommit(repoRoot, curr)
		if err == nil {
			for _, p := range commit.Parents {
				if p != "" {
					queue = append(queue, p)
				}
			}
		}
	}

	return "", fmt.Errorf("no common ancestor found")
}

// Merge merges a target branch into the current branch.
func Merge(repoRoot, targetBranch string) error {
	headHash, err := GetHEADCommit(repoRoot)
	if err != nil {
		return err
	}

	targetHash, err := GetBranchCommit(repoRoot, targetBranch)
	if err != nil {
		return err
	}

	if headHash == targetHash {
		fmt.Println("Already up to date.")
		return nil
	}

	ancestor, err := findCommonAncestor(repoRoot, headHash, targetHash)
	if err != nil {
		ancestor = ""
	}

	if ancestor == headHash {
		// Fast-forward
		fmt.Println("Fast-forwarding...")
		if err := restoreFiles(repoRoot, headHash, targetHash); err != nil {
			return err
		}
		
		currBranch, _ := GetCurrentBranch(repoRoot)
		if currBranch != "" {
			if err := UpdateBranch(repoRoot, currBranch, targetHash); err != nil {
				return err
			}
		} else {
			headPath := filepath.Join(repoRoot, MgitDir, "HEAD")
			os.WriteFile(headPath, []byte(targetHash+"\n"), 0644)
		}
		return nil
	}

	if ancestor == targetHash {
		fmt.Println("Already up to date.")
		return nil
	}

	// True merge
	fmt.Println("Performing 3-way merge...")

	headCommit, err := ReadCommit(repoRoot, headHash)
	if err != nil {
		return err
	}

	targetCommit, err := ReadCommit(repoRoot, targetHash)
	if err != nil {
		return err
	}

	var ancestorFiles map[string]string
	if ancestor != "" {
		ancestorCommit, err := ReadCommit(repoRoot, ancestor)
		if err == nil {
			ancestorFiles = ancestorCommit.Files
		}
	}
	if ancestorFiles == nil {
		ancestorFiles = make(map[string]string)
	}

	mergedFiles := make(map[string]string)
	allFiles := make(map[string]bool)
	conflictedFiles := make(map[string]bool)
	hasConflicts := false
	for f := range headCommit.Files {
		allFiles[f] = true
	}
	for f := range targetCommit.Files {
		allFiles[f] = true
	}
	for f := range ancestorFiles {
		allFiles[f] = true
	}

	for f := range allFiles {
		hHash := headCommit.Files[f]
		tHash := targetCommit.Files[f]
		aHash := ancestorFiles[f]

		if hHash == tHash {
			if hHash != "" {
				mergedFiles[f] = hHash
			}
			continue
		}

		if hHash == aHash {
			// Changed in target
			if tHash != "" {
				mergedFiles[f] = tHash
			}
		} else if tHash == aHash {
			// Changed in head
			if hHash != "" {
				mergedFiles[f] = hHash
			}
		} else {
			// Conflict
			hasConflicts = true
			fmt.Printf("CONFLICT (content): Merge conflict in %s\n", f)

			var hContent, tContent []byte
			if hHash != "" {
				hContent, _ = os.ReadFile(filepath.Join(repoRoot, MgitDir, "objects", hHash))
			}
			if tHash != "" {
				tContent, _ = os.ReadFile(filepath.Join(repoRoot, MgitDir, "objects", tHash))
			}

			hStr := string(hContent)
			if len(hStr) > 0 && hStr[len(hStr)-1] != '\n' {
				hStr += "\n"
			}
			tStr := string(tContent)
			if len(tStr) > 0 && tStr[len(tStr)-1] != '\n' {
				tStr += "\n"
			}
			conflictContent := fmt.Sprintf("<<<<<<< HEAD\n%s=======\n%s>>>>>>> %s\n", hStr, tStr, targetBranch)
			absPath := filepath.Join(repoRoot, f)
			os.MkdirAll(filepath.Dir(absPath), 0755)
			os.WriteFile(absPath, []byte(conflictContent), 0644)

			if hHash != "" {
				mergedFiles[f] = hHash
			}
			conflictedFiles[f] = true
		}
	}

	// Write merged files to working tree and index
	idx := &Index{Entries: mergedFiles}
	if err := WriteIndex(repoRoot, idx); err != nil {
		return err
	}

	for f, blobHash := range mergedFiles {
		if conflictedFiles[f] {
			continue // Already written with markers
		}
		absPath := filepath.Join(repoRoot, f)
		os.MkdirAll(filepath.Dir(absPath), 0755)

		src, _ := os.Open(filepath.Join(repoRoot, MgitDir, "objects", blobHash))
		dst, _ := os.Create(absPath)
		io.Copy(dst, src)
		src.Close()
		dst.Close()
	}

	if hasConflicts {
		mergeHeadPath := filepath.Join(repoRoot, MgitDir, "MERGE_HEAD")
		os.WriteFile(mergeHeadPath, []byte(targetHash+"\n"), 0644)
		return fmt.Errorf("Automatic merge failed; fix conflicts and then commit the result")
	}

	msg := fmt.Sprintf("Merge branch '%s'", targetBranch)
	return CreateCommit(repoRoot, msg, []string{headHash, targetHash})
}

// Reset moves the current branch pointer and optionally updates the working tree and index.
func Reset(repoRoot, mode, target string) error {
	targetHash := target
	if utils.Exists(filepath.Join(repoRoot, MgitDir, "refs", "heads", target)) {
		targetHash, _ = GetBranchCommit(repoRoot, target)
	} else if !utils.Exists(filepath.Join(repoRoot, MgitDir, "objects", target)) {
		return fmt.Errorf("fatal: Could not parse object '%s'", target)
	}

	currentHash, err := GetHEADCommit(repoRoot)
	if err != nil {
		return err
	}

	if mode == "hard" {
		if err := restoreFiles(repoRoot, currentHash, targetHash); err != nil {
			return err
		}
	} else if mode == "soft" {
		// Do nothing to working tree or index
	} else {
		// Mixed (default) - Update index to match target, leave working tree
		targetCommit, err := ReadCommit(repoRoot, targetHash)
		if err != nil {
			return err
		}
		idx := &Index{Entries: targetCommit.Files}
		if err := WriteIndex(repoRoot, idx); err != nil {
			return err
		}
	}

	branch, _ := GetCurrentBranch(repoRoot)
	if branch != "" {
		if err := UpdateBranch(repoRoot, branch, targetHash); err != nil {
			return err
		}
	} else {
		headPath := filepath.Join(repoRoot, MgitDir, "HEAD")
		if err := os.WriteFile(headPath, []byte(targetHash+"\n"), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("HEAD is now at %s\n", targetHash[:7])
	return nil
}
