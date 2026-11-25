# MiniGit

Track changes in your own folders with Git-style commands or a browser dashboard. Both use the same local history.

![MiniGit dashboard showing the working tree, branches, and commit history](docs/dashboard.png)

## Build once

Open **Terminal** in the MiniGit source folder. You’ll need Go 1.26.11+ and Node.js 24+.

```sh
(cd web && npm ci && npm run build)
go build -tags ui -o mgit .
export PATH="$PWD:$PATH"
```

The last line makes `mgit` available in this Terminal tab, even after you switch folders.

## Track your folder

In the same Terminal tab, replace the path below with the folder you want to track.

```sh
cd "/path/to/your/folder"
mgit init
mgit status
mgit add .
mgit commit -m "First commit"
```

Run `mgit init` once per folder. After editing files, run `mgit status` to see changes, then stage and commit them. Use `mgit add .` for everything under the current folder, or `mgit add "path/to/file-or-folder"` to stage a specific file or subfolder.

## Open the dashboard

From the folder you’re tracking, run `mgit ui` and open [http://127.0.0.1:8080](http://127.0.0.1:8080). You can stage files, commit, switch branches, and view the same history from your browser. Press **Ctrl+C** in Terminal to stop the dashboard and return to the command prompt.

## Branches and history

| Command | Action |
| --- | --- |
| `mgit log` | View commit history |
| `mgit branch experiment` | Create a branch after your first commit |
| `mgit checkout experiment` | Switch to that branch |
| `mgit checkout main` | Switch back to the main branch |
| `mgit merge experiment` | Merge the experiment branch into the current branch |

MiniGit uses Go for content-addressed storage, snapshots, and three-way merges. The React dashboard is bundled into the executable. [Technical walkthrough](PROJECT_DOCUMENTATION.md) · [Automated checks](.github/workflows/ci.yml)
