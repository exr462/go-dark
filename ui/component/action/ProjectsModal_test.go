package componentaction

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

func newProjectsTestUI(t *testing.T, projects ...config.Project) *model.UI {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return &model.UI{
		Config:    config.Config{BasePath: "/ws", ShortCuts: action.DefaultShortcuts, Projects: projects},
		ViewState: model.StateProjectsModal,
		Inputs:    initializer.MakeInputs(),
	}
}

func typeText(ui *model.UI, text string) {
	for _, r := range text {
		ProjectsModal(ui, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func TestProjectsModal_AddsLocalProjectWithChosenType(t *testing.T) {
	ui := newProjectsTestUI(t)

	ProjectsModal(ui, testKeyMsg("a"))
	typeText(ui, "web")
	for range 3 { // name -> git URL -> path -> type row
		ProjectsModal(ui, tea.KeyPressMsg{Code: tea.KeyTab})
	}
	ProjectsModal(ui, testKeyMsg("l")) // maven -> gradle
	ProjectsModal(ui, testKeyMsg("l")) // gradle -> npm
	ProjectsModal(ui, testKeyMsg("ctrl+s"))

	i := config.FindProject(ui.Config.Projects, "local/web")
	if i < 0 {
		t.Fatalf("expected local/web to be added, got %+v (status %q)", ui.Config.Projects, ui.StatusMsg)
	}
	p := ui.Config.Projects[i]
	if p.Type != "npm" || p.Path != "/ws/web" || !p.Buildable || !p.IsLocal() {
		t.Fatalf("unexpected project %+v", p)
	}
	if ui.AddingProject {
		t.Fatal("the form must close after saving")
	}
	saved, _ := config.LoadConfig()
	if config.FindProject(saved.Projects, "local/web") < 0 {
		t.Fatal("the new project must be persisted")
	}
}

func TestProjectsModal_RejectsInvalidNames(t *testing.T) {
	ui := newProjectsTestUI(t, config.Project{ID: "local/web", Source: config.LocalSource, Name: "web"})

	for _, name := range []string{"", "a/b", "web"} {
		ProjectsModal(ui, testKeyMsg("a"))
		typeText(ui, name)
		ProjectsModal(ui, testKeyMsg("ctrl+s"))
		if len(ui.Config.Projects) != 1 || !ui.AddingProject {
			t.Fatalf("name %q must be rejected, projects=%+v", name, ui.Config.Projects)
		}
		ProjectsModal(ui, testKeyMsg("esc"))
	}
}

func TestProjectsModal_RemovesOnlyLocalProjects(t *testing.T) {
	ui := newProjectsTestUI(t,
		config.Project{ID: "local/tool", Source: config.LocalSource, Name: "tool"},
		config.Project{ID: "rail/api", Source: "rail", Name: "api"},
	)
	// "rail" has no readable manifest but is subscribed, so its project stays.
	ui.Config.Teams = []config.Team{{Name: "rail-team", Manifest: "rail"}}

	ui.SelectedManagedProject = 1
	ProjectsModal(ui, testKeyMsg("d"))
	if len(ui.Config.Projects) != 2 {
		t.Fatal("team projects must not be removable")
	}

	ui.SelectedManagedProject = 0
	ProjectsModal(ui, testKeyMsg("d"))
	if config.FindProject(ui.Config.Projects, "local/tool") >= 0 || len(ui.Config.Projects) != 1 {
		t.Fatalf("expected local/tool to be removed, got %+v", ui.Config.Projects)
	}
}

func TestTeamsModal_SubscribeReturnsCommandAndClosesForm(t *testing.T) {
	ui := newProjectsTestUI(t)
	ui.ViewState = model.StateTeamsModal

	TeamsModal(ui, testKeyMsg("a"))
	if !ui.AddingTeam || ui.FocusedInput != kbd.TeamGitURL {
		t.Fatal("a must open the subscribe form")
	}
	for _, r := range "git@host:rail/team.git" {
		TeamsModal(ui, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	cmd := TeamsModal(ui, testKeyMsg("enter"))

	if cmd == nil || ui.AddingTeam {
		t.Fatal("enter must start the subscription and close the form")
	}
}
