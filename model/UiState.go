package model

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/lsp"
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
	CfgCatGeneral  GitOperationStep = iota // BasePath & Git Credentials
	CfgCatJDKs                             // Registered Java Environments Pool
	CfgCatMavens                           // Registered Maven Engines Pool
	CfgCatProjects                         // Registered Workspaces List
)

type ApplicationViewState int

const (
	StateDashboard ApplicationViewState = iota
	StateAddProjectModal
	StateInstaller
	StateGitOperationsModal
	StateJDKConfigModal
	StateHelpModal
	StateMavenConfigModal
	StateBuildModal
	StateSessionLogsModal
	StateFuzzyModal
	StateConfigDeckModal
	StateDockerModal
	StateEditorModal
	StateDependencyConfigModal
)

type DockerStats struct {
	CPU      string // e.g. "12.4%"
	Memory   string // e.g. "1.45GB / 16GB"
	Running  int    // Number of active running containers
	Services int    // Number of total configured containers
}

type DockerContainer struct {
	ID     string
	Names  string
	Image  string
	Status string
	Ports  string
}

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

type GitOperationStep int

const (
	StepSelectGitProject GitOperationStep = iota
	StepSelectGitCommand
	StepSelectGitBranch
)

type JDKOperationStep int

const (
	StepSelectJDKAction JDKOperationStep = iota
	StepAddNewJDKVersion
	StepAssignJDKToProject
)

type MavenOperationStep int

const (
	StepSelectMvnAction MavenOperationStep = iota
	StepAddNewMvnVersion
	StepAssignMvnToProject
)

// UIState holds the shared application global model
type UIState struct {
	// !!! GLOBAL VARIABLES MAPPINGS !!!
	Config       config.Config
	ViewState    ApplicationViewState
	WindowWidth  int
	WindowHeight int

	// !!! GLOBAL INSTALLER VARIABLES MAPPINGS !!!
	InstallerStep InstallerStep

	// !!! GLOBAL SCREEN ACTIVITIES VARIABLES MAPPINGS !!!
	ActiveFocus       FocusArea
	SelectedMenuIndex int
	FocusedInput      int
	StatusMsg         string
	Inputs            []textinput.Model

	// !!! GLOBAL FILE SYSTEM VARIABLES MAPPINGS !!!
	SelectedFile     int
	TreeNodes        []FileNode
	Files            []string
	FileViewer       viewport.Model
	CurrentDirectory string

	// !!! GLOBAL BUILD VARIABLES MAPPINGS !!!
	GitCommands         []string
	GitOperationStep    GitOperationStep
	SelectedGitProject  int
	SelectedGitCommand  int
	SelectedBuildOption int
	AvailableBranches   []string
	GitMissing          bool
	SelectedGitBranch   int
	GitStatusOutput     string
	HelpModel           kbd.HelpModel
	KeyMap              kbd.KeyMap
	LastGitActionLog    string

	// !!! GLOBAL BUILD VARIABLES MAPPINGS !!!
	BuildOptions []string
	BuildLogs    []string
	IsBuilding   bool

	// !!! GLOBAL PROJECT VARIABLES MAPPINGS !!!
	SelectedProject int

	// !!! GLOBAL JDK ENGINE PARAMETERS VARIABLES MAPPINGS !!!
	JDKStep          JDKOperationStep // Jdk Step
	SelectedJDKIndex int

	// !!! GLOBAL MAVEN ENGINE PARAMETERS VARIABLES MAPPINGS !!!
	MavenStep          MavenOperationStep // The Maven Operation Step
	SelectedMavenIndex int

	// !!! GLOBAL BACKGROUND SESSION ENGINE STRUCTURES VARIABLES MAPPINGS !!!
	Sessions         map[int]*BuildSession // Stores historical logs keyed by session index identifier
	ActiveSessionID  int                   // The session ID currently being viewed/active
	ViewingSessionID int                   // The session ID currently selected via Ctrl+S inspector modal
	NextSessionID    int                   // Counter managing increment steps

	// !!! GLOBAL FUZZY FINDER RECON ENGINE PARAMETERS VARIABLES MAPPINGS !!!
	FuzzyMode            FuzzyMode       // Track if searching names vs text blocks
	FuzzyResults         []FuzzyResult   // Store current matched elements
	SelectedFuzzy        int             // Selection line index pointer inside results list
	FuzzyQueryInput      textinput.Model // Sub-editor text box specifically for typing queries
	FuzzyViewer          viewport.Model
	SelectedConfigOption int

	// !!! GLOBAL DOCKER MANAGEMENT STATE MAPPINGS !!!
	DockerTelemetry   DockerStats       // Stores data displayed in top-right header
	DockerContainers  []DockerContainer // Parsed rows for the Ctrl+D interaction table
	SelectedDockerRow int               // Highlighted container index in the modal view

	// !!! GLOBAL LSP STATE MAPPINGS !!!
	Provider         lsp.LanguageProvider
	Code             string
	Err              error
	ActiveCodeBuffer string // Holds the text content of the editor buffer
	LastError        error

	// !!! GLOBAL DEPENDENCY SCREEN STATE MAPPINGS !!!
	DepScreen DependencyScreenState
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

type DockerTelemetryMsg DockerStats
type DockerContainersMsg []DockerContainer
