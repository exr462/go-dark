package componentaction

import (
	"slices"
	"testing"

	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func newDependencyTestUI(t *testing.T) *model.UI {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	projects := []config.Project{
		{ID: "local/tool", Source: config.LocalSource, Name: "tool"},
		{ID: "rail/api", Source: "rail", Name: "api", DependsOn: []string{"rail/parent"}},
		{ID: "rail/parent", Source: "rail", Name: "parent"},
	}
	ui := &model.UI{
		Config:    config.Config{ShortCuts: action.DefaultShortcuts, Projects: projects},
		ViewState: model.StateDependencyConfigModal,
	}
	// Edit "rail/api", which is NOT at the same index as anything in a
	// registry - the screen must work purely by ID.
	ui.DepScreen = model.NewDependencyScreen(projects[1], projects)
	return ui
}

func TestDependencyScreen_ToggleAndSaveMarksTeamOverride(t *testing.T) {
	ui := newDependencyTestUI(t)

	DependencyScreen(ui, testKeyMsg("space")) // cursor 0 = local/tool
	DependencyScreen(ui, testKeyMsg("ctrl+s"))

	api := ui.Config.Projects[1]
	if !slices.Equal(api.DependsOn, []string{"rail/parent", "local/tool"}) {
		t.Fatalf("expected local/tool to be added, got %v", api.DependsOn)
	}
	if !api.DepsOverridden {
		t.Fatal("editing a team project's dependencies must mark them as a local override")
	}
	if ui.ViewState != model.StateDashboard {
		t.Fatal("save must return to the dashboard")
	}
}

func TestDependencyScreen_EscapeDiscardsEdits(t *testing.T) {
	ui := newDependencyTestUI(t)

	DependencyScreen(ui, testKeyMsg("space"))
	DependencyScreen(ui, testKeyMsg("esc"))

	if api := ui.Config.Projects[1]; !slices.Equal(api.DependsOn, []string{"rail/parent"}) || api.DepsOverridden {
		t.Fatalf("escape must restore the original dependencies, got %+v", api)
	}
}

func TestDependencyScreen_ToggleOffRemovesDependency(t *testing.T) {
	ui := newDependencyTestUI(t)

	DependencyScreen(ui, testKeyMsg("down")) // cursor 1 = rail/parent
	DependencyScreen(ui, testKeyMsg("space"))

	if deps := ui.Config.Projects[1].DependsOn; len(deps) != 0 {
		t.Fatalf("expected rail/parent to be removed, got %v", deps)
	}
}
