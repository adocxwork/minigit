# mgit - Project Documentation

**mgit** is an educational, miniature Version Control System designed to demonstrate the core architectural concepts behind systems like Git. This document provides an in-depth technical overview of how `mgit` functions under the hood.

---

## 1. System Architecture

`mgit` utilizes a local, hidden directory (`.mgit/`) at the root of a project to store all version control data. 

### Content-Addressable Storage
All file contents and commit snapshots are stored in the `.mgit/objects/` directory. `mgit` uses **SHA-1 hashing** to uniquely identify data.
*   **Blobs**: When a file is staged (`mgit add`), its content is hashed. The hash becomes the filename in the `objects/` directory, and the content is the file's payload.
*   **Commits**: A commit object is a JSON representation containing the author's message, timestamp, a map of all files (with their respective blob hashes at that point in time), and an array of parent commit hashes.

### The Index (Staging Area)
The index acts as the middle-ground between the working directory and the commit history. It is stored in `.mgit/index`.
*   It is a serialized map of file paths to blob hashes.
*   `mgit add` writes to the index.
*   `mgit commit` packages the current state of the index into a Commit Object.

### References (Refs)
References are lightweight pointers to commit hashes, stored in `.mgit/refs/heads/`.
*   A branch is simply a text file (e.g., `.mgit/refs/heads/main`) containing the SHA-1 hash of the latest commit on that branch.
*   `HEAD` is a special file (`.mgit/HEAD`) that points to the currently checked-out branch.

### Reset Subsystem
The `mgit reset` command allows for history manipulation and rollback across three distinct operational layers:
1.  **Soft Reset (`--soft`)**: Only updates the branch reference (`HEAD`) to point to the target commit. The index (staging area) and working directory remain completely untouched.
2.  **Mixed Reset (`--mixed` / default)**: Updates the branch reference *and* rewrites the index to match the tree of the target commit. The working directory is left alone.
3.  **Hard Reset (`--hard`)**: Updates the branch reference, rewrites the index, *and* violently overwrites the working directory to precisely match the target commit, discarding any uncommitted local changes.

---

## 2. Merge Subsystem

The `mgit merge` functionality replicates true version control merging by employing a 3-way merge algorithm.

### Common Ancestor Resolution
When merging `Branch B` into `Branch A`, `mgit` traverses the parent history of both branches to find the most recent common commit (the Ancestor). 

### Conflict Detection & Resolution
`mgit` compares the file hashes of `HEAD`, `Target`, and the `Ancestor`:
1.  **Fast-Forward**: If `HEAD` is the ancestor, the merge simply moves the `HEAD` pointer forward to the target commit.
2.  **Clean 3-Way Merge**: If a file was changed in `Target` but not in `HEAD` (relative to the Ancestor), the `Target` version is automatically accepted.
3.  **Merge Conflicts**: If a file was modified differently in *both* `HEAD` and `Target`, `mgit` detects a conflict:
    *   The merge halts.
    *   The conflicting file is rewritten in the working directory containing standard Git conflict markers (`<<<<<<< HEAD`, `=======`, `>>>>>>>`).
    *   The target branch hash is saved to `.mgit/MERGE_HEAD`.
    *   The user must manually resolve the file, run `mgit add`, and run `mgit commit`. 
    *   The commit system reads `MERGE_HEAD` and generates a Commit Object with **two parents**, accurately finalizing the merge.

---

## 3. Integrated Web UI

`mgit` features a built-in HTTP server (`mgit ui`) that hosts a local graphical interface for managing the repository visually.

### Frontend Technology
The frontend is built using **React, Vite, and TypeScript**. It is designed to mimic a professional IDE (like VSCode):
*   **Dark Theme**: Minimalist, professional color palette.
*   **2-Column Layout**: Left panel handles staging and committing; Right panel handles branch management and commit history.

### The `//go:embed` Architecture
Instead of requiring users to download a separate web application, the compiled React bundle (`web/dist/`) is injected directly into the `mgit` Go executable at compile time using Go's native `//go:embed` directive.
*   This makes the `mgit` binary entirely self-contained and portable.
*   The Go backend utilizes standard HTTP multiplexers to serve the embedded static files, while providing dynamic API endpoints (`/api/status`, `/api/commit`, etc.) that execute core `mgit` operations on behalf of the frontend.
