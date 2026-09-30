package initializer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

type initializerWelcomePanel struct {
	UI *model.UI
}

func (m *initializerWelcomePanel) OnAction() tea.Cmd {
	inputs := MakeInputs()
	initialState := model.StateSystemCheckModal
	if m.UI.IsFirstRun || m.UI.GitMissing {
		initialState = model.StateGitConfigurationModal
		inputs[kbd.GitWorkspace].Focus()
	}
	m.UI.ViewState = initialState
	m.UI.Inputs = inputs
	return func() tea.Msg {
		return config.VoidMsg{}
	}
}

func InitializeWelcomePanel(ui *model.UI) Initializer {
	return &initializerWelcomePanel{ui}
}
