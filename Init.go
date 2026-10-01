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
	ta.ShowLineNumbers = false

	// RenderOpenEditor draws its own gutter/highlighting straight from
	// ta.Value() and ta.LineInfo() - it never calls ta.View(). That means
	// ta's own soft-wrapping must never kick in, or LineInfo().CharOffset
	// (used to position the cursor overlay) ends up relative to a wrapped
	// *segment* instead of the real line, silently misplacing every typed
	// character and making keys like End/"$"/"A" look like they don't work.
	// A generously large width guarantees no code line ever wraps.
	ta.SetWidth(editorTextareaWidth)
	ta.SetHeight(editorTextareaHeight)
	ui.Editor = ta
}

// editorTextareaWidth/Height intentionally far exceed any realistic terminal
// size so the embedded textarea's internal soft-wrap/viewport logic never
// engages - RenderOpenEditor does all real scrolling/line-wrapping itself.
const (
	editorTextareaWidth  = 4096
	editorTextareaHeight = 4096
)

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
		// NOTE: previously this ran twice ("Load env" + "Updating env") calling the
		// exact same initializer with no behavioral difference - trimmed to one step.
		{Name: lipgloss.NewStyle().Render("👷 Load env"), Load: initializer.InitializeWelcomePanel(ui).OnAction},
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
