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
	"github.com/exr462/go-dark/ui/components"
)

type preflightMsg struct {
	Function func() tea.Cmd
}

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) Init() tea.Cmd {
	if m.ui.ViewState == model.StateGitConfigurationModal {
		m.ui.PreviousViewState = model.StateDashboard
		return textinput.Blink
	}
	m.maxParallelism = 4
	m.ui.Prechecks = []model.Precheck{
		{
			Name: lipgloss.NewStyle().Render("🛎 Profile "),
			Load: m.loadingProfile,
		},
		{
			Name: lipgloss.NewStyle().Render("👷 Load env"),
			Load: m.settingWelcomePanel,
		},
		{
			Name: lipgloss.NewStyle().Render("⛵ Updating env"),
			Load: m.updateWorkspaceFiles,
		},
		{
			Name: lipgloss.NewStyle().Render("🐳 Docker"),
			Load: m.pollDockerTelemetryCmd,
		},
		{
			Name: lipgloss.NewStyle().Render("🪁 Async"),
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
	m.ui.Progress = progress.New(
		progress.WithDefaultGradient(),
		progress.WithSolidFill(components.MochaGreen),
		progress.WithoutPercentage())
	m.ui.Spinner = spinner.New()
	m.ui.Editor = ta
	return tea.Batch(m.preflight())
}

func (m *appModel) preflight() tea.Cmd {
	d := time.Millisecond * time.Duration(500)
	return tea.Tick(d, func(time time.Time) tea.Msg {
		precheck := m.ui.Prechecks[m.ui.Index]
		return preflightMsg{
			Function: precheck.Load,
		}
	})
}
