# Weekly Progress Report - mgit Project

**Week 1**
* Finalized the project idea and gathered requirements for building a mini version control system.
* Set up the Go (Golang) development environment and studied Go's file I/O capabilities.

**Week 2**
* Researched how Git works under the hood, specifically focusing on content-addressable storage.
* Sketched out the basic folder structure we would need (like `.mgit/objects`).

**Week 3**
* Set up the command-line interface (CLI) foundation.
* Integrated the Cobra library to help manage commands like `init`, `add`, and `commit`.

**Week 4**
* Implemented the `mgit init` command.
* Added logic to automatically create the hidden `.mgit` folder and its necessary sub-directories.

**Week 5**
* Started working on file tracking by implementing a hashing function (SHA-1).
* Wrote the code that takes a file, hashes its contents, and saves it as a "blob" object.

**Week 6**
* Developed the staging area concept (the index).
* Implemented the `mgit add` command to track files and update the index file.

**Week 7**
* Designed the structure for a "Commit".
* Decided to use JSON to store commit details like the message, timestamp, parent commit, and tracked files.

**Week 8**
* Successfully implemented the `mgit commit` command.
* Linked the staging area to the commit creation so it saves a permanent snapshot of the files.

**Week 9**
* Implemented the `mgit status` command.
* Wrote the logic to compare the working directory with the staging area to detect modified or untracked files.

**Week 10**
* Added the `mgit log` command to view history.
* Wrote a loop that reads a commit, prints its message, and moves backward to its parent commit.

**Week 11**
* Introduced basic branching with the `mgit branch` command.
* Implemented the `mgit checkout` command to allow switching between different branches and restoring old files.

**Week 12**
* Researched and implemented a simple `mgit merge` command.
* Added support for "fast-forward" merges to combine work from different branches safely.

**Week 13**
* Conducted final manual testing of all the commands to ensure they work together smoothly.
* Cleaned up the codebase, fixed minor bugs, and wrote the final project documentation.
