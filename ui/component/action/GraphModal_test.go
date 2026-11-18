package componentaction

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/component"
)

func graphTestUI(t *testing.T) *model.UI {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	poms := map[string]string{
		"parent": `<project><groupId>g</groupId><artifactId>parent</artifactId><version>3.5.8</version></project>`,
		"auth":   `<project><parent><groupId>g</groupId><artifactId>parent</artifactId><version>3.5.8</version></parent><artifactId>auth</artifactId><version>1.0</version></project>`,
		"svc": `<project><parent><groupId>g</groupId><artifactId>parent</artifactId><version>3.4.9</version></parent><artifactId>svc</artifactId><version>2.0</version>
			<dependencies><dependency><groupId>g</groupId><artifactId>auth</artifactId><version>1.0</version></dependency></dependencies></project>`,
	}
	for name, pom := range poms {
		_ = os.MkdirAll(filepath.Join(root, name), 0o755)
		_ = os.WriteFile(filepath.Join(root, name, "pom.xml"), []byte(pom), 0o644)
	}
	return &model.UI{
		Config: config.Config{
			ShortCuts: action.DefaultShortcuts,
			Teams:     []config.Team{{Name: "rail-team", Manifest: "rail"}}, // keeps rail projects without a manifest file
			Projects: []config.Project{
				{ID: "rail/parent", Source: "rail", Name: "parent", Path: filepath.Join(root, "parent"), Fetched: true},
				{ID: "rail/auth", Source: "rail", Name: "auth", Path: filepath.Join(root, "auth"), Fetched: true, DependsOn: []string{"rail/parent"}},
				{ID: "rail/svc", Source: "rail", Name: "svc", Path: filepath.Join(root, "svc"), Fetched: true, DependsOn: []string{"rail/parent"}},
			},
		},
		WindowWidth:  140,
		WindowHeight: 40,
	}
}

func openAndScan(t *testing.T, ui *model.UI) {
	t.Helper()
	cmd := OpenGraph(ui)
	if ui.ViewState != model.StateGraphModal || !ui.GraphScanning || cmd == nil {
		t.Fatal("opening the graph must start a scan")
	}
	WorkspaceScanned(ui, cmd().(model.WorkspaceScannedMsg))
	if ui.GraphScanning || ui.Workspace == nil {
		t.Fatal("the scan result must be stored")
	}
}

func TestGraph_RendersWavesAndMarkers(t *testing.T) {
	ui := graphTestUI(t)
	openAndScan(t, ui)

	out := ansi.Strip(component.RenderGraphModal(ui))

	for _, want := range []string{"W1", "W2", "+auth", "⚠1", "3.5.8", "Scanned 3 Maven projects"} {
		if !strings.Contains(out, want) {
			t.Errorf("graph is missing %q:\n%s", want, out)
		}
	}
}

func TestGraph_DetailListsReferencesWithDrift(t *testing.T) {
	ui := graphTestUI(t)
	openAndScan(t, ui)
	rows, _ := model.BuildGraph(ui.Config.Projects, ui.Workspace)
	for i, r := range rows {
		if r.Project.ID == "rail/svc" {
			ui.SelectedGraph = i
		}
	}

	GraphModal(ui, testKeyMsg("enter"))
	out := ansi.Strip(component.RenderGraphModal(ui))

	if !ui.GraphDetail || !strings.Contains(out, "⚠ parent is at 3.5.8") || !strings.Contains(out, "pom.xml") {
		t.Fatalf("detail must list svc's references with the drift:\n%s", out)
	}
	GraphModal(ui, testKeyMsg("esc"))
	if ui.GraphDetail || ui.ViewState != model.StateGraphModal {
		t.Fatal("esc closes the detail first")
	}
	GraphModal(ui, testKeyMsg("esc"))
	if ui.ViewState != model.StateDashboard {
		t.Fatal("a second esc returns to the dashboard")
	}
}

func TestGraph_AcceptSuggestionsPersistsAsOverride(t *testing.T) {
	ui := graphTestUI(t)
	openAndScan(t, ui)
	rows, _ := model.BuildGraph(ui.Config.Projects, ui.Workspace)
	for i, r := range rows {
		if r.Project.ID == "rail/svc" {
			ui.SelectedGraph = i
		}
	}

	GraphModal(ui, testKeyMsg("a"))

	svc := ui.Config.Projects[config.FindProject(ui.Config.Projects, "rail/svc")]
	if !slices.Equal(svc.DependsOn, []string{"rail/parent", "rail/auth"}) || !svc.DepsOverridden {
		t.Fatalf("expected auth added as a local override, got %+v", svc)
	}
	saved, _ := config.LoadConfig()
	if i := config.FindProject(saved.Projects, "rail/svc"); i < 0 || !slices.Contains(saved.Projects[i].DependsOn, "rail/auth") {
		t.Fatalf("the accepted dependency must be saved, got %+v", saved.Projects)
	}
	if !strings.Contains(ui.StatusMsg, "Added 1 dependency to svc") {
		t.Fatalf("unexpected status %q", ui.StatusMsg)
	}

	// Nothing left to accept: A is a no-op.
	GraphModal(ui, testKeyMsg("A"))
	if strings.Contains(ui.StatusMsg, "suggested") {
		t.Fatal("accepting with no suggestions must not report anything")
	}
}
