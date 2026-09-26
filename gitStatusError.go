package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
)

func (m *appModel) gitStatusError(msg config.GitStatusErrorMsg) (tea.Model, tea.Cmd) {
	m.state.GitStatusOutput = fmt.Sprintf("❌ Error: %v", msg)
	return m, nil
}
