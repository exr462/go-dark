package component

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/align"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	alignOldStyle    = lipgloss.NewStyle().Foreground(color.Red)
	alignNewStyle    = lipgloss.NewStyle().Foreground(color.Green)
	alignTargetStyle = lipgloss.NewStyle().Foreground(color.Crust).Background(color.Green).Bold(true)
)

// RenderAlignModal draws the version alignment screen.
func RenderAlignModal(ui *model.UI) string {
	width := max(ui.WindowWidth-8, 20)
	var b strings.Builder
	b.WriteString(decorator.Title.Render("🔁 Align Workspace Versions") + "\n")
	b.WriteString(decorator.Description.Render(fmt.Sprintf(
		"Point every reference to a workspace project at one version. Commits go to branch %s in each touched repo - nothing is pushed.", align.Branch)) + "\n")
	b.WriteString(scanStatus(ui) + "\n\n")

	rows := model.AlignRows(ui.Workspace, ui.ReleaseTags)
	changes := model.AlignChanges(rows, ui.AlignChoices)
	switch {
	case ui.Workspace == nil:
	case len(rows) == 0:
		b.WriteString(decorator.Success.Render("✅ Every workspace project is referenced at its current version.") + "\n")
	default:
		selected := min(ui.SelectedAlign, len(rows)-1)
		for i, row := range rows {
			line := alignLine(ui, row)
			if i == selected {
				line = decorator.Selected.Render(">") + " " + line
			} else {
				line = "  " + line
			}
			b.WriteString(ansi.Truncate(line, width, "…") + "\n")
		}
		edits, repos := 0, len(changes)
		for _, c := range changes {
			edits += len(c)
		}
		b.WriteString("\n" + decorator.Section.Render(fmt.Sprintf("Plan: %d edit%s in %d repo%s", edits, plural(edits), repos, plural(repos))) + "\n")
		if ui.AlignPreview {
			b.WriteString(alignPreview(ui, changes, width, max(ui.WindowHeight-len(rows)-16, 4)))
		}
	}

	if len(ui.AlignResults) > 0 {
		b.WriteString("\n" + decorator.Section.Render("Last run:") + "\n")
		for _, r := range ui.AlignResults {
			name := projectName(ui, r.ProjectID)
			var line string
			switch {
			case r.OK():
				line = decorator.Success.Render(fmt.Sprintf("  ✅ %s: committed %s (%d file%s)", name, r.Commit, len(r.Files), plural(len(r.Files))))
			case r.Skipped != "":
				line = decorator.Warning.Render(fmt.Sprintf("  ⏭ %s: skipped - %s", name, r.Skipped))
			default:
				line = decorator.Error.Render(fmt.Sprintf("  ❌ %s: %v", name, r.Err))
			}
			b.WriteString(ansi.Truncate(line, width, "…") + "\n")
		}
	}

	b.WriteString("\n")
	switch {
	case ui.AlignRunning:
		b.WriteString(decorator.Warning.Render("⏳ Committing...") + "\n")
	case ui.AlignConfirm:
		downgrades := 0
		for _, cs := range changes {
			for _, c := range cs {
				if model.IsDowngrade(c) {
					downgrades++
				}
			}
		}
		if downgrades > 0 {
			b.WriteString(decorator.Error.Render(fmt.Sprintf("⚠ This includes %d downgrade%s to an older version.", downgrades, plural(downgrades))) + "\n")
		}
		b.WriteString(decorator.Warning.Render(fmt.Sprintf("Commit these edits to %d repo%s on %s? [enter] yes  [esc] no", len(changes), plural(len(changes)), align.Branch)) + "\n")
	default:
		b.WriteString(sessHintStyle.Render(fmt.Sprintf("[%s] Select | [%s] Target | [%s] Preview | [%s] Commit | [%s] Back",
			sessKeyHint.Render("↑/↓"), sessKeyHint.Render("←/→"), sessKeyHint.Render("p"), sessKeyHint.Render("enter"),
			sessKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))))
	}

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4+decorator.BorderFrame).Render(b.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}

func alignLine(ui *model.UI, row model.AlignRow) string {
	choice := ui.AlignChoices[row.ProjectID]
	target := "keep as is"
	if v := row.Target(choice); v != "" {
		target = "→ " + v
	}
	release := row.ReleaseVersion
	if release == "" {
		release = "-"
	}
	line := fmt.Sprintf("%-26s workspace %-16s release %-10s in use: %s  %s",
		projectName(ui, row.ProjectID), row.WorkspaceVersion, release, model.VersionSummary(row.Versions), alignTargetStyle.Render(" "+target+" "))
	if n := row.Downgrades(choice); n > 0 {
		line += " " + decorator.Error.Render(fmt.Sprintf("↓ %d downgrade%s", n, plural(n)))
	}
	if n := len(row.Locked); n > 0 {
		line += decorator.Meta.Render(fmt.Sprintf("  (%d not editable)", n))
	}
	return line
}

func alignPreview(ui *model.UI, changes map[string][]model.AlignChange, width, height int) string {
	var b strings.Builder
	shown := 0
	for _, id := range sortedKeys(changes) {
		dir := ""
		if p := ui.Workspace.Projects[id]; p != nil {
			dir = p.Dir
		}
		for _, d := range model.PreviewChanges(changes[id]) {
			if shown >= height {
				b.WriteString(decorator.Meta.Render("  ...") + "\n")
				return b.String()
			}
			where := decorator.Meta.Render(fmt.Sprintf("  %s/%s:%d", projectName(ui, id), model.RelPath(dir, d.File), d.Line))
			if d.Downgrade {
				where += " " + decorator.Error.Render("↓ downgrade")
			}
			b.WriteString(ansi.Truncate(where, width, "…") + "\n")
			b.WriteString(ansi.Truncate(alignOldStyle.Render("  - "+strings.TrimSpace(d.Old)), width, "…") + "\n")
			b.WriteString(ansi.Truncate(alignNewStyle.Render("  + "+strings.TrimSpace(d.New)), width, "…") + "\n")
			shown += 3
		}
	}
	return b.String()
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}

// plural is "s" unless n is 1.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
