package core

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testRepo(t *testing.T) string {
	t.Helper()
	t.Chdir(t.TempDir())
	must(t, InitRepo())
	root, err := GetRepoPath()
	must(t, err)
	return root
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func put(t *testing.T, path, contents string) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0755))
	must(t, os.WriteFile(path, []byte(contents), 0644))
}

func fileIs(t *testing.T, name, want string) {
	t.Helper()
	actual, err := os.ReadFile(name)
	must(t, err)
	if string(actual) != want {
		t.Fatalf("%s = %q, want %q", name, actual, want)
	}
}

func absent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %q to be missing, got %v", path, err)
	}
}

func failure(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), text) {
		t.Fatalf("expected error containing %q, got %v", text, err)
	}
}

func commitAll(t *testing.T, root, message string) string {
	t.Helper()
	must(t, AddPaths(root, []string{"."}))
	must(t, CreateCommit(root, message, nil))
	head, err := GetHEADCommit(root)
	must(t, err)
	return head
}

func TestStageDeletionAndEmptySnapshot(t *testing.T) {
	root := testRepo(t)
	put(t, "note.txt", "initial")
	put(t, ".git/private", "exclude")
	base := commitAll(t, root, "initial")
	failure(t, CreateCommit(root, "unchanged", nil), "nothing staged")
	must(t, os.Remove("note.txt"))

	staged, modified, untracked, err := GetStatus(root)
	must(t, err)
	if len(staged) != 0 || len(untracked) != 0 || !reflect.DeepEqual(modified, []string{"note.txt"}) {
		t.Fatalf("status before staging deletion: %v %v %v", staged, modified, untracked)
	}
	must(t, AddPaths(root, []string{"note.txt"}))
	staged, modified, _, err = GetStatus(root)
	must(t, err)
	if len(modified) != 0 || !reflect.DeepEqual(staged, []string{"note.txt"}) {
		t.Fatalf("status after staging deletion: %v %v", staged, modified)
	}
	must(t, CreateCommit(root, "delete last file", nil))
	history, err := History(root)
	must(t, err)
	if len(history) != 2 || len(history[0].Files) != 0 || history[0].Parents[0] != base {
		t.Fatalf("unexpected history: %+v", history)
	}
}

func TestCheckoutPreservesLocalFiles(t *testing.T) {
	root := testRepo(t)
	put(t, "shared", "before")
	base := commitAll(t, root, "initial")
	must(t, CreateBranch(root, "other", base))
	must(t, Checkout(root, "other"))
	put(t, "shared", "after")
	put(t, "collision", "incoming")
	commitAll(t, root, "other")
	must(t, Checkout(root, "main"))
	put(t, "collision", "untracked")
	failure(t, Checkout(root, "other"), "untracked")
	fileIs(t, "shared", "before")
	fileIs(t, "collision", "untracked")
	must(t, os.Remove("collision"))
	must(t, os.Remove("shared"))
	failure(t, Checkout(root, "other"), "working tree is not clean")
	absent(t, "shared")
	must(t, Reset(root, "hard", base))
	must(t, Checkout(root, "other"))
	fileIs(t, "shared", "after")
	must(t, Checkout(root, base))
	if branch, err := GetCurrentBranch(root); branch != "" || err != nil {
		t.Fatalf("expected detached HEAD, got %q %v", branch, err)
	}
}

func TestCheckoutFileDirectoryTransition(t *testing.T) {
	root := testRepo(t)
	put(t, "item", "base")
	base := commitAll(t, root, "file")
	must(t, CreateBranch(root, "base", base))
	must(t, os.Remove("item"))
	child := filepath.Join("item", "child")
	put(t, child, "child")
	commitAll(t, root, "directory")
	extra := filepath.Join("item", "extra")
	put(t, extra, "local")
	failure(t, Checkout(root, "base"), "untracked")
	fileIs(t, extra, "local")
	fileIs(t, child, "child")
	must(t, os.Remove(extra))
	must(t, Checkout(root, "base"))
	fileIs(t, "item", "base")
	must(t, Checkout(root, "main"))
	fileIs(t, child, "child")
}

func TestResetModesAndBlobValidation(t *testing.T) {
	root := testRepo(t)
	put(t, "file", "first")
	base := commitAll(t, root, "base")
	put(t, "file", "second")
	latest := commitAll(t, root, "latest")
	put(t, "file", "local edit")
	put(t, "staged-only", "staged")
	put(t, "local-only", "untracked")
	must(t, AddPaths(root, []string{"staged-only"}))
	failure(t, Reset(root, "unknown", base), "unknown reset mode")
	must(t, Reset(root, "soft", base))
	idx, err := ReadIndex(root)
	must(t, err)
	failure(t, Reset(root, "soft", idx.Entries["file"]), "valid commit")
	fileIs(t, "file", "local edit")
	must(t, Reset(root, "hard", base))
	fileIs(t, "file", "first")
	fileIs(t, "local-only", "untracked")
	absent(t, "staged-only")
	must(t, Reset(root, "mixed", latest))
	fileIs(t, "file", "first")
	idx, err = ReadIndex(root)
	must(t, err)
	commit, err := ReadCommit(root, latest)
	must(t, err)
	if idx.Entries["file"] != commit.Files["file"] {
		t.Fatal("mixed reset did not update the index")
	}
	put(t, filepath.Join(MgitDir, "objects", commit.Files["file"]), "damaged")
	failure(t, Reset(root, "hard", latest), "checksum")
	fileIs(t, "file", "first")
}

func TestThreeWayMergeAppliesDeletionsAndTracksConflicts(t *testing.T) {
	root := testRepo(t)
	put(t, "conflict", "base")
	put(t, "delete", "base")
	base := commitAll(t, root, "base")
	must(t, CreateBranch(root, "topic", base))
	put(t, "conflict", "ours")
	ours := commitAll(t, root, "ours")
	must(t, Checkout(root, "topic"))
	put(t, "conflict", "theirs")
	put(t, "topic-file", "incoming")
	must(t, os.Remove("delete"))
	theirs := commitAll(t, root, "theirs")
	must(t, Checkout(root, "main"))
	failure(t, Merge(root, "topic"), "merge conflicts")
	absent(t, "delete")
	fileIs(t, "topic-file", "incoming")
	fileIs(t, "conflict", "<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> topic\n")
	failure(t, CreateCommit(root, "premature", nil), "resolve and stage")
	failure(t, AddPaths(root, []string{"conflict"}), "resolve the conflict")
	failure(t, Checkout(root, "topic"), "finish the merge")
	put(t, "conflict", "resolved")
	must(t, AddPaths(root, []string{"conflict"}))
	must(t, CreateCommit(root, "merge topic", nil))
	history, err := History(root)
	must(t, err)
	if !reflect.DeepEqual(history[0].Parents, []string{ours, theirs}) {
		t.Fatalf("merge parents: %v", history[0].Parents)
	}
	must(t, Merge(root, "topic"))
}

func TestFastForwardAndCleanMerge(t *testing.T) {
	root := testRepo(t)
	put(t, "a", "base")
	base := commitAll(t, root, "base")
	must(t, CreateBranch(root, "topic", base))
	must(t, Checkout(root, "topic"))
	put(t, "b", "topic")
	tip := commitAll(t, root, "topic addition")
	must(t, Checkout(root, "main"))
	must(t, Merge(root, "topic"))
	if head, err := GetHEADCommit(root); head != tip || err != nil {
		t.Fatalf("fast-forward yielded %q %v", head, err)
	}
	put(t, "c", "main")
	ours := commitAll(t, root, "main addition")
	must(t, Checkout(root, "topic"))
	must(t, os.Remove("a"))
	theirs := commitAll(t, root, "topic deletion")
	must(t, Checkout(root, "main"))
	must(t, Merge(root, "topic"))
	absent(t, "a")
	fileIs(t, "b", "topic")
	fileIs(t, "c", "main")
	history, err := History(root)
	must(t, err)
	if !reflect.DeepEqual(history[0].Parents, []string{ours, theirs}) {
		t.Fatalf("merge parents: %v", history[0].Parents)
	}
}

func TestStagePathsCannotLeaveRepositoryOrPartiallyUpdateIndex(t *testing.T) {
	root := testRepo(t)
	put(t, "inside", "inside")
	outside := filepath.Join(t.TempDir(), "outside")
	put(t, outside, "outside")
	failure(t, AddPaths(root, []string{"inside", outside}), "inside the repository")
	failure(t, AddPaths(root, []string{".mgit/HEAD"}), "invalid repository path")
	idx, err := ReadIndex(root)
	must(t, err)
	if len(idx.Entries) != 0 {
		t.Fatal("failed batch staged some entries")
	}
	link := "symbolic-link"
	if err := os.Symlink(filepath.Dir(outside), link); err != nil {
		t.Skipf("environment cannot create symlinks: %v", err)
	}
	failure(t, AddPaths(root, []string{filepath.Join(link, filepath.Base(outside))}), "symbolic links")
	must(t, AddPaths(root, []string{"."}))
	idx, err = ReadIndex(root)
	must(t, err)
	if len(idx.Entries) != 1 || idx.Entries["inside"] == "" {
		t.Fatalf("unexpected index: %+v", idx.Entries)
	}
}

func TestWorktreeRollbackIfMetadataUpdateFails(t *testing.T) {
	root := testRepo(t)
	put(t, "a", "before")
	base := commitAll(t, root, "base")
	must(t, CreateBranch(root, "base", base))
	must(t, os.Remove("a"))
	put(t, filepath.Join("a", "child"), "after")
	next := commitAll(t, root, "replacement")
	must(t, Checkout(root, "base"))
	old, err := ReadCommit(root, base)
	must(t, err)
	target, err := ReadCommit(root, next)
	must(t, err)
	expected := errors.New("simulated metadata write failure")
	err = updateWorktree(root, old.Files, target.Files, target.Files, false, func() error { return expected })
	if !errors.Is(err, expected) {
		t.Fatalf("unexpected error: %v", err)
	}
	fileIs(t, "a", "before")
	idx, err := ReadIndex(root)
	must(t, err)
	if !reflect.DeepEqual(idx.Entries, old.Files) {
		t.Fatalf("rollback did not restore index: %v", idx.Entries)
	}
}

func TestLockCoordinatesRepositoryOperations(t *testing.T) {
	root := testRepo(t)
	unlock, err := LockRepository(root)
	must(t, err)
	if release, err := LockRepository(root); !errors.Is(err, ErrRepositoryBusy) {
		if release != nil {
			release()
		}
		t.Fatalf("expected repository busy, got %v", err)
	}
	unlock()
	unlock, err = LockRepository(root)
	must(t, err)
	unlock()
}

func TestHardResetAbandonsConflictWithDeletedCurrentVersion(t *testing.T) {
	root := testRepo(t)
	put(t, "file", "base")
	base := commitAll(t, root, "base")
	must(t, CreateBranch(root, "topic", base))
	must(t, os.Remove("file"))
	ours := commitAll(t, root, "remove file")
	must(t, Checkout(root, "topic"))
	put(t, "file", "incoming")
	commitAll(t, root, "edit file")
	must(t, Checkout(root, "main"))
	failure(t, Merge(root, "topic"), "merge conflicts")
	put(t, "local-only", "keep")
	failure(t, Reset(root, "mixed", ours), "finish the merge")
	must(t, Reset(root, "hard", ours))
	absent(t, "file")
	fileIs(t, "local-only", "keep")
	merging, conflicts, err := MergeStatus(root)
	must(t, err)
	if merging || len(conflicts) > 0 {
		t.Fatalf("merge state remains after reset: %v %v", merging, conflicts)
	}
}
