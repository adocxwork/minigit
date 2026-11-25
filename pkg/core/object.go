package core

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mgit/pkg/utils"
)

type Commit struct {
	ID        string            `json:"id"`
	Parents   []string          `json:"parents,omitempty"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Files     map[string]string `json:"files"`
}

func validObjectID(id string) bool {
	if len(id) != sha1.Size*2 || strings.ToLower(id) != id {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func validateSnapshot(files map[string]string) error {
	for path, hash := range files {
		if !validWorkPath(path) || !validObjectID(hash) {
			return fmt.Errorf("invalid snapshot entry: %q", path)
		}
		for parent := filepath.Dir(path); parent != "."; parent = filepath.Dir(parent) {
			if _, exists := files[parent]; exists {
				return fmt.Errorf("snapshot contains both file %q and its child %q", parent, path)
			}
		}
	}
	return nil
}

func storeBlob(meta *os.Root, src io.Reader) (string, error) {
	if info, err := checkedInfo(meta, "objects"); err != nil {
		return "", err
	} else if !info.IsDir() {
		return "", fmt.Errorf("invalid objects directory")
	}

	temp, dst, err := tempFile(meta, "objects")
	if err != nil {
		return "", err
	}
	defer meta.Remove(temp)
	hash := sha1.New()
	contentHash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(dst, hash, contentHash), src)
	if err := errors.Join(copyErr, dst.Close()); err != nil {
		return "", err
	}

	id := hex.EncodeToString(hash.Sum(nil))
	name := filepath.Join("objects", id)
	existing, err := openRegular(meta, name)
	if errors.Is(err, os.ErrNotExist) {
		return id, meta.Rename(temp, name)
	}
	if err != nil {
		return "", err
	}
	defer existing.Close()
	existingHash := sha256.New()
	if _, err := io.Copy(existingHash, existing); err != nil {
		return "", err
	}
	if !bytes.Equal(existingHash.Sum(nil), contentHash.Sum(nil)) {
		return "", fmt.Errorf("object %s already contains different data", id)
	}
	return id, nil
}

func copyBlob(meta *os.Root, hash string, dst io.Writer) error {
	if !validObjectID(hash) {
		return fmt.Errorf("invalid blob ID: %q", hash)
	}
	f, err := openRegular(meta, filepath.Join("objects", hash))
	if err != nil {
		return fmt.Errorf("read blob %s: %w", hash, err)
	}
	defer f.Close()
	sum := sha1.New()
	if _, err := io.Copy(io.MultiWriter(dst, sum), f); err != nil {
		return err
	}
	if hex.EncodeToString(sum.Sum(nil)) != hash {
		return fmt.Errorf("blob %s failed its checksum", hash)
	}
	return nil
}

func WriteCommit(repoRoot string, commit Commit) (string, error) {
	if err := validateSnapshot(commit.Files); err != nil {
		return "", err
	}
	for _, parent := range commit.Parents {
		if !validObjectID(parent) {
			return "", fmt.Errorf("invalid parent commit ID: %q", parent)
		}
	}
	commit.ID = ""
	data, err := json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return "", err
	}
	commit.ID = utils.HashBytes(data)
	data, err = json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return "", err
	}

	name := filepath.Join("objects", commit.ID)
	existing, err := readMetadata(repoRoot, name)
	if err == nil {
		if !bytes.Equal(existing, data) {
			return "", fmt.Errorf("object %s already contains different data", commit.ID)
		}
		return commit.ID, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return commit.ID, writeMetadata(repoRoot, name, data)
}

func ReadCommit(repoRoot, hash string) (*Commit, error) {
	if !validObjectID(hash) {
		return nil, fmt.Errorf("unknown branch or invalid commit ID: %q", hash)
	}
	data, err := readMetadata(repoRoot, filepath.Join("objects", hash))
	if err != nil {
		return nil, fmt.Errorf("read commit %s: %w", hash, err)
	}
	var commit Commit
	if err := json.Unmarshal(data, &commit); err != nil || commit.ID != hash {
		return nil, fmt.Errorf("object %s is not a valid commit", hash)
	}
	commit.ID = ""
	payload, err := json.MarshalIndent(commit, "", "  ")
	if err != nil || utils.HashBytes(payload) != hash {
		return nil, fmt.Errorf("commit %s failed its checksum", hash)
	}
	commit.ID = hash
	for _, parent := range commit.Parents {
		if !validObjectID(parent) {
			return nil, fmt.Errorf("invalid parent commit ID: %q", parent)
		}
	}
	return &commit, validateSnapshot(commit.Files)
}

func readHEAD(repoRoot string) (*Commit, error) {
	hash, err := GetHEADCommit(repoRoot)
	if err != nil {
		return nil, err
	}
	if hash == "" {
		return &Commit{Files: map[string]string{}}, nil
	}
	return ReadCommit(repoRoot, hash)
}
