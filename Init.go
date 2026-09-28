package main

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/component"
)

type preflightMsg struct {
	Function func() tea.Cmd
}

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) Init() tea.Cmd {
	// Initialize your base configurations, providers, and UI sub-models first
	loadPrechecks(m.ui)
	initializeProgress(m.ui)
	m.ui.MaxParallelism = 4
	m.ui.Spinner = spinner.New()
	initializeEditor(m.ui)

	m.providerFactory = lsp.NewProviderFactory([]lsp.LanguageProvider{
		lsp.GoProvider{},
		lsp.KotlinProvider{},
		lsp.JavaProvider{},
	})

	// 🛠️ CRITICAL CONTROL FOR SEAMLESS FIRST RUNS
	if m.ui.IsFirstRun {
		m.ui.ViewState = model.StateGitConfigurationModal
		m.ui.PreviousViewState = model.StateDashboard
		return tea.Batch(textinput.Blink, m.ui.Spinner.Tick)
	}

	// If it's NOT a first run, boot up the kernel setup ticks immediately
	return tea.Batch(m.ui.Spinner.Tick, preflight(m.ui))
}

func initializeEditor(ui *model.UI) {
	ta := textarea.New()
	ta.SetHeight(10)
	ta.SetWidth(10)
	ta.ShowLineNumbers = true
	ui.Editor = ta
}

func initializeProgress(ui *model.UI) {
	ui.Progress = progress.New(
		progress.WithDefaultGradient(),
		progress.WithSolidFill(component.MochaGreen),
		progress.WithoutPercentage(),
	)
}

func loadPrechecks(ui *model.UI) {
	ui.Prechecks = []model.Precheck{
		{Name: lipgloss.NewStyle().Render("🛎 Profile "), Load: initializer.InitializeProfile(ui).OnAction},
		{Name: lipgloss.NewStyle().Render("👷 Load env"), Load: initializer.InitializeWelcomePanel(ui).OnAction},
		{Name: lipgloss.NewStyle().Render("⛵ Updating env"), Load: initializer.InitializeWelcomePanel(ui).OnAction},
		{Name: lipgloss.NewStyle().Render("🐳 Docker"), Load: initializer.InitializeDockerTelemetry(ui).OnAction},
		{Name: lipgloss.NewStyle().Render("🪁 Async"), Load: initializer.InitializeTriggerPipeline(ui).OnAction},
	}
}

func preflight(ui *model.UI) tea.Cmd {
	// Safeguard against slicing panics if the index goes boundary out of range
	if ui.Index >= len(ui.Prechecks) {
		return nil
	}

	d := time.Millisecond * time.Duration(500)
	return tea.Tick(d, func(t time.Time) tea.Msg {
		// Double check index boundary safety inside the closures
		if ui.Index >= len(ui.Prechecks) {
			return nil
		}
		precheck := ui.Prechecks[ui.Index]
		return preflightMsg{
			Function: precheck.Load,
		}
	})
}
