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
func CreateCommit(repoRoot, message string) error {
	idx, err := ReadIndex(repoRoot)
	if err != nil {
		return err
	}

	if len(idx.Entries) == 0 {
		return fmt.Errorf("nothing to commit, working tree clean")
	}

	parentHash, err := GetHEADCommit(repoRoot)
	if err != nil {
		return err
	}

	commit := Commit{
		Parent:    parentHash,
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
	return nil
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

	commit, err := ReadCommit(repoRoot, commitHash)
	if err != nil {
		return err
	}

	// 1. Remove files from the current HEAD
	headCommitHash, _ := GetHEADCommit(repoRoot)
	if headCommitHash != "" {
		headCommit, err := ReadCommit(repoRoot, headCommitHash)
		if err == nil {
			for file := range headCommit.Files {
				os.Remove(filepath.Join(repoRoot, file))
				dir := filepath.Dir(filepath.Join(repoRoot, file))
				for dir != repoRoot {
					os.Remove(dir)
					dir = filepath.Dir(dir)
				}
			}
		}
	}

	// 2. Write files from target commit
	idx := &Index{Entries: make(map[string]string)}
	for relPath, blobHash := range commit.Files {
		absPath := filepath.Join(repoRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return err
		}

		blobPath := filepath.Join(repoRoot, MgitDir, "objects", blobHash)

		src, err := os.Open(blobPath)
		if err != nil {
			return err
		}

		dst, err := os.Create(absPath)
		if err != nil {
			src.Close()
			return err
		}

		io.Copy(dst, src)
		src.Close()
		dst.Close()

		idx.Entries[relPath] = blobHash
	}

	// 3. Update index to match the target commit
	if err := WriteIndex(repoRoot, idx); err != nil {
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

	idx, err := ReadIndex(repoRoot)
	if err != nil {
		return err
	}

	headCommitHash, _ := GetHEADCommit(repoRoot)
	headFiles := make(map[string]string)
	if headCommitHash != "" {
		commit, err := ReadCommit(repoRoot, headCommitHash)
		if err == nil {
			headFiles = commit.Files
		}
	}

	// Collect working directory files
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
		return err
	}

	var staged []string
	var modified []string
	var untracked []string

	// Check for untracked and modified
	for file, wdHash := range workDirFiles {
		idxHash, inIdx := idx.Entries[file]
		if !inIdx {
			untracked = append(untracked, file)
		} else if idxHash != wdHash {
			modified = append(modified, file)
		}
	}

	// Check for staged
	for file, idxHash := range idx.Entries {
		headHash, inHead := headFiles[file]
		if !inHead || headHash != idxHash {
			staged = append(staged, file)
		}
	}

	sort.Strings(staged)
	sort.Strings(modified)
	sort.Strings(untracked)

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

		commitHash = commit.Parent
	}

	return nil
}
