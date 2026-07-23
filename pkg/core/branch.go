package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mgit/pkg/utils"
)

// GetCurrentBranch returns the current branch name, or empty if detached HEAD.
func GetCurrentBranch(repoRoot string) (string, error) {
	headPath := filepath.Join(repoRoot, MgitDir, "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(data))

	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/"), nil
	}
	return "", nil // Detached HEAD
}

// GetBranchCommit returns the commit hash a branch points to.
func GetBranchCommit(repoRoot, branchName string) (string, error) {
	branchPath := filepath.Join(repoRoot, MgitDir, "refs", "heads", branchName)
	if !utils.Exists(branchPath) {
		return "", fmt.Errorf("branch '%s' not found", branchName)
	}

	data, err := os.ReadFile(branchPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// UpdateBranch updates a branch to point to a specific commit.
func UpdateBranch(repoRoot, branchName, commitHash string) error {
	branchPath := filepath.Join(repoRoot, MgitDir, "refs", "heads", branchName)
	return os.WriteFile(branchPath, []byte(commitHash+"\n"), 0644)
}

// ListBranches returns a list of all branches.
func ListBranches(repoRoot string) ([]string, error) {
	headsDir := filepath.Join(repoRoot, MgitDir, "refs", "heads")
	entries, err := os.ReadDir(headsDir)
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, entry := range entries {
		if !entry.IsDir() {
			branches = append(branches, entry.Name())
		}
	}
	return branches, nil
}

// CreateBranch creates a new branch pointing to a specific commit.
func CreateBranch(repoRoot, branchName, commitHash string) error {
	branchPath := filepath.Join(repoRoot, MgitDir, "refs", "heads", branchName)
	if utils.Exists(branchPath) {
		return fmt.Errorf("branch '%s' already exists", branchName)
	}
	return os.WriteFile(branchPath, []byte(commitHash+"\n"), 0644)
}

// UpdateHEAD updates the HEAD file to point to a new reference (branch).
func UpdateHEAD(repoRoot, branchName string) error {
	headPath := filepath.Join(repoRoot, MgitDir, "HEAD")
	content := fmt.Sprintf("ref: refs/heads/%s\n", branchName)
	return os.WriteFile(headPath, []byte(content), 0644)
}

// GetHEADCommit returns the current commit hash HEAD points to.
func GetHEADCommit(repoRoot string) (string, error) {
	branch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		return "", err
	}

	if branch != "" {
		commit, err := GetBranchCommit(repoRoot, branch)
		if err != nil {
			// Initial commit scenario: branch exists in HEAD but file doesn't exist yet
			return "", nil
		}
		return commit, nil
	}

	// Detached HEAD scenario (HEAD contains just the hash)
	headPath := filepath.Join(repoRoot, MgitDir, "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
