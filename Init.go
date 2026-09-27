package main

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) Init() tea.Cmd {
	if m.state.ViewState == model.StateGitConfigurationModal {
		m.state.PreviousViewState = model.StateDashboard
		return textinput.Blink
	}
	ta := textarea.New()
	//ta.SetWidth(m.state.WindowWidth - 8)
	//ta.SetHeight(m.state.WindowHeight - 12)
	ta.SetHeight(10)
	ta.SetWidth(10)
	//ta.MaxHeight = m.state.WindowHeight - 10
	//ta.MaxWidth = m.state.WindowWidth - 8
	ta.ShowLineNumbers = true
	m.providerFactory = lsp.NewProviderFactory([]lsp.LanguageProvider{
		lsp.GoProvider{},
		lsp.KotlinProvider{},
		lsp.JavaProvider{},
	})
	m.state.Editor = ta
	return tea.Batch(m.updateWorkspaceFiles(), m.pollDockerTelemetryCmd(), m.TriggerPipelineCmd(4), tea.DisableMouse)
}
