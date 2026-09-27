# Go Dark

## User guide

### Initial Step
Fill in the following fields:\
- workspace: where the projects are stored
- git username: to push (still a feature to do) than those info are used
- git email: same as above
- max tag list size: only shows X amount of tags

![welcome](images/welcome.png)

### Dashboard

![project_structure](images/project_structure.png)

### Git

Shortcut: ctrl+g\
Select the project you want to clone\
![git_projects](images/git_projects.png)

![git_operations](images/git_operations.png)\
Select the branch you want to check out\
![branches](images/branches.png)

Project successfully checked out\
![cloned_project](images/cloned_project.png)


### JDK
Shortcut: ctrl+j\
You can choose between configuring a new jdk or assigning a configured one\
![jdk_wizard_welcome](images/jdk_wizard_welcome.png)
![jdk_17](images/jdk_17.png)
![assign_jdk](images/assign_jdk.png)
![select_jdk](images/select_jdk.png)


### Mvn 
Shortcut: ctrl+u\
You can choose between configuring a new mvn or assigning a configured one\
![maven_wizard_welcome](images/maven_wizard_welcome.png)
![mvn_386](images/mvn_386.png)
![assign_mvn](images/assign_mvn.png)
![select_mvn](images/select_mvn.png)

### Building project
Shortcut: ctrl+b\
![build_project](images/build_project.png)\
Session runs in the background\
ctrl+s to call the screen\
![session](images/session.png)
![session_footer](images/session_footer.png)


### Profile
Shortcut: ctrl+y\
![edit_git](images/edit_git.png)
![profile](images/profile.png)

### Edit Shortcuts
Shortcut: ctrl+y\
![edit_shortcuts](images/edit_shortcuts.png)
or using shortcut to go direct to the modal: ctrl+k\
![edit_shortcuts_modal](images/edit_shortcuts_modal.png)
![edited_shortcut](images/edited_shortcut.png)
![shortcut_applied](images/shortcut_applied.png)

## Tech Stack

### TUI
bubble tea: https://github.com/charmbracelet/bubbletea

### Engine
go: https://go.dev/

### Run application
```bash
go run .
```

Logfile: debug.log

### Debugging

Before running this configuration, start your application and Delve as described below.

Allow Delve to compile your application:

```bash
dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient
```

Or compile the application using Go 1.18 or newer:

```bash
go build -gcflags "all=-N -l" github.com/exr462/go-dark
```

and then run it with Delve using the following command:

```bash
dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./go-dark
```