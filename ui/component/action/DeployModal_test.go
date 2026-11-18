package componentaction

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

// testKeyMsg builds a tea.KeyPressMsg whose String() matches the given key
// representation (e.g. "esc", "ctrl+s", or a single printable rune).
func testKeyMsg(key string) tea.KeyPressMsg {
	switch key {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+s":
		return tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	default:
		return tea.KeyPressMsg{Code: []rune(key)[0], Text: key}
	}
}

func newDeployTestUI() *model.UI {
	return &model.UI{
		Config: config.Config{
			ShortCuts: action.DefaultShortcuts,
			Projects: []config.Project{
				{ID: "rail/parent", Name: "parent", Fetched: true},
				{ID: "rail/til-purchase", Name: "til-purchase", Fetched: true, Deployable: true, DependsOn: []string{"rail/parent"}},
			},
		},
	}
}

func TestStartDeploy_PopulatesScriptAndStartsStreaming(t *testing.T) {
	ui := newDeployTestUI()

	cmd := startDeploy(ui)
	if cmd == nil {
		t.Fatal("expected a tick command to be returned")
	}
	if !ui.IsDeploying {
		t.Fatal("expected IsDeploying to be true after starting a deploy")
	}
	if ui.DeployDone {
		t.Fatal("expected DeployDone to be false right after starting")
	}
	if len(ui.DeployScript) == 0 {
		t.Fatal("expected a non-empty pre-computed rollout script")
	}
	if ui.DeployNamespace == "" {
		t.Fatal("expected a namespace to be assigned")
	}
}

func TestStartDeploy_NoDeployableProjectsSetsError(t *testing.T) {
	ui := newDeployTestUI()
	ui.Config.Projects = []config.Project{{Name: "parent", Fetched: true}}

	startDeploy(ui)
	if ui.DeployError == nil {
		t.Fatal("expected an error when no deployable projects exist")
	}
	if ui.IsDeploying {
		t.Fatal("expected IsDeploying to stay false on a planning error")
	}
}

func TestDeployTick_StreamsEntireScriptThenFinishes(t *testing.T) {
	ui := newDeployTestUI()
	startDeploy(ui)

	total := len(ui.DeployScript)
	steps := 0
	for ui.IsDeploying && steps <= total+1 {
		DeployTick(ui)
		steps++
	}

	if ui.IsDeploying {
		t.Fatal("expected streaming to finish within the expected number of ticks")
	}
	if !ui.DeployDone {
		t.Fatal("expected DeployDone to be true once streaming finishes")
	}
	if len(ui.DeployLogs) != total {
		t.Fatalf("expected all %d lines revealed, got %d", total, len(ui.DeployLogs))
	}
	for i := range ui.DeployLogs {
		if ui.DeployLogs[i] != ui.DeployScript[i] {
			t.Fatalf("revealed line %d does not match script: got %q want %q", i, ui.DeployLogs[i], ui.DeployScript[i])
		}
	}
}

func TestDeployModal_EscapeCancelsAndReturnsToDashboard(t *testing.T) {
	ui := newDeployTestUI()
	startDeploy(ui)
	ui.ViewState = model.StateDeployModal

	DeployModal(ui, testKeyMsg("esc"))

	if ui.IsDeploying {
		t.Fatal("expected Escape to stop an in-flight deploy")
	}
	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected ViewState to return to Dashboard, got %v", ui.ViewState)
	}
}
