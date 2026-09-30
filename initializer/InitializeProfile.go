package initializer

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

type initializerProfile struct {
	UI *model.UI
}

func (m *initializerProfile) OnAction() tea.Cmd {
	m.UI.Config, m.UI.IsFirstRun = config.LoadConfig()
	if m.UI.IsFirstRun {
		m.UI.Config.ShortCuts = action.DefaultShortcuts
	}
	_, gitErr := exec.LookPath("git")
	m.UI.GitMissing = gitErr != nil
	return func() tea.Msg {
		return config.VoidMsg{}
	}
}

func InitializeProfile(ui *model.UI) Initializer {
	return &initializerProfile{ui}
}
