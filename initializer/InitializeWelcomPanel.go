package initializer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

type initializerWelcomePanel struct {
	ui *model.UI
}

func (m *initializerWelcomePanel) OnAction() tea.Cmd {
	inputs := MakeInputs()
	initialState := model.StateSystemCheckModal
	if m.ui.IsFirstRun || m.ui.GitMissing {
		initialState = model.StateGitConfigurationModal
		inputs[model.GitWorkspace].Focus()
	}
	m.ui.ViewState = initialState
	m.ui.Inputs = inputs
	return func() tea.Msg {
		return config.VoidMsg{}
	}
}

func InitializeWelcomePanel(ui *model.UI) Initializer {
	return &initializerWelcomePanel{ui}
}
