# MiniGit

**A local version control system written in Go, with a React dashboard embedded in the executable.**

MiniGit stores file contents by hash, builds commit history from snapshots and parent links, and merges branches by comparing their snapshots with a common ancestor. The CLI and browser use the same repository engine, so a commit made in one appears in the other.

![MiniGit dashboard showing the working tree, branches, and first-parent history for a sample repository](docs/dashboard.png)

## Try it

Build with **Go 1.26.11 or newer** and **Node.js 24 or newer**. Run these commands from the project root in a macOS or Linux shell.

```sh
(cd web && npm ci && npm run build)
go build -tags ui -o mgit .
```

Start a disposable repository and make its first commit.

```sh
MGIT_BIN="$PWD/mgit"
cd "$(mktemp -d)"
"$MGIT_BIN" init
printf '# Field notes\n' > notes.md
"$MGIT_BIN" add notes.md
"$MGIT_BIN" commit -m "Start field notes"
"$MGIT_BIN" ui
```

Open [the local dashboard](http://127.0.0.1:8080). Create a branch, switch to it, and edit `notes.md` in your editor. Stage and commit the edit in the dashboard, then switch back to `main` and merge the branch. With no commits on `main` since the branch was created, MiniGit takes the fast-forward path.

For a CLI-only build, run `go build -o mgit .` from the project root. On Windows, use `mgit.exe` for the executable name.

## What to look at

- **Storage and staging.** Blobs are deduplicated by content. The index captures exactly what goes into the next commit, including deletions. Status compares the working tree, index, and HEAD. [Staging and objects](pkg/core/staging.go)
- **Branching and merging.** Lightweight refs point into a commit DAG. Diverged branches produce two-parent commits, and conflicting edits require an explicit resolution. [Merge engine](pkg/core/merge.go)
- **Safe snapshot changes.** Checkout and merge stop on tracked changes or untracked collisions. The worktree updater verifies incoming blobs before modifying files and keeps backups for rollback on ordinary filesystem errors. [Worktree updater](pkg/core/worktree.go)
- **One executable for the CLI and dashboard.** The optional Go build tag embeds the frontend. The dashboard shows staged and unstaged files, branch actions, searchable first-parent history, and reset modes. [HTTP server and API](pkg/api/server.go)

## Commands

| Command | What it does |
| --- | --- |
| `mgit init` | Create a local repository with `main` as the initial branch. |
| `mgit add <path>` | Stage a file or directory. Use `.` to stage changes and deletions under the current directory. |
| `mgit status` / `mgit log` | Inspect working changes or first-parent history. |
| `mgit commit -m "message"` | Commit the index. A resolved merge records both parents. |
| `mgit branch [name]` | List branches or create one at HEAD. |
| `mgit checkout <branch\|commit-id>` | Switch branches or check out a full commit ID in detached HEAD. |
| `mgit merge <branch>` | Merge a branch into the current revision. |
| `mgit reset [--soft\|--mixed\|--hard] <target>` | Move HEAD and optionally update the index and working tree. Defaults to mixed. |
| `mgit ui [--port 8080]` | Serve the embedded dashboard on loopback. |

## Verify and explore

```sh
go test -race ./...
go vet ./...
(cd web && npm run lint && npm run build)
go test -tags ui ./...
```

The [CI workflow](.github/workflows/ci.yml) builds and tests on Linux and Windows. The [technical walkthrough](PROJECT_DOCUMENTATION.md) covers the on-disk format, merge rules, failure handling, and interview questions.

MiniGit is an educational, single-machine VCS with its own `.mgit/` format. It focuses on regular files, whole-file snapshots, and file-level merging. Keep a separate backup of important work.
