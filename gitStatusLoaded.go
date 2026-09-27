package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
)

func (m *appModel) gitStatusLoaded(msg config.GitStatusLoadedMsg) (tea.Model, tea.Cmd) {
	m.ui.GitStatusOutput = string(msg)
	if m.ui.GitStatusOutput == "" {
		m.ui.GitStatusOutput = "✨ Working tree clean."
	}
	return m, nil
}
