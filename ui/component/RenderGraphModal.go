package component

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	graphWaveStyle      = lipgloss.NewStyle().Foreground(color.Lavender).Bold(true)
	graphVersionStyle   = lipgloss.NewStyle().Foreground(color.Sky)
	graphDepStyle       = lipgloss.NewStyle().Foreground(color.Subtext0)
	graphSuggestStyle   = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	graphUnconfirmStyle = lipgloss.NewStyle().Foreground(color.Peach)
	graphDriftStyle     = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
)

// graphChrome is the rows used by title, legend, hints, border and padding.
const graphChrome = 12

// RenderGraphModal draws the dependency graph: projects grouped in build
// waves with what the poms say about their dependencies.
func RenderGraphModal(ui *model.UI) string {
	width := max(ui.WindowWidth-8, 20)
	var b strings.Builder
	b.WriteString(decorator.Title.Render(fmt.Sprintf("🕸 Dependency Graph (%s)", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenGraph))) + "\n")
	b.WriteString(decorator.Description.Render("Build waves from the manifest; pom scan: ") +
		graphSuggestStyle.Render("+found in poms, not in manifest") + "  " +
		graphUnconfirmStyle.Render("?in manifest, not in poms") + "  " +
		graphDriftStyle.Render("⚠ uses an outdated version") + "\n")
	b.WriteString(scanStatus(ui) + "\n\n")

	rows, err := model.BuildGraph(ui.Config.Projects, ui.Workspace)
	switch {
	case err != nil:
		b.WriteString(decorator.Error.Render("❌ "+err.Error()) + "\n")
	case len(rows) == 0:
		b.WriteString(decorator.Meta.Render("  (No cloned projects yet)") + "\n")
	default:
		selected := min(ui.SelectedGraph, len(rows)-1)
		listRows := max(ui.WindowHeight-graphChrome, 3)
		if ui.GraphDetail {
			listRows = max(listRows/3, 3)
		}
		start := max(min(selected-listRows/2, len(rows)-listRows), 0)
		end := min(start+listRows, len(rows))
		for i := start; i < end; i++ {
			line := graphLine(ui, rows[i], i == start || rows[i].Wave != rows[i-1].Wave)
			if i == selected {
				line = decorator.Selected.Render(">") + " " + line
			} else {
				line = "  " + line
			}
			b.WriteString(ansi.Truncate(line, width, "…") + "\n")
		}
		if len(rows) > listRows {
			b.WriteString(decorator.Meta.Render(fmt.Sprintf("  %d-%d of %d", start+1, end, len(rows))) + "\n")
		}
		if ui.GraphDetail {
			b.WriteString("\n" + graphDetail(ui, rows[selected], width, max(ui.WindowHeight-graphChrome-listRows-4, 3)))
		}
	}

	b.WriteString("\n" + sessHintStyle.Render(fmt.Sprintf("[%s] Select | [%s] Pom details | [%s] Accept suggestions | [%s] Accept all | [%s] Rescan | [%s] Back",
		sessKeyHint.Render("↑/↓"), sessKeyHint.Render("enter"), sessKeyHint.Render("a"), sessKeyHint.Render("A"), sessKeyHint.Render("r"),
		sessKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))))

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4+decorator.BorderFrame).Render(b.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}

func scanStatus(ui *model.UI) string {
	switch {
	case ui.GraphScanning && ui.Workspace == nil:
		return decorator.Meta.Render("⏳ Scanning poms...")
	case ui.GraphScanning:
		return decorator.Meta.Render("⏳ Rescanning poms (showing the previous scan)...")
	case ui.Workspace == nil:
		return decorator.Meta.Render("Poms not scanned yet - press r")
	}
	msg := fmt.Sprintf("Scanned %d Maven projects, %d artifacts, %d cross-project references", len(ui.Workspace.Projects), len(ui.Workspace.Artifacts), len(ui.Workspace.Refs))
	if n := len(ui.WorkspaceErrors); n > 0 {
		msg += fmt.Sprintf(" - %d problem%s, e.g. %v", n, map[bool]string{true: "", false: "s"}[n == 1], ui.WorkspaceErrors[0])
	}
	return decorator.Meta.Render(msg)
}

func graphLine(ui *model.UI, row model.GraphRow, firstOfWave bool) string {
	wave := "   "
	if firstOfWave {
		wave = fmt.Sprintf("W%-2d", row.Wave+1)
	}
	var deps []string
	for _, d := range row.Project.DependsOn {
		deps = append(deps, projectName(ui, d))
	}
	line := fmt.Sprintf("%s %-26s %s", graphWaveStyle.Render(wave), row.Project.Name, graphVersionStyle.Render(fmt.Sprintf("%-16s", row.Version)))
	if len(deps) > 0 {
		line += graphDepStyle.Render(" → " + strings.Join(deps, ", "))
	}
	for _, s := range row.Suggested {
		line += " " + graphSuggestStyle.Render("+"+projectName(ui, s))
	}
	for _, u := range row.Unconfirmed {
		line += " " + graphUnconfirmStyle.Render("?"+projectName(ui, u))
	}
	if n := len(row.Drift); n > 0 {
		line += " " + graphDriftStyle.Render(fmt.Sprintf("⚠%d", n))
	}
	return line
}

func graphDetail(ui *model.UI, row model.GraphRow, width, height int) string {
	var b strings.Builder
	b.WriteString(decorator.Section.Render(fmt.Sprintf("📄 %s - pom references to other workspace projects:", row.Project.Name)) + "\n")
	if ui.Workspace == nil || !row.Scanned {
		b.WriteString(decorator.Meta.Render("  (not a scanned Maven project)") + "\n")
		return b.String()
	}
	refs := ui.Workspace.RefsFrom(row.Project.ID)
	if len(refs) == 0 {
		b.WriteString(decorator.Meta.Render("  (none)") + "\n")
	}
	for i, r := range refs {
		if i == height {
			b.WriteString(decorator.Meta.Render(fmt.Sprintf("  ... %d more", len(refs)-height)) + "\n")
			break
		}
		version := r.Version
		if version == "" {
			version = "(managed)"
		}
		current := ""
		if t := ui.Workspace.Projects[r.Target]; t != nil && r.Version != "" && r.Version != t.Version {
			current = graphDriftStyle.Render(fmt.Sprintf(" ⚠ %s is at %s", projectName(ui, r.Target), t.Version))
		}
		where, _ := filepath.Rel(row.Project.Path, r.POM)
		if r.Edit != nil && r.Edit.Property != "" {
			where += " via ${" + r.Edit.Property + "}"
		}
		line := fmt.Sprintf("  %-18s %-40s %-16s%s  %s", r.Kind, r.GA.ArtifactID, version, current, graphDepStyle.Render(where))
		b.WriteString(ansi.Truncate(line, width, "…") + "\n")
	}
	return b.String()
}

// projectName shows a project ID by its name when it's in the workspace.
func projectName(ui *model.UI, id string) string {
	if i := config.FindProject(ui.Config.Projects, id); i >= 0 {
		return ui.Config.Projects[i].Name
	}
	return id
}
