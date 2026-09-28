package component

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

func RenderHelpModal(ui *model.UI) string {
	var body strings.Builder
	body.WriteString(decorator.Title.Render("📖 Go-Dark Command & Shortcuts Cheat Sheet") + "\n\n")

	body.WriteString(decorator.Section.Render("🌐 GLOBAL DASHBOARD CONTROLS") + "\n")
	body.WriteString(decorator.Row(kbd.Tab, "Cycle active panel focus (Projects ➜ Tree ➜ Menu)"))
	body.WriteString(decorator.Row("↑/↓/j/k", "Navigate items in the currently focused panel"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save), "In Tree: expand/collapse folder or preview file | In Menu: run action"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenEditFile), "Open selected tree file in external editor (nvim)"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenHelp), "Toggle this Help modal on/off"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.QuitApplication), "Quit application"))
	body.WriteString("\n")

	body.WriteString(decorator.Section.Render("🎛️ MODAL INTERFACE SHORTCUTS") + "\n")
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenGitOperations), "Git Ops Center: Checkout predefined projects, pull, branch, status"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenFuzzy), "Fuzzy Finder: Search filenames and deep code contents (Ctrl+T toggles mode)"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenConfiguration), "Config Deck: Centralized environment & project workspace control"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenNewProject), "Add Project: Register a new workspace configuration"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenBuild), "Build Flight Deck: Launch Maven / Docker background build pipelines"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenSession), "Session Inspector: Track and unpack background compiler log streams"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenDocker), "Docker Control: Monitor container status, start/stop/restart"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenJdk), "JDK Manager: Register JAVA_HOME profiles and bind to projects"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenMvn), "Maven Manager: Register MAVEN_HOME profiles and bind to projects"))
	body.WriteString("\n")

	body.WriteString(decorator.Section.Render("✍️ DIALOGS & FORMS NAVIGATION") + "\n")
	body.WriteString(decorator.Row("Tab / Down", "Move cursor focus forward to next field"))
	body.WriteString(decorator.Row("Shift+Tab / Up", "Move cursor focus backward to previous field"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save), "Confirm selection / submit form data"))
	body.WriteString(decorator.Row(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape), "Go back one level / close active modal safely") + "\n\n")

	body.WriteString(decorator.Author.Render("🤖 Author: Daniel Noulet © 2026"))

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(ui.WindowWidth-4, 90)).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
