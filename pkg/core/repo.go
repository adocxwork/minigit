package core

import (
	"fmt"
	"os"
	"path/filepath"

	"mgit/pkg/utils"
)

const (
	MgitDir = ".mgit"
)

// InitRepo initializes a new mgit repository in the current directory.
func InitRepo() error {
	if utils.Exists(MgitDir) {
		return fmt.Errorf("mgit repository already exists")
	}

	dirsToCreate := []string{
		MgitDir,
		filepath.Join(MgitDir, "objects"),
		filepath.Join(MgitDir, "refs", "heads"),
	}

	for _, dir := range dirsToCreate {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create HEAD file pointing to main branch
	headPath := filepath.Join(MgitDir, "HEAD")
	if err := os.WriteFile(headPath, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		return fmt.Errorf("failed to create HEAD file: %w", err)
	}

	fmt.Println("Initialized empty mgit repository in .mgit/")
	return nil
}

// GetRepoPath returns the path to the .mgit directory, searching upwards if necessary.
func GetRepoPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		mgitPath := filepath.Join(dir, MgitDir)
		if utils.Exists(mgitPath) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not an mgit repository (or any of the parent directories): %s", MgitDir)
}
