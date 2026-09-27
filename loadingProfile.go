package main

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
)

func (m *appModel) loadingProfile() tea.Cmd {
	var cfg config.Config
	cfg, m.isFirstRun = config.LoadConfig()
	m.state.Config = cfg
	_, gitErr := exec.LookPath("git")
	m.state.GitMissing = gitErr != nil
	return func() tea.Msg {
		return config.VoidMsg{}
	}
}
