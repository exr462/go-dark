package model

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/exr462/go-build/config"
)

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
	Config          config.Config
	ViewState       AppViewState
	InstallerStep   InstallerStep
	JDKStep         JDKOpsStep
	GitOpsStep      GitOpsStep
	MvnStep         MvnOpsStep
	ActiveFocus     FocusArea
	SelectedProj    int
	SelectedFile    int
	SelectedGitProj int
	SelectedGitCmd  int
	SelectedJDKIdx  int
	SelectedMvnIdx  int
	SelectedMenuIdx int
	GitCommands     []string
	Files           []string
	FileViewer      viewport.Model
	StatusMsg       string
	TerminalW       int
	TerminalH       int
	Inputs          []textinput.Model
	FocusedInput    int
	GitMissing      bool
}

type StatusMsg string
type FileLoadMsg string
type ConfigRefreshedMsg config.Config
