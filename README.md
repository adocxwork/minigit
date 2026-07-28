# mgit - The Miniature Version Control System

**mgit** is a lightweight, simplified version control system built purely in Go (Golang). Designed as an educational and functional alternative to Git, it implements core version control concepts from scratch—including content-addressable storage, branching, automated 3-way merges, full merge conflict resolution, and a built-in Web GUI!

---

## Key Features

### 1. Core Version Control
*   **Initialization**: Run `mgit init` to create a `.mgit` repository.
*   **Staging**: Run `mgit add <file>` or `mgit add .` to stage files to the index.
*   **Committing**: Run `mgit commit -m "msg"` to record snapshots permanently.
*   **History & Status**: Check the state of your working tree with `mgit status` and view your commit timeline with `mgit log`.

### 2. Branching, Merging & History Rewriting
*   **Branches**: Create and manage isolated streams of work using `mgit branch` and `mgit checkout`.
*   **Resetting**: Undo commits and changes using `mgit reset [--soft | --mixed | --hard] <commit>`.
*   **Intelligent 3-Way Merge**: Running `mgit merge <branch>` automatically finds the common ancestor and merges changes.
*   **Merge Conflict Resolution**: Just like real Git, if a merge conflicts, `mgit` halts, generates standard conflict markers (`<<<<<<< HEAD`, `=======`, `>>>>>>>`), and locks the repository into a `MERGE_HEAD` state. You can manually resolve the files, stage them, and commit to generate a true multi-parent merge commit!

### 3. Integrated Web GUI (IDE-Style & Reactive)
*   **Zero Dependencies**: Run `mgit ui` to instantly launch a fully featured Web GUI.
*   **Reactive Polling**: The UI updates automatically via background polling. Changes you make in your CLI or editor appear in the GUI instantly without manual refreshes!
*   **Custom Notifications**: All destructive actions (like Hard Resets) are protected by in-app Modals, and alerts are handled by sleek Toast notifications.
*   **Built-in React**: The modular React frontend is compiled and **embedded** directly inside the Go binary.

---

## How to Setup & Build

Since the React Web GUI is embedded inside the Go executable, building the project requires a 2-step process. (Node.js is only required for building the UI, not for running it).

```bash
# 1. Build the React Frontend (Optional: only if modifying UI code)
cd web
npm install
npm run build
cd ..

# 2. Compile the Go Executable
go build -o mgit main.go
```

Once compiled, you are left with a single portable `mgit` binary!

---

## Command Reference

| Command | Description |
|---|---|
| `mgit init` | Initialize a new, empty mgit repository in the current directory. |
| `mgit add <path>` | Add file contents to the staging area. Use `.` to add all. |
| `mgit commit -m "<msg>"` | Record staged changes to the repository. |
| `mgit status` | Show the working tree status (Staged, Modified, Untracked). |
| `mgit log` | Show the commit history for the current branch. |
| `mgit branch` | List all branches in the repository. |
| `mgit branch <name>` | Create a new branch pointing to the current HEAD. |
| `mgit checkout <name>` | Switch branches or restore working tree files. |
| `mgit merge <branch>` | Merge the specified branch into the current active branch. |
| `mgit reset <commit>` | Reset current HEAD to a specific state (`--soft`, `--mixed`, `--hard`). |
| `mgit ui` | Start the local HTTP Web GUI server on port 8080. |

---

## Example Usage Workflow

```bash
# Initialize a new repository
./mgit init

# Create and stage a file
echo "Hello mgit" > my_file.txt
./mgit add .

# Create the initial commit
./mgit commit -m "Initial commit"

# Create a new feature branch and switch to it
./mgit branch feature-gui
./mgit checkout feature-gui

# Make changes
echo "Feature work" > my_file.txt
./mgit add .
./mgit commit -m "Added feature work"

# Switch back to main and merge the feature branch
./mgit checkout main
./mgit merge feature-gui

# Don't want to use the CLI? Launch the visual UI!
./mgit ui
```
