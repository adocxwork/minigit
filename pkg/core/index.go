package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Index struct {
	Entries map[string]string `json:"entries"`
}

func ReadIndex(repoRoot string) (*Index, error) {
	idx := &Index{Entries: map[string]string{}}
	data, err := readMetadata(repoRoot, "index")
	if errors.Is(err, os.ErrNotExist) {
		return idx, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, idx); err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	if idx.Entries == nil {
		idx.Entries = map[string]string{}
	}
	return idx, validateSnapshot(idx.Entries)
}

func WriteIndex(repoRoot string, idx *Index) error {
	if err := validateSnapshot(idx.Entries); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return writeMetadata(repoRoot, "index", data)
}
