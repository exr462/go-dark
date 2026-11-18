package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

// RenderTeamsModal lists the subscribed team manifests and the add form.
func RenderTeamsModal(ui *model.UI) string {
	var body strings.Builder
	body.WriteString(decorator.Title.Render("🛰 Team Manifests") + "\n")
	body.WriteString(decorator.Description.Render("Each team keeps a go-dark.json manifest in a git repository. Its projects are merged into your workspace.") + "\n")
	body.WriteString(strings.Repeat("─", max(ui.WindowWidth-8, 20)) + "\n\n")

	if len(ui.Config.Teams) == 0 {
		body.WriteString(decorator.Meta.Render("  (No team manifests yet - press a to subscribe to one)") + "\n")
	}
	for i, t := range ui.Config.Teams {
		manifest := t.Manifest
		if manifest == "" {
			manifest = "not synced yet"
		}
		row := fmt.Sprintf("%s  [%s]  %s", t.Name, manifest, t.GitURL)
		if i == ui.SelectedTeam && !ui.AddingTeam {
			body.WriteString(decorator.Selected.Render("> "+row) + "\n")
		} else {
			body.WriteString(decorator.InactivePlain.Render("  "+row) + "\n")
		}
	}

	body.WriteString("\n")
	var hint string
	if ui.AddingTeam {
		body.WriteString(decorator.Bold.Render("Team manifest git URL:") + "\n")
		body.WriteString(ui.Inputs[kbd.TeamGitURL].View() + "\n\n")
		hint = fmt.Sprintf("[%s] Subscribe | [%s] Cancel",
			deckKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Enter)),
			deckKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))
	} else {
		hint = fmt.Sprintf("[%s] Navigate | [%s] Subscribe | [%s] Sync all | [%s] Unsubscribe | [%s] Back",
			deckKeyHint.Render("↑/↓"), deckKeyHint.Render("a"), deckKeyHint.Render("s"), deckKeyHint.Render("d"),
			deckKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))
	}
	if ui.StatusMsg != "" {
		body.WriteString(decorator.Info.Render(ui.StatusMsg) + "\n")
	}
	body.WriteString(deckHintStyle.Render(hint) + "\n")

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4+decorator.BorderFrame).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}
