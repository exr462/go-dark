package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) Init() tea.Cmd {
	if m.state.ViewState == model.StateGitConfigurationModal {
		return textinput.Blink
	}
	return tea.Batch(m.updateWorkspaceFiles(), m.pollDockerTelemetryCmd(), m.TriggerPipelineCmd(4))
}
