package main

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
)

type preflightMsg struct {
	Function func() tea.Cmd
}

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) Init() tea.Cmd {
	if m.state.ViewState == model.StateGitConfigurationModal {
		m.state.PreviousViewState = model.StateDashboard
		return textinput.Blink
	}
	m.maxParallelism = 4
	m.state.Prechecks = []model.Precheck{
		{
			Name: lipgloss.NewStyle().Render("Updating Workspace"),
			Load: m.updateWorkspaceFiles,
		},
		{
			Name: lipgloss.NewStyle().Render("Polling Docker Telemetry"),
			Load: m.pollDockerTelemetryCmd,
		},
		{
			Name: lipgloss.NewStyle().Render("Starting Pipeline in the background"),
			Load: m.TriggerPipelineCmd,
		},
	}
	ta := textarea.New()
	ta.SetHeight(10)
	ta.SetWidth(10)
	ta.ShowLineNumbers = true
	m.providerFactory = lsp.NewProviderFactory([]lsp.LanguageProvider{
		lsp.GoProvider{},
		lsp.KotlinProvider{},
		lsp.JavaProvider{},
	})
	m.state.Progress = progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage())
	m.state.Spinner = spinner.New()
	m.state.Editor = ta
	return tea.Batch(m.preflight())
}

func (m *appModel) preflight() tea.Cmd {
	d := time.Millisecond * time.Duration(500)
	return tea.Tick(d, func(time time.Time) tea.Msg {
		precheck := m.state.Prechecks[m.state.Index]
		return preflightMsg{
			Function: precheck.Load,
		}
	})
}
