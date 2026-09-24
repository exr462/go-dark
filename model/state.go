package model

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/exr462/go-build/config"
)

type BuildSession struct {
	ID          int
	ProjectName string
	Command     string
	IsRunning   bool
	Logs        []string
}

type FileNode struct {
	Name       string
	FullPath   string
	IsDir      bool
	IsExpanded bool
	Depth      int
}

type FocusArea int

const (
	FocusProjects FocusArea = iota
	FocusTree
	FocusMenu
)

type AppViewState int

const (
	StateDashboard AppViewState = iota
	StateAddProjectModal
	StateInstaller
	StateGitOpsModal
	StateJDKConfigModal
	StateHelpModal
	StateMavenConfigModal
	StateBuildModal
	StateSessionLogsModal
)

type InstallerStep int

const (
	StepSetGlobalPrefs InstallerStep = iota // Handles BasePath, User, and Email fields
	StepAddFirstProject
)

type GitOpsStep int

const (
	StepSelectGitProject GitOpsStep = iota
	StepSelectGitCommand
)

type JDKOpsStep int

const (
	StepSelectJDKAction JDKOpsStep = iota
	StepAddNewJDKVersion
	StepAssignJDKToProject
)

type MvnOpsStep int

const (
	StepSelectMvnAction MvnOpsStep = iota
	StepAddNewMvnVersion
	StepAssignMvnToProject
)

// UIState holds the shared application global model
type UIState struct {
	Config           config.Config
	ViewState        AppViewState
	InstallerStep    InstallerStep
	JDKStep          JDKOpsStep
	GitOpsStep       GitOpsStep
	MvnStep          MvnOpsStep
	ActiveFocus      FocusArea
	SelectedProj     int
	SelectedFile     int
	TreeNodes        []FileNode
	SelectedGitProj  int
	SelectedGitCmd   int
	SelectedJDKIdx   int
	SelectedMvnIdx   int
	SelectedMenuIdx  int
	GitCommands      []string
	SelectedBuildOpt int
	BuildOptions     []string
	BuildLogs        []string
	IsBuilding       bool
	// !!! GLOBAL BACKGROUND SESSION ENGINE STRUCTURES VARIABLES MAPPINGS !!!
	Sessions         map[int]*BuildSession // Stores historical logs keyed by session index identifier
	ActiveSessionID  int                   // The session ID currently being viewed/active
	ViewingSessionID int                   // The session ID currently selected via Ctrl+S inspector modal
	NextSessionID    int                   // Counter managing increment steps

	Files        []string
	FileViewer   viewport.Model
	StatusMsg    string
	TerminalW    int
	TerminalH    int
	Inputs       []textinput.Model
	FocusedInput int
	GitMissing   bool
}

type StatusMsg string
type FileLoadMsg string
type ConfigRefreshedMsg config.Config
type BuildLogLineMsg struct {
	SessionID int
	Line      string
}
type BuildCompleteMsg struct {
	SessionID int
	Err       error
}
