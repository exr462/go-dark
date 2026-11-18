package componentaction

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/maven"
	"github.com/exr462/go-dark/model"
)

// OpenGraph shows the dependency graph and (re)scans the poms in the
// background; the screen shows the last scan until the new one arrives.
func OpenGraph(ui *model.UI) tea.Cmd {
	ui.ViewState = model.StateGraphModal
	ui.GraphDetail = false
	return ScanWorkspace(ui)
}

// ScanWorkspace scans every cloned project's poms in the background.
func ScanWorkspace(ui *model.UI) tea.Cmd {
	ui.GraphScanning = true
	projects := slices.Clone(ui.Config.Projects)
	return func() tea.Msg {
		w, errs := maven.Scan(projects)
		return model.WorkspaceScannedMsg{Workspace: w, Errors: errs}
	}
}

// WorkspaceScanned stores a finished pom scan.
func WorkspaceScanned(ui *model.UI, msg model.WorkspaceScannedMsg) {
	ui.Workspace = msg.Workspace
	ui.WorkspaceErrors = msg.Errors
	ui.GraphScanning = false
}

// GraphModal handles the dependency graph: up/down select, enter toggles
// the pom details, a/A accept the pom-suggested dependencies of the
// selected / every project (saved as local overrides), r rescans.
func GraphModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	rows, err := model.BuildGraph(ui.Config.Projects, ui.Workspace)
	switch msg.String() {
	case binding(ui, action.Escape):
		if ui.GraphDetail {
			ui.GraphDetail = false
			return nil
		}
		ui.ViewState = model.StateDashboard
		return nil
	case "r":
		return ScanWorkspace(ui)
	}
	if err != nil || len(rows) == 0 {
		return nil
	}
	ui.SelectedGraph = min(ui.SelectedGraph, len(rows)-1)

	switch msg.String() {
	case "up", "k":
		ui.SelectedGraph = max(ui.SelectedGraph-1, 0)
	case "down", "j":
		ui.SelectedGraph = min(ui.SelectedGraph+1, len(rows)-1)
	case "enter":
		ui.GraphDetail = !ui.GraphDetail
	case "a":
		if n := acceptSuggestions(ui, rows[ui.SelectedGraph]); n > 0 {
			ui.StatusMsg = fmt.Sprintf("✅ Added %d dependenc%s to %s", n, plural(n, "y", "ies"), rows[ui.SelectedGraph].Project.Name)
			return ApplyProjectChange(ui)
		}
	case "A":
		total := 0
		for _, row := range rows {
			total += acceptSuggestions(ui, row)
		}
		if total > 0 {
			ui.StatusMsg = fmt.Sprintf("✅ Added %d suggested dependenc%s", total, plural(total, "y", "ies"))
			return ApplyProjectChange(ui)
		}
	}
	return nil
}

// acceptSuggestions adds a row's suggested dependencies to its project and
// returns how many were added. Team projects keep them as a local override.
func acceptSuggestions(ui *model.UI, row model.GraphRow) int {
	i := config.FindProject(ui.Config.Projects, row.Project.ID)
	if i < 0 || len(row.Suggested) == 0 {
		return 0
	}
	p := &ui.Config.Projects[i]
	added := 0
	for _, s := range row.Suggested {
		if !slices.Contains(p.DependsOn, s) {
			p.DependsOn = append(slices.Clone(p.DependsOn), s)
			added++
		}
	}
	if added > 0 && !p.IsLocal() {
		p.DepsOverridden = true
	}
	return added
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
