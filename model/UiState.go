package model

import (
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/docker"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/maven"
	"github.com/exr462/go-dark/shell"
)

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
	StateTeamsModal
	StateProjectsModal
	StateGraphModal
	StateAlignModal
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
	Config                 config.Config
	ViewState              ApplicationViewState
	PreviousViewState      ApplicationViewState
	WindowWidth            int
	WindowHeight           int
	Editor                 textarea.Model
	ActiveLanguageProvider lsp.LanguageProvider
	ActiveFilePath         string     // Absolute path of the file currently loaded into Editor
	EditorOriginalContent  string     // Snapshot used to detect unsaved edits and to revert on Cancel
	EditorDirty            bool       // True once Editor.Value() diverges from EditorOriginalContent
	EditorMode             EditorMode // Vim-style mode: Normal, Insert or Command
	EditorCommandBuffer    string     // Characters typed after ":" while in Command mode (without the leading ":")
	EditorPendingKey       string     // Holds a leading key of a two-key Normal-mode combo (e.g. "g" before "gg", "d" before "dd")
	IsFirstRun             bool
	Index                  int
	Spinner                spinner.Model
	Progress               progress.Model
	Prechecks              []Precheck
	Done                   bool
	MaxParallelism         int
	TerminalTabs           []int // shell.Manager session IDs shown as terminal tabs, in tab order
	ActiveTerminalTab      int   // Index into TerminalTabs
	TerminalPicking        bool  // True while choosing the project for a new tab
	TerminalPickIndex      int   // Highlighted project in the new-tab picker

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

	// !!! GLOBAL PROJECT VARIABLES MAPPINGS !!!
	SelectedProject int

	// !!! GLOBAL JDK ENGINE PARAMETERS VARIABLES MAPPINGS !!!
	JDKStep          JDKOperationStep // Jdk Step
	SelectedJDKIndex int

	// !!! GLOBAL MAVEN ENGINE PARAMETERS VARIABLES MAPPINGS !!!
	MavenStep          MavenOperationStep // The Maven Operation Step
	SelectedMavenIndex int

	// !!! GLOBAL BACKGROUND SESSION ENGINE STRUCTURES VARIABLES MAPPINGS !!!
	Shells          *shell.Manager // Every PTY session: terminal tabs, builds, pipeline builds (nil in some tests)
	SelectedSession int            // Highlighted row of the session inspector

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

	// !!! GLOBAL TEAMS & PROJECTS MANAGEMENT STATE MAPPINGS !!!
	SelectedTeam           int  // Highlighted row in the teams modal
	AddingTeam             bool // True while typing a new team manifest git URL
	SelectedManagedProject int  // Highlighted row in the projects modal
	AddingProject          bool // True while the "add local project" form is open
	ProjectFormField       int  // Focused row of the form: 0 name, 1 git URL, 2 path, 3 type
	ProjectFormType        int  // Index into config.PresetNames

	// !!! GLOBAL DEPENDENCY GRAPH STATE MAPPINGS !!!
	Workspace       *maven.Workspace  // Last pom scan (nil until the graph screen scanned once)
	WorkspaceErrors []error           // Problems found while scanning poms
	GraphScanning   bool              // A pom scan is running
	SelectedGraph   int               // Highlighted project (index into GraphRows) on the graph screen
	GraphDetail     bool              // Show every pom reference of the highlighted project
	ReleaseTags     map[string]string // Newest release version per scanned project (from git tags)

	// !!! GLOBAL VERSION ALIGNMENT STATE MAPPINGS !!!
	AlignChoices  map[string]AlignTarget // Chosen target per producing project (default: workspace version)
	SelectedAlign int                    // Highlighted row
	AlignPreview  bool                   // Show the line-by-line preview
	AlignConfirm  bool                   // Waiting for the second enter before committing
	AlignRunning  bool                   // Commits are being made
	AlignResults  []AlignResult          // Outcome of the last run
	AlignReturnTo ApplicationViewState   // Screen to go back to

	// !!! GLOBAL DEPENDENCY SCREEN STATE MAPPINGS !!!
	DepScreen DependencyScreenState

	// !!! GLOBAL LOCAL DEPLOY STATE MAPPINGS (local Rancher sandbox) !!!
	DeployWaves    [][]deploy.Target  // What will be deployed, in dependency waves
	DeploySkipped  []deploy.Target    // Deployable projects that can't be deployed locally (and why)
	DeployPlanErr  error              // e.g. a dependency cycle
	DeployProgress *deploy.Progress   // Live per-target status of the current/last run
	DeployRunning  bool               // A deployment is in progress
	DeployFinished bool               // The last run ended (see DeployErr)
	DeployErr      error              // Why the last run failed (nil on success)
	DeployCancel   context.CancelFunc // Cancels the running deployment
	ClusterChecked bool               // The preflight finished at least once
	ClusterRunning bool               // the cluster's container runs and its API answers
	ClusterErr     error              // container missing / guard refusal
	ClusterStartID int                // Session starting the cluster's container (0 when none)
	ShipStage      ShipStage          // Where the full train (build -> images -> deploy) is
	ShipDriftAck   bool               // The user saw the version drift warning and ships anyway
	ScaledDown     []string           // Deployments go-dark scaled to 0 to make room (restorable)
}

type FileLoadMsg string
type ConfigRefreshedMsg config.Config

// Terminal pane layout (see component.RenderTerminalExecution): one blank
// row, then a bordered box (1-cell border, 1-column side padding) whose first
// row is the tab bar, then the hint line at the very bottom.
const (
	TerminalPaneLeft = 4 // margin(2) + border(1) + padding(1)
	TerminalPaneTop  = 3 // blank row(1) + border(1) + tab bar(1)
)

// TerminalPaneSize is the size, in cells, of the embedded terminal screen
// for the current window. Sessions are started and resized to it.
func (ui *UI) TerminalPaneSize() (width, height int) {
	return max(ui.WindowWidth-2*TerminalPaneLeft, 20), max(ui.WindowHeight-TerminalPaneTop-3, 5)
}

// ActiveTerminalSession returns the session ID of the active tab, or 0.
func (ui *UI) ActiveTerminalSession() int {
	if ui.ActiveTerminalTab < 0 || ui.ActiveTerminalTab >= len(ui.TerminalTabs) {
		return 0
	}
	return ui.TerminalTabs[ui.ActiveTerminalTab]
}

// DeployPreflightMsg reports the local cluster state.
type DeployPreflightMsg struct {
	Running    bool
	Err        error    // cluster container missing, or the guard refused the context
	ScaledDown []string // deployments go-dark scaled to 0 earlier
}

// DeployFinishedMsg ends a deployment run.
type DeployFinishedMsg struct{ Err error }

// ShipStage tracks the full train started from the deploy screen.
type ShipStage int

const (
	ShipIdle ShipStage = iota
	ShipBuilding
	ShipDeploying
	ShipDone
	ShipFailed
)

// ShipStartMsg starts the train's build stage (sent once the checks pass).
type ShipStartMsg struct{}

// ScaledMsg reports a scale-down (z) or restore (u) on the ship screen.
type ScaledMsg struct {
	Restore bool
	Names   []string // deployments scaled down / restored
	Left    []string // deployments still scaled down by go-dark afterwards
	Err     error
}
