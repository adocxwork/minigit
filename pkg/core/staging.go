package core

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

func walkWorktree(root *os.Root, start string, visit func(string, fs.DirEntry) error) error {
	return fs.WalkDir(root.FS(), filepath.ToSlash(start), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		path = filepath.FromSlash(path)
		if reservedPath(entry.Name()) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		return visit(path, entry)
	})
}

// Add resolves command-line paths relative to the caller's directory.
func Add(repoRoot, path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		cwd, err = filepath.EvalSymlinks(cwd)
		if err != nil {
			return err
		}
		path = filepath.Join(cwd, path)
	}
	relative, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return err
	}
	return AddPaths(repoRoot, []string{relative})
}

// AddPaths stages repository-relative paths together, including tracked deletions.
func AddPaths(repoRoot string, paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("at least one path is required")
	}

	idx, err := ReadIndex(repoRoot)
	if err != nil {
		return err
	}
	merge, err := readMerge(repoRoot)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	meta, err := openMetadata(repoRoot)
	if err != nil {
		return err
	}
	defer meta.Close()

	for _, path := range paths {
		if path == "" || !filepath.IsLocal(path) {
			return fmt.Errorf("path must be inside the repository: %q", path)
		}
		path = filepath.Clean(path)
		if path != "." && !validWorkPath(path) {
			return fmt.Errorf("invalid repository path: %q", path)
		}
		if err := stagePath(root, meta, idx.Entries, merge.Conflicts, path); err != nil {
			return err
		}
	}

	if merge.Target == "" {
		return WriteIndex(repoRoot, idx)
	}
	return updateIndex(repoRoot, idx.Entries, func() error {
		return writeMerge(repoRoot, merge)
	})
}

func pathWithin(path, parent string) bool {
	return parent == "." || path == parent || strings.HasPrefix(path, parent+string(filepath.Separator))
}

func stagePath(root, meta *os.Root, entries, conflicts map[string]string, path string) error {
	info, err := checkedInfo(root, path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	staged := map[string]string{}
	if info != nil {
		err := walkWorktree(root, path, func(name string, entry fs.DirEntry) error {
			if entry.IsDir() {
				return nil
			}
			if !validWorkPath(name) {
				return fmt.Errorf("invalid repository path: %q", name)
			}
			if !entry.Type().IsRegular() && name != path && entries[name] == "" {
				if _, conflict := conflicts[name]; !conflict {
					return nil
				}
			}
			src, err := openRegular(root, name)
			if err != nil {
				return err
			}
			defer src.Close()
			id, err := storeBlob(meta, src)
			if err != nil {
				return err
			}
			if id == conflicts[name] {
				return fmt.Errorf("resolve the conflict in %q before staging", name)
			}
			staged[name] = id
			return nil
		})
		if err != nil {
			return err
		}
	}

	matched := info != nil
	for name := range entries {
		if pathWithin(name, path) {
			delete(entries, name)
			matched = true
		}
	}
	for name := range conflicts {
		if pathWithin(name, path) {
			delete(conflicts, name)
			matched = true
		}
	}
	if !matched {
		return fmt.Errorf("path %q did not match any files", path)
	}
	for name, hash := range staged {
		for parent := filepath.Dir(name); parent != "."; parent = filepath.Dir(parent) {
			delete(entries, parent)
		}
		entries[name] = hash
	}
	return nil
}

// GetStatus compares the committed snapshot, index and working tree.
func GetStatus(repoRoot string) (staged, modified, untracked []string, err error) {
	idx, err := ReadIndex(repoRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	head, err := readHEAD(repoRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	defer root.Close()

	staged, modified, untracked = []string{}, []string{}, []string{}
	for name, hash := range idx.Entries {
		if head.Files[name] != hash {
			staged = append(staged, name)
		}
	}
	for name := range head.Files {
		if idx.Entries[name] == "" {
			staged = append(staged, name)
		}
	}

	seen := map[string]bool{}
	err = walkWorktree(root, ".", func(name string, entry fs.DirEntry) error {
		if entry.IsDir() {
			return nil
		}
		indexHash := idx.Entries[name]
		if indexHash == "" {
			if entry.Type().IsRegular() {
				untracked = append(untracked, name)
			}
			return nil
		}
		seen[name] = true
		if !entry.Type().IsRegular() {
			modified = append(modified, name)
			return nil
		}

		file, err := openRegular(root, name)
		if err != nil {
			return err
		}
		defer file.Close()
		hash := sha1.New()
		if _, err := io.Copy(hash, file); err != nil {
			return err
		}
		if hex.EncodeToString(hash.Sum(nil)) != indexHash {
			modified = append(modified, name)
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	for name := range idx.Entries {
		if !seen[name] {
			modified = append(modified, name)
		}
	}
	sort.Strings(staged)
	sort.Strings(modified)
	sort.Strings(untracked)
	return staged, modified, untracked, nil
}

func CreateCommit(repoRoot, message string, parents []string) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("commit message is required")
	}
	head, err := readHEAD(repoRoot)
	if err != nil {
		return err
	}
	idx, err := ReadIndex(repoRoot)
	if err != nil {
		return err
	}
	merge, err := readMerge(repoRoot)
	if err != nil {
		return err
	}
	if len(merge.Conflicts) > 0 {
		return fmt.Errorf("resolve and stage all merge conflicts before committing")
	}

	if len(parents) == 0 && head.ID != "" {
		parents = []string{head.ID}
	}
	if merge.Target != "" && !slices.Contains(parents, merge.Target) {
		if _, err := ReadCommit(repoRoot, merge.Target); err != nil {
			return err
		}
		parents = append(parents, merge.Target)
	}
	if maps.Equal(head.Files, idx.Entries) && len(parents) < 2 {
		return fmt.Errorf("nothing staged to commit")
	}
	return finishMerge(repoRoot, merge, func() error {
		return saveCommit(repoRoot, message, parents, idx.Entries)
	})
}
