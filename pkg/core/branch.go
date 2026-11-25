package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const branchPrefix = "ref: refs/heads/"

func GetCurrentBranch(repoRoot string) (string, error) {
	data, err := readMetadata(repoRoot, "HEAD")
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(data))
	if branch, ok := strings.CutPrefix(content, branchPrefix); ok {
		if !IsValidRefName(branch) {
			return "", fmt.Errorf("invalid branch in HEAD: %q", branch)
		}
		return branch, nil
	}
	if !validObjectID(content) {
		return "", fmt.Errorf("invalid HEAD: %q", content)
	}
	return "", nil
}

func GetBranchCommit(repoRoot, branchName string) (string, error) {
	if !IsValidRefName(branchName) {
		return "", fmt.Errorf("invalid branch name: %q", branchName)
	}
	data, err := readMetadata(repoRoot, filepath.Join("refs", "heads", branchName))
	if err != nil {
		return "", err
	}
	hash := strings.TrimSpace(string(data))
	if !validObjectID(hash) {
		return "", fmt.Errorf("invalid commit in branch %q", branchName)
	}
	return hash, nil
}

func UpdateBranch(repoRoot, branchName, commitHash string) error {
	if !IsValidRefName(branchName) || !validObjectID(commitHash) {
		return fmt.Errorf("invalid branch name or commit ID")
	}
	return writeMetadata(repoRoot, filepath.Join("refs", "heads", branchName), []byte(commitHash+"\n"))
}

func ListBranches(repoRoot string) ([]string, error) {
	meta, err := openMetadata(repoRoot)
	if err != nil {
		return nil, err
	}
	defer meta.Close()

	dir := filepath.Join("refs", "heads")
	if _, err := checkedInfo(meta, dir); err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(meta.FS(), filepath.ToSlash(dir))
	if err != nil {
		return nil, err
	}
	branches := []string{}
	for _, entry := range entries {
		if entry.Type().IsRegular() && IsValidRefName(entry.Name()) {
			branches = append(branches, entry.Name())
		}
	}
	// Include the current unborn branch so the UI can display it.
	current, err := GetCurrentBranch(repoRoot)
	if err != nil {
		return nil, err
	}
	if current != "" {
		if _, err := GetBranchCommit(repoRoot, current); errors.Is(err, os.ErrNotExist) {
			branches = append(branches, current)
		} else if err != nil {
			return nil, err
		}
	}
	return branches, nil
}

// IsValidRefName validates a single branch name or full commit ID.
func IsValidRefName(name string) bool {
	return name != "" && utf8.ValidString(name) && filepath.IsLocal(name) &&
		!strings.HasPrefix(name, ".") && !strings.HasSuffix(name, ".") &&
		!strings.HasSuffix(name, ".lock") && !strings.Contains(name, "..") &&
		!strings.ContainsAny(name, "/\\~^:?*[]<>|\"") &&
		strings.IndexFunc(name, unicode.IsSpace) == -1 &&
		strings.IndexFunc(name, unicode.IsControl) == -1
}

func CreateBranch(repoRoot, branchName, commitHash string) error {
	if !IsValidRefName(branchName) {
		return fmt.Errorf("invalid branch name: %q", branchName)
	}
	if commitHash == "" {
		return fmt.Errorf("create a commit before creating a branch")
	}
	if _, err := ReadCommit(repoRoot, commitHash); err != nil {
		return err
	}
	_, err := GetBranchCommit(repoRoot, branchName)
	if !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return err
		}
		return fmt.Errorf("branch %q already exists", branchName)
	}
	return UpdateBranch(repoRoot, branchName, commitHash)
}

func UpdateHEAD(repoRoot, branchName string) error {
	if !IsValidRefName(branchName) {
		return fmt.Errorf("invalid branch name: %q", branchName)
	}
	return writeMetadata(repoRoot, "HEAD", []byte(branchPrefix+branchName+"\n"))
}

func GetHEADCommit(repoRoot string) (string, error) {
	branch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		return "", err
	}
	if branch == "" {
		data, err := readMetadata(repoRoot, "HEAD")
		return strings.TrimSpace(string(data)), err
	}
	hash, err := GetBranchCommit(repoRoot, branch)
	if errors.Is(err, os.ErrNotExist) {
		// Only a missing branch file in an intact refs directory is an unborn branch.
		meta, openErr := openMetadata(repoRoot)
		if openErr != nil {
			return "", openErr
		}
		defer meta.Close()
		info, dirErr := checkedInfo(meta, filepath.Join("refs", "heads"))
		if dirErr != nil {
			return "", dirErr
		}
		if !info.IsDir() {
			return "", fmt.Errorf("invalid branch directory")
		}
		return "", nil
	}
	return hash, err
}

func moveHEAD(repoRoot, hash string) error {
	branch, err := GetCurrentBranch(repoRoot)
	if err != nil {
		return err
	}
	if branch != "" {
		return UpdateBranch(repoRoot, branch, hash)
	}
	return writeMetadata(repoRoot, "HEAD", []byte(hash+"\n"))
}

func resolveCommit(repoRoot, target string) (*Commit, bool, error) {
	if !IsValidRefName(target) {
		return nil, false, fmt.Errorf("invalid reference name: %q", target)
	}
	hash, err := GetBranchCommit(repoRoot, target)
	if err == nil {
		commit, err := ReadCommit(repoRoot, hash)
		return commit, true, err
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	commit, err := ReadCommit(repoRoot, target)
	return commit, false, err
}
