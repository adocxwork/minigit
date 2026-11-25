package core

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const MgitDir = ".mgit"

func InitRepo() error {
	if err := os.Mkdir(MgitDir, 0755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("mgit repository already exists")
		}
		return err
	}
	initialized := false
	defer func() {
		if !initialized {
			os.RemoveAll(MgitDir)
		}
	}()

	for _, name := range []string{"objects", filepath.Join("refs", "heads")} {
		if err := os.MkdirAll(filepath.Join(MgitDir, name), 0755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(MgitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644); err != nil {
		return err
	}
	initialized = true
	fmt.Println("Initialized empty mgit repository in .mgit/")
	return nil
}

// GetRepoPath finds the repository root from the current directory.
func GetRepoPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Normalize a shell working directory that was entered through a symlink.
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}

	for {
		info, err := os.Lstat(filepath.Join(dir, MgitDir))
		switch {
		case err == nil && info.IsDir():
			return dir, nil
		case err == nil:
			return "", fmt.Errorf("%s must be a directory", filepath.Join(dir, MgitDir))
		case !errors.Is(err, os.ErrNotExist):
			return "", err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not an mgit repository")
		}
		dir = parent
	}
}

func openMetadata(repoRoot string) (*os.Root, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	info, err := root.Lstat(MgitDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s must be a directory", MgitDir)
	}
	return root.OpenRoot(MgitDir)
}

var ErrRepositoryBusy = errors.New("repository is busy")

// LockRepository coordinates operations between the CLI and UI.
func LockRepository(repoRoot string) (func(), error) {
	meta, err := openMetadata(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := meta.Mkdir("operation.lock", 0700); err != nil {
		meta.Close()
		if errors.Is(err, os.ErrExist) {
			return nil, ErrRepositoryBusy
		}
		return nil, err
	}
	return func() {
		defer meta.Close()
		meta.Remove("operation.lock")
	}, nil
}

func reservedPath(name string) bool {
	return strings.EqualFold(name, MgitDir) || strings.EqualFold(name, ".git")
}

func validWorkPath(name string) bool {
	if !filepath.IsLocal(name) || filepath.Clean(name) != name || name == "." ||
		!utf8.ValidString(name) {
		return false
	}
	for _, part := range strings.Split(name, string(filepath.Separator)) {
		if reservedPath(part) {
			return false
		}
	}
	return true
}

// checkedInfo rejects symlink parents instead of traversing them.
func checkedInfo(root *os.Root, name string) (os.FileInfo, error) {
	if parent := filepath.Dir(name); parent != "." {
		info, err := checkedInfo(root, parent)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, &os.PathError{Op: "inspect parent of", Path: name, Err: os.ErrNotExist}
		}
	}
	info, err := root.Lstat(name)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symbolic links are unsupported: %s", name)
	}
	return info, err
}

func openRegular(root *os.Root, name string) (*os.File, error) {
	info, err := checkedInfo(root, name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", name)
	}
	return root.Open(name)
}

func readRegular(root *os.Root, name string) ([]byte, error) {
	f, err := openRegular(root, name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func readMetadata(repoRoot, name string) ([]byte, error) {
	meta, err := openMetadata(repoRoot)
	if err != nil {
		return nil, err
	}
	defer meta.Close()
	return readRegular(meta, name)
}

func writeMetadata(repoRoot, name string, data []byte) error {
	meta, err := openMetadata(repoRoot)
	if err != nil {
		return err
	}
	defer meta.Close()
	if parent := filepath.Dir(name); parent != "." {
		info, err := checkedInfo(meta, parent)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", parent)
		}
	}
	return writeAtomic(meta, name, func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}

func tempFile(root *os.Root, dir string) (string, *os.File, error) {
	name := filepath.Join(dir, ".mgit-tmp-"+rand.Text())
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	return name, file, err
}

func writeAtomic(root *os.Root, name string, write func(*os.File) error) error {
	temp, file, err := tempFile(root, filepath.Dir(name))
	if err != nil {
		return err
	}
	defer root.Remove(temp)
	writeErr := write(file)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return root.Rename(temp, name)
}
