# mgit

**mgit** is a minimalist, Git-like version control system written in Go. It provides fundamental tracking, staging, committing, and branching features via a simple command-line interface.

## How to setup this tool?

### Prerequisites
* Go (Golang) installed on your system.

### Installation Steps

1. **Clone or Download the source code**
   Open your terminal and navigate to the `minigit` project directory.
   ```bash
   cd path/to/minigit
   ```

2. **Build the executable**
   Compile the Go source code into a runnable binary.
   ```bash
   go build -o mgit main.go
   ```

3. **Install Globally (Optional but Recommended)**
   To use `mgit` from any folder on your computer without typing the full path, move the built executable to your system's binaries folder:
   
   **On macOS / Linux:**
   ```bash
   sudo mv mgit /usr/local/bin/
   ```
   *(You can now use the `mgit` command from anywhere).*

## How to use this tool?

`mgit` works completely independently from its source code. You use the compiled `mgit` tool to track **other** projects on your computer.

### Quick Start Example

1. **Create a new project directory:**
   ```bash
   mkdir my-awesome-project
   cd my-awesome-project
   ```

2. **Initialize mgit:**
   ```bash
   mgit init
   ```
   *(This creates a hidden `.mgit` folder to track your project's history).*

3. **Create and stage a file:**
   ```bash
   echo "Hello World" > readme.txt
   mgit add readme.txt
   ```

4. **Commit your changes:**
   ```bash
   mgit commit -m "Initial commit with readme"
   ```

5. **Check your project's status and history:**
   ```bash
   mgit status
   mgit log
   ```

## Available Features / Commands

`mgit` supports the core workflow of a version control system:

* `mgit init`
  Initialize a new, empty repository.

* `mgit add <file|directory>`
  Add file contents to the staging area (the index). You can pass a specific file (e.g., `mgit add file.txt`) or a directory (e.g., `mgit add .` to add everything).

* `mgit commit -m "<message>"`
  Record the staged changes to the repository with a descriptive message.

* `mgit status`
  Show the working tree status. It displays files that are untracked, modified, or staged for the next commit.

* `mgit log`
  Display the commit history for the current branch in reverse chronological order.

* `mgit branch`
  List all existing branches in the repository. The currently active branch is marked with an asterisk (`*`).

* `mgit branch <branch-name>`
  Create a new branch pointing to the current commit.

* `mgit checkout <branch-name|commit-hash>`
  Switch branches or restore working tree files. This will update your files to match the exact state they were in on that branch or commit.
