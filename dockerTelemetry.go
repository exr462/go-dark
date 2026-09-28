package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) dockerTelemetry(msg model.DockerTelemetryMsg) (tea.Model, tea.Cmd) {
	m.ui.DockerTelemetry = model.DockerStats(msg)
	return m, initializer.InitializeDockerTelemetry(m.ui).OnAction()
}
