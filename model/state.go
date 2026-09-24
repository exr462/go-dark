package model

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/exr462/go-dark/config"
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

type ConfigCategory int

const (
	CfgCatGeneral  GitOpsStep = iota // BasePath & Git Credentials
	CfgCatJDKs                       // Registered Java Environments Pool
	CfgCatMavens                     // Registered Maven Engines Pool
	CfgCatProjects                   // Registered Workspaces List
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
	StateFuzzyModal
	StateConfigDeckModal
)

type FuzzyMode int

const (
	FuzzyModeFiles   FuzzyMode = iota // Finds matching file name titles
	FuzzyModeContent                  // Scans deeply inside text strings
)

// FuzzyResult Add a tracking structure to hold fuzzy matching results rows
type FuzzyResult struct {
	FileName string
	FullPath string
	LineNum  int    // Used if content-searching (0 if file-only search)
	Snippet  string // Shows the matched text phrase context match
}

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
	// !!! GLOBAL FUZZY FINDER RECON ENGINE PARAMETERS VARIABLES MAPPINGS !!!
	FuzzyMode            FuzzyMode       // Track if searching names vs text blocks
	FuzzyResults         []FuzzyResult   // Store current matched elements
	SelectedFuzzy        int             // Selection line index pointer inside results list
	FuzzyQueryInput      textinput.Model // Sub-editor text box specifically for typing queries
	FuzzyViewer          viewport.Model
	SelectedConfigOption int
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
