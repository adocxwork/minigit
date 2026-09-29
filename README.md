# mgit

<p align="center">
  <strong>A miniature, educational Version Control System built from scratch in Go and React.</strong>
</p>

## Overview

`mgit` is a lightweight, fully functional version control system designed to demonstrate the core architectural concepts behind tools like Git. It features a complete command-line interface (CLI) and an integrated, reactive Web GUI embedded directly into the compiled Go binary. 

This project is built for educational purposes to showcase how content-addressable storage, tree-based histories, and 3-way merge algorithms operate under the hood.

## Features

### 1. Core Version Control
*   **Content-Addressable Storage**: Files are securely hashed (SHA-1) and stored as immutable blobs to prevent duplication.
*   **Staging Area (Index)**: Granular control over which modifications are included in the next snapshot.
*   **Committing**: Permanent, timestamped snapshots of the repository state linking back to their parent commits to form a directed acyclic graph (DAG).
*   **History & Status**: Real-time diffing between the working directory, index, and HEAD commit to accurately report staged, modified, and untracked files.

### 2. Branching & History Rewriting
*   **Branches**: Lightweight pointers for creating isolated streams of development (`mgit branch`, `mgit checkout`).
*   **Intelligent 3-Way Merge**: Running `mgit merge <branch>` automatically finds the common ancestor and attempts a clean 3-way merge.
*   **Merge Conflict Resolution**: If a conflict occurs, `mgit` halts, generates standard conflict markers (`<<<<<<< HEAD`, `=======`, `>>>>>>>`), and locks the repository into a `MERGE_HEAD` state for manual resolution.
*   **Reset Subsystem**: Undo mistakes and manipulate history using `mgit reset [--soft | --mixed | --hard] <commit>`.
*   **Workspace Protection**: Destructive commands (like checkout and merge) strictly verify the working directory is clean to prevent accidental data loss.

### 3. Integrated Web GUI
*   **Zero Dependencies**: Run `mgit ui` to instantly launch a fully featured Web GUI on localhost:8080.
*   **Built-in React**: The React frontend (built with Vite and TypeScript) is compiled and **embedded** directly inside the Go binary using the `//go:embed` directive.
*   **Reactive Polling**: The UI updates automatically via background polling. Changes made via the CLI or external editors reflect in the GUI instantly without manual refreshes.
*   **Professional IDE Interface**: A sleek, dark-themed, 2-column layout heavily inspired by modern IDEs, featuring custom modals, toast notifications, and keyboard shortcuts.

## Installation & Build

### Prerequisites
*   [Go](https://golang.org/doc/install) (1.16+)
*   [Node.js](https://nodejs.org/) and `npm` (for building the frontend)

### Build Instructions

1.  **Clone the repository** (if applicable) and navigate to the root directory.
2.  **Build the Frontend**:
    ```bash
    cd web
    npm install
    npm run build
    cd ..
    ```
3.  **Build the Go Binary**:
    ```bash
    go build -o mgit main.go
    ```

You will now have a standalone `mgit` executable in your directory.

## Command Reference

| Command | Description |
| :--- | :--- |
| `mgit init` | Initialize a new, empty repository in the current directory. |
| `mgit add <path>` | Add file contents to the staging area (index). |
| `mgit commit -m "msg"` | Record changes to the repository history. |
| `mgit status` | Show the working tree status (staged, modified, untracked). |
| `mgit log` | Display the commit history of the current branch. |
| `mgit branch <name>` | Create a new branch pointing to the current HEAD. |
| `mgit checkout <name>` | Switch branches or restore working tree files. |
| `mgit merge <branch>` | Merge the specified branch into the current active branch. |
| `mgit reset <commit>` | Reset current HEAD to a specific state (`--soft`, `--mixed`, `--hard`). |
| `mgit ui` | Start the local HTTP Web GUI server on port 8080. |

## Technical Documentation

For a deep dive into the internal architecture, algorithms, and data structures used in `mgit`, please refer to the [Project Documentation](PROJECT_DOCUMENTATION.md).
