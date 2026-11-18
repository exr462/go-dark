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

// shortcutModalChrome is the rows taken by the title, hints, border and
// padding; the rest of the window shows shortcut fields (3 rows each).
const shortcutModalChrome = 9

func RenderShortcutConfigurationModal(ui *model.UI) string {
	focused := 0
	for i, f := range kbd.ShortcutFields {
		if f.Field == ui.FocusedInput {
			focused = i
		}
	}

	// Window the list around the focused field so it fits any terminal.
	visible := max((ui.WindowHeight-shortcutModalChrome)/3, 1)
	start := max(min(focused-visible/2, len(kbd.ShortcutFields)-visible), 0)
	end := min(start+visible, len(kbd.ShortcutFields))

	var b strings.Builder
	b.WriteString(decorator.Title.Render("🔏 Shortcut configuration screen") + "\n\n")
	for _, f := range kbd.ShortcutFields[start:end] {
		b.WriteString(decorator.Bold.Render(f.Label) + "\n")
		b.WriteString(ui.Inputs[f.Field].View() + "\n\n")
	}
	b.WriteString(installerHint.Render(fmt.Sprintf(
		"[%s] Navigate Fields (%d/%d) | [%s] To Save | [%s] Exit",
		installerKey.Render("Tab"), focused+1, len(kbd.ShortcutFields),
		installerKey.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
		installerKey.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
	)) + "\n")

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(ui.WindowWidth-4, 85)+decorator.BorderFrame).Render(b.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}
