# mgit - Project Documentation

## Overview
`mgit` is a simplified, Git-like version control system (VCS) written in Go. This project was built to demonstrate the fundamental concepts of version control, such as content-addressable storage, commit history, and branch management, without the overwhelming complexity of the real Git tool.

## Architecture

The project is structured into three main layers:
1. **CLI Layer (`cmd/`)**: Uses `github.com/spf13/cobra` to handle user input and command parsing.
2. **Core Logic (`pkg/core/`)**: Contains the business logic of the version control system (commits, indexing, branching, etc.).
3. **Utilities (`pkg/utils/`)**: Contains helper functions for cryptography and file I/O operations.

## Data Model & Storage

All repository data is stored entirely within a hidden `.mgit` directory located at the root of an initialized repository. The storage mechanism relies on a simplified content-addressable model.

### 1. Objects (`.mgit/objects/`)
Objects are the core data units in `mgit`. There are two types of objects:
* **Blobs**: When a file is added to the staging area (`mgit add`), its contents are hashed using SHA-1. A file is created in the `objects` directory named after this hash, containing the exact contents of the file.
* **Commits**: When a commit is created (`mgit commit`), a JSON object is generated containing:
  * `ID`: The SHA-1 hash of the commit JSON.
  * `Parent`: The hash of the previous commit (if any).
  * `Message`: The commit message.
  * `Timestamp`: The time the commit was created.
  * `Files`: A snapshot of the repository, represented as a map of file paths to their respective blob hashes.
This JSON is also hashed and stored in the `objects` directory.

### 2. The Index (`.mgit/index`)
The index serves as the "staging area". It is a JSON file that maintains a mapping of file paths to the SHA-1 hash of their staged contents. 
When `mgit add` is run, the file is hashed, copied to `objects/`, and the `index` is updated. 
When `mgit status` is run, the index is compared against the working directory (to find modified/untracked files) and the HEAD commit (to find staged files).

### 3. References (`.mgit/refs/heads/`)
Branches in `mgit` are extremely lightweight. A branch is simply a text file located in `refs/heads/`. The name of the file is the branch name (e.g., `main`, `feature`), and the contents of the file is a single string: the SHA-1 hash of the latest commit on that branch.

### 4. HEAD (`.mgit/HEAD`)
The `HEAD` file determines what is currently checked out in the working directory. It can be in one of two states:
* **Attached**: It contains a reference to a branch, e.g., `ref: refs/heads/main`.
* **Detached**: It directly contains a commit hash instead of a branch reference. This happens if you checkout a specific commit hash instead of a branch.

## Command Implementations

### `init`
Creates the `.mgit` folder structure and a default `HEAD` file pointing to `refs/heads/main`.

### `add`
Recursively walks through the provided path. For each file, it calculates the SHA-1 hash, copies the file into `.mgit/objects/<hash>`, and updates the `index` JSON file with the new file path to hash mapping.

### `commit`
Reads the staging area (`index`). Creates a new `Commit` struct, assigning the current HEAD commit as its parent. The struct is serialized to JSON, hashed, and saved to `.mgit/objects/`. Finally, the current branch file in `refs/heads/` is updated to point to this new commit hash.

### `status`
Performs a three-way comparison:
1. **Working Directory vs Index**: Files in the working directory that aren't in the index are *Untracked*. Files in both but with different hashes are *Modified*.
2. **Index vs HEAD Commit**: Files in the index whose hashes differ from the HEAD commit's snapshot are *Staged for commit*.

### `checkout`
When checking out a target (branch or commit hash):
1. Reads the target commit's snapshot.
2. Deletes all files currently tracked by the HEAD commit from the working directory.
3. Copies the blobs specified in the target commit from `.mgit/objects/` back into the working directory.
4. Updates the `.mgit/index` to match the target commit.
5. Updates `.mgit/HEAD` to point to the new branch or commit.

### `log`
Reads the `HEAD` file to find the current commit. It then opens the commit JSON, prints its details, reads the `Parent` hash, and repeats the process until it reaches the initial commit (which has no parent).
