# mgit - Detailed Project Documentation

## What is a Version Control System (VCS)?
Imagine you are writing a very long essay or building a big project. Sometimes, you make a mistake and want to go back to how the project looked yesterday. Instead of manually saving copies of your project folder (like `project_final`, `project_final_2`, `project_real_final`), a Version Control System (VCS) does this for you automatically. It saves snapshots of your work so you can safely try new things, see what changed, and easily go back in time if something breaks.

**mgit** is a miniature, educational version of a Version Control System. It is built to help users understand exactly how tools like Git work behind the scenes.

---

## 1. How the Core Works

When you start using `mgit`, it needs a place to store all your history and saved files.

### The Hidden `.mgit` Folder
When you run the `init` command, the project creates a hidden folder named `.mgit`. This folder acts as the "brain" or the database of your project. If you delete this folder, all your history is gone, but your actual current files remain safe.

### Saving Files (The Object Store)
When you tell `mgit` to track a file (using the `add` command), it does something very clever:
1. It reads the contents of your file.
2. It generates a unique ID for that content using a math algorithm called SHA-1 hashing.
3. It saves a copy of your file inside the `.mgit/objects/` folder, using that unique ID as the filename.
By doing this, `mgit` ensures that it never saves duplicate copies of the exact same file, saving space.

### The Staging Area (The Index)
The staging area is like a shopping cart. Before you permanently save a snapshot of your project, you put the files you modified into the staging area (using `add`). This way, you can choose exactly which files should be included in your next save, and which ones should be left out for later.

### Snapshots (Commits)
A "Commit" is a permanent snapshot of your project. When you run the `commit` command, `mgit` takes everything in your staging area and packages it together with:
* Your name and message (e.g., "Added login page").
* The exact date and time.
* A link to the previous commit (so they form a chain of history).

---

## 2. Navigating History

### Branches
In a project, you might want to try a new crazy idea without ruining the main project. This is what Branches are for. In `mgit`, a branch is literally just a tiny text file that contains the unique ID of your latest commit. 
When you create a new branch, you are just creating a new text file. You can then switch back and forth between branches (using `checkout`), and `mgit` will magically change the files in your folder to match that branch's history.

### Merging
When you are done with your crazy idea on a separate branch, you will want to combine it back into the main project. This is called Merging.
`mgit` looks at the history of both branches. If the files can be safely combined, it does it automatically. If you changed the exact same line of code in both branches differently, `mgit` will stop and warn you. This is called a **Merge Conflict**. It will ask you to manually choose which line of code to keep before finalizing the merge.

### Undo Mistakes (Reset)
If you made a terrible mistake, you can use the `reset` command. This tells `mgit` to forcefully move your project back in time to an older commit. 
* A **Soft** reset just moves the history pointer, keeping your files safe.
* A **Hard** reset moves the history pointer and actively erases any new files you were working on to perfectly match the old history.

---

## 3. The Graphical User Interface (GUI)

While `mgit` works perfectly well in the command-line terminal, we also built a visual web interface to make it easier to use. 

When you run the command `mgit ui`, a local web server starts, and you can open a web browser to see your project.
* **Visual File List**: You can see exactly which files have been modified and click a button to stage them.
* **History Table**: You can read all your past commits in a neat table.
* **Interactive Buttons**: You can click buttons to create branches, merge code, or reset history instead of typing long commands.

The entire web interface is built using modern web technologies (React) and is embedded directly inside the `mgit` program, making it very fast and easy to run on any computer.
