# Weekly Progress Report - mgit Project

**Week 1**
* Finalized the project idea and gathered requirements for building a mini version control system.
* Set up the Go (Golang) development environment on my machine.

**Week 2**
* Researched how version control systems save files and manage history.
* Planned out the basic hidden folder structure we would need to store data (the `.mgit` folder).

**Week 3**
* Set up the command-line (terminal) interface foundation.
* Used a library to help manage basic terminal commands easily.

**Week 4**
* Wrote the code for the `init` command.
* Added logic to automatically create the hidden folder and sub-folders when a user starts a new project.

**Week 5**
* Started working on saving files securely.
* Implemented a hashing function that gives every file a unique ID based on its text content so it can be saved without duplicates.

**Week 6**
* Developed the staging area concept (often called the index).
* Created the `add` command so users can pick and choose which files they want to save.

**Week 7**
* Designed the structure for a "Commit" (a permanent snapshot of the code).
* Decided to use JSON format to store commit details like the message, time, and the files included.

**Week 8**
* Successfully implemented the `commit` command.
* Linked the staging area to the commit creation so it saves a permanent history of the files.

**Week 9**
* Implemented the `status` and `log` commands.
* Wrote logic to compare the user's current files with the staging area, and logic to display the past history of commits.

**Week 10**
* Introduced branching with the `branch` command to allow multiple streams of work.
* Implemented the `checkout` command to allow users to switch between different branches and restore old files safely.

**Week 11**
* Implemented the `merge` command to combine work from different branches safely, and added warnings if two branches changed the same file.
* Added the `reset` command to allow users to undo mistakes and go back in time.

**Week 12**
* Started working on the visual Graphical User Interface (GUI) for the project.
* Designed a clean, dark-themed web page using React so users can view their history and click buttons instead of typing commands.

**Week 13**
* Connected the web GUI to the Go backend program so that they communicate smoothly.
* Conducted final manual testing of all features, cleaned up the code, and finalized all project documentation for submission.
