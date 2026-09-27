package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) dockerContainers(msg model.DockerContainersMsg) (tea.Model, tea.Cmd) {
	m.ui.DockerContainers = msg
	return m, nil
}
