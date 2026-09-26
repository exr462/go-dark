package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) configRefreshed(msg model.ConfigRefreshedMsg, cmds []tea.Cmd) []tea.Cmd {
	m.state.Config = config.Config(msg)
	cmds = append(cmds, m.updateWorkspaceFiles())
	return cmds
}
