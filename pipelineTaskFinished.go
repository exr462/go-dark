package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/task"
)

func (m *appModel) pipelineTaskFinished(msg task.PipelineTaskFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.state.StatusMsg = fmt.Sprintf("❌ Build Error on component: %s", msg.ProjectName)
	} else {
		m.state.StatusMsg = fmt.Sprintf("✅ Component complete: %s", msg.ProjectName)
	}
	return m, nil
}
