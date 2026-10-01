package model

import (
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/docker"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/terminal"
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

type ApplicationViewState int

const (
	StateDashboard ApplicationViewState = iota
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
	StateGitConfigurationModal
	StateShortcutConfigurationModal
	StateSystemCheckModal
	StateTerminalCockpit
	StateDeployModal
)

type FuzzyMode int

const (
	FuzzyModeFiles   FuzzyMode = iota // Finds matching file name titles
	FuzzyModeContent                  // Scans deeply inside text strings
)

// EditorMode drives the Vim-style modal editing workflow of the in-app code
// editor: Normal mode for navigation/commands, Insert mode for typing, and
// Command mode for ":w" / ":q" / ":wq" / ":q!" style command-line input.
type EditorMode int

const (
	EditorModeNormal EditorMode = iota
	EditorModeInsert
	EditorModeCommand
)

// FuzzyResult Add a tracking structure to hold fuzzy matching results rows
type FuzzyResult struct {
	FileName string
	FullPath string
	LineNum  int    // Used if content-searching (0 if file-only search)
	Snippet  string // Shows the matched text phrase context match
}
type ConfigurationField int

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

type Precheck struct {
	Name string
	Load func() tea.Cmd
}

type MavenOperationStep int

const (
	StepSelectMvnAction MavenOperationStep = iota
	StepAddNewMvnVersion
	StepAssignMvnToProject
)

// UI holds the shared application global model
type UI struct {
	// !!! GLOBAL VARIABLES MAPPINGS !!!
	Config                  config.Config
	ViewState               ApplicationViewState
	PreviousViewState       ApplicationViewState
	WindowWidth             int
	WindowHeight            int
	Editor                  textarea.Model
	ActiveLanguageProvider  lsp.LanguageProvider
	ActiveFilePath          string     // Absolute path of the file currently loaded into Editor
	EditorOriginalContent   string     // Snapshot used to detect unsaved edits and to revert on Cancel
	EditorDirty             bool       // True once Editor.Value() diverges from EditorOriginalContent
	EditorMode              EditorMode // Vim-style mode: Normal, Insert or Command
	EditorCommandBuffer     string     // Characters typed after ":" while in Command mode (without the leading ":")
	EditorPendingKey        string     // Holds a leading key of a two-key Normal-mode combo (e.g. "g" before "gg", "d" before "dd")
	IsFirstRun              bool
	Index                   int
	Spinner                 spinner.Model
	Progress                progress.Model
	Prechecks               []Precheck
	Done                    bool
	MaxParallelism          int
	ActiveTerminalSessionID int                           // Tracks current terminal channel handle
	TerminalLogs            []terminal.LogLine            // Active console feed store
	TerminalSessions        map[int]terminal.SessionState // Global ongoing background routines registry

	// !!! GLOBAL SCREEN ACTIVITIES VARIABLES MAPPINGS !!!
	ActiveFocus       FocusArea
	SelectedMenuIndex int
	FocusedInput      kbd.InputField
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
	DockerTelemetry   docker.DockerStats       // Stores data displayed in top-right header
	DockerContainers  []docker.DockerContainer // Parsed rows for the Ctrl+D interaction table
	SelectedDockerRow int                      // Highlighted container index in the modal view

	// !!! GLOBAL LSP STATE MAPPINGS !!!
	Provider         lsp.LanguageProvider
	Code             string
	Err              error
	ActiveCodeBuffer string // Holds the text content of the editor buffer
	LastError        error

	// !!! GLOBAL DEPENDENCY SCREEN STATE MAPPINGS !!!
	DepScreen DependencyScreenState

	// !!! GLOBAL DEPLOY ENGINE STATE MAPPINGS (simulated Rancher/K8s rollout) !!!
	DeployNamespace string
	DeployScript    []string // Full pre-computed rollout transcript
	DeployLogs      []string // Lines revealed so far (streamed for a live feel)
	IsDeploying     bool
	DeployDone      bool
	DeployError     error
}

type FileLoadMsg string
type ConfigRefreshedMsg config.Config
