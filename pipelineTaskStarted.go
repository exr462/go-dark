package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/task"
)

func (m *appModel) pipelineTaskStarted(msg task.PipelineTaskStartedMsg) (tea.Model, tea.Cmd) {
	m.state.StatusMsg = fmt.Sprintf("🏗️  Building: %s...", msg)
	return m, nil
}
