package initializer

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

type initializerProfile struct {
	ui *model.UI
}

func (m *initializerProfile) OnAction() tea.Cmd {
	m.ui.Config, m.ui.IsFirstRun = config.LoadConfig()
	if m.ui.IsFirstRun {
		m.ui.Config.ShortCuts = action.DefaultShortcuts
	}
	_, gitErr := exec.LookPath("git")
	m.ui.GitMissing = gitErr != nil
	return func() tea.Msg {
		return config.VoidMsg{}
	}
}

func InitializeProfile(ui *model.UI) Initializer {
	return &initializerProfile{ui}
}
