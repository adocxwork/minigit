package core

import (
	"encoding/json"
	"os"
	"path/filepath"

	"mgit/pkg/utils"
)

type Index struct {
	Entries map[string]string `json:"entries"` // filepath relative to repo root -> blob hash
}

// ReadIndex reads the staging area.
func ReadIndex(repoRoot string) (*Index, error) {
	indexPath := filepath.Join(repoRoot, MgitDir, "index")
	idx := &Index{Entries: make(map[string]string)}

	if !utils.Exists(indexPath) {
		return idx, nil
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, idx); err != nil {
		return nil, err
	}

	return idx, nil
}

// WriteIndex saves the staging area.
func WriteIndex(repoRoot string, idx *Index) error {
	indexPath := filepath.Join(repoRoot, MgitDir, "index")
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0644)
}
