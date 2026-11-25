package core

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

func updateIndex(repoRoot string, entries map[string]string, finalize func() error) error {
	previous, err := ReadIndex(repoRoot)
	if err != nil {
		return err
	}
	if err := WriteIndex(repoRoot, &Index{Entries: entries}); err != nil {
		return err
	}
	if err := finalize(); err != nil {
		return errors.Join(err, WriteIndex(repoRoot, previous))
	}
	return nil
}

// updateWorktree checks all destinations and stages verified blobs before
// moving files. Backups let ordinary filesystem errors roll back the update.
func updateWorktree(repoRoot string, old, working, index map[string]string, force bool, finalize func() error) (result error) {
	if err := validateSnapshot(working); err != nil {
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

	backups := map[string]os.FileInfo{}
	for name, hash := range old {
		if !force && hash == working[name] {
			continue
		}
		info, err := checkedInfo(root, name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			backups[name] = info
		} else if !info.IsDir() {
			return fmt.Errorf("refusing to replace special file %q", name)
		}
	}

	writes := []string{}
	dirsToRemove := []string{}
	for name, hash := range working {
		if !force && hash == old[name] {
			continue
		}
		dirs, err := checkDestination(root, name, old, backups)
		if err != nil {
			return err
		}
		dirsToRemove = append(dirsToRemove, dirs...)
		writes = append(writes, name)
	}
	if len(writes)+len(backups) == 0 {
		return updateIndex(repoRoot, index, finalize)
	}
	sort.Strings(writes)
	sort.Sort(sort.Reverse(sort.StringSlice(dirsToRemove)))

	temp := filepath.Join(MgitDir, "worktree-"+rand.Text())
	if err := root.Mkdir(temp, 0700); err != nil {
		return err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			root.RemoveAll(temp)
		}
	}()

	copies := map[string]string{}
	for _, name := range writes {
		fileName, file, err := tempFile(root, temp)
		if err != nil {
			return err
		}
		copyErr := copyBlob(meta, working[name], file)
		if info := backups[name]; copyErr == nil && info != nil {
			copyErr = file.Chmod(info.Mode().Perm())
		}
		if err := errors.Join(copyErr, file.Close()); err != nil {
			return err
		}
		copies[name] = fileName
	}

	saved := map[string]string{}
	removedDirs := []string{}
	createdDirs := []string{}
	installed := []string{}
	defer func() {
		if result == nil {
			return
		}
		var rollback error
		for _, name := range slices.Backward(installed) {
			rollback = errors.Join(rollback, root.Remove(name))
		}
		for _, dir := range slices.Backward(createdDirs) {
			rollback = errors.Join(rollback, root.Remove(dir))
		}
		for _, dir := range slices.Backward(removedDirs) {
			rollback = errors.Join(rollback, root.Mkdir(dir, 0755))
		}
		for name, backup := range saved {
			rollback = errors.Join(rollback, root.Rename(backup, name))
		}
		if rollback != nil {
			keepBackup = true
			result = errors.Join(result, fmt.Errorf("rollback failed; backups are in %s: %w", temp, rollback))
		}
	}()

	for name := range backups {
		backup := filepath.Join(temp, "original", name)
		if err := root.MkdirAll(filepath.Dir(backup), 0700); err != nil {
			return err
		}
		if err := root.Rename(name, backup); err != nil {
			return err
		}
		saved[name] = backup
	}
	for _, dir := range dirsToRemove {
		if err := root.Remove(dir); err != nil {
			return err
		}
		removedDirs = append(removedDirs, dir)
	}
	for _, name := range writes {
		if err := createParents(root, filepath.Dir(name), &createdDirs); err != nil {
			return err
		}
		if _, err := root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
			if err != nil {
				return err
			}
			return fmt.Errorf("destination already exists: %q", name)
		}
		if err := root.Rename(copies[name], name); err != nil {
			return err
		}
		installed = append(installed, name)
	}
	return updateIndex(repoRoot, index, finalize)
}

func checkDestination(root *os.Root, name string, old map[string]string, backups map[string]os.FileInfo) ([]string, error) {
	for parent := filepath.Dir(name); parent != "."; parent = filepath.Dir(parent) {
		info, err := checkedInfo(root, parent)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() && backups[parent] == nil {
			return nil, fmt.Errorf("untracked file %q blocks %q", parent, name)
		}
	}
	info, err := checkedInfo(root, name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if backups[name] != nil {
		return nil, nil
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("would overwrite untracked path %q", name)
	}
	hasTrackedChild := false
	for path := range old {
		if strings.HasPrefix(path, name+string(filepath.Separator)) {
			hasTrackedChild = true
			break
		}
	}
	if !hasTrackedChild {
		return nil, fmt.Errorf("untracked directory %q blocks the update", name)
	}
	dirs := []string{}
	err = fs.WalkDir(root.FS(), filepath.ToSlash(name), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		path = filepath.FromSlash(path)
		if entry.IsDir() {
			dirs = append(dirs, path)
		} else if backups[path] == nil {
			return fmt.Errorf("untracked path %q blocks the update", path)
		}
		return nil
	})
	return dirs, err
}

func createParents(root *os.Root, dir string, created *[]string) error {
	if dir == "." {
		return nil
	}
	if err := createParents(root, filepath.Dir(dir), created); err != nil {
		return err
	}

	info, err := checkedInfo(root, dir)
	if errors.Is(err, os.ErrNotExist) {
		if err := root.Mkdir(dir, 0755); err != nil {
			return err
		}
		*created = append(*created, dir)
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", dir)
	}
	return nil
}
