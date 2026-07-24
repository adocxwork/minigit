package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"mgit/pkg/utils"
)

type Commit struct {
	ID        string            `json:"id"`
	Parents   []string          `json:"parents,omitempty"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Files     map[string]string `json:"files"` // map of filepath to blob hash
}

// WriteBlob hashes the file, stores it in .mgit/objects, and returns the hash.
func WriteBlob(repoRoot, filePath string) (string, error) {
	hash, err := utils.HashFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	objectPath := filepath.Join(repoRoot, MgitDir, "objects", hash)
	if utils.Exists(objectPath) {
		return hash, nil // already exists
	}

	src, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(objectPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return hash, nil
}

// WriteCommit creates a commit object and stores it in .mgit/objects.
func WriteCommit(repoRoot string, commit Commit) (string, error) {
	data, err := json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return "", err
	}

	hash := utils.HashBytes(data)
	commit.ID = hash

	// Remarshal to include the ID inside the JSON for easier debugging
	data, err = json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return "", err
	}

	objectPath := filepath.Join(repoRoot, MgitDir, "objects", hash)
	if err := os.WriteFile(objectPath, data, 0644); err != nil {
		return "", err
	}

	return hash, nil
}

// ReadCommit reads and parses a commit object from .mgit/objects.
func ReadCommit(repoRoot, hash string) (*Commit, error) {
	objectPath := filepath.Join(repoRoot, MgitDir, "objects", hash)
	data, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, fmt.Errorf("commit not found: %s", hash)
	}

	var commit Commit
	if err := json.Unmarshal(data, &commit); err != nil {
		return nil, err
	}

	return &commit, nil
}
