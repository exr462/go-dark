package component

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

func RenderShortcutConfigurationModal(ui *model.UI) string {
	var boxContent string
	boxContent = fmt.Sprintf(
		// Title
		"%s\n\n"+
			// FuzzyKey
			"%s\n%s\n\n"+
			// GitOperationsKey
			"%s\n%s\n\n"+
			// ProfileKey
			"%s\n%s\n\n"+
			// SessionKey
			"%s\n%s\n\n"+
			// DockerKey
			"%s\n%s\n\n"+
			// MvnKey
			"%s\n%s\n\n"+
			// JdkKey
			"%s\n%s\n\n"+
			// BuildKey
			"%s\n%s\n\n"+
			// QuitKey
			"%s\n%s\n\n"+
			// EditKey
			"%s\n%s\n\n"+
			// NewProjectKey
			"%s\n%s\n\n"+
			// SubmitKey
			"%s\n%s\n\n"+
			// CancelKey
			"%s\n%s\n\n"+
			// HelpKey
			"%s\n%s\n\n"+
			// ToggleKey
			"%s\n%s\n\n"+
			// EditShortcutsKey
			"%s\n%s\n\n"+
			// Footer
			"%s\n",
		decorator.Title.Render("🔏 Shortcut configuration screen"),
		decorator.Bold.Render("Open Fuzzy Finder"),
		ui.Inputs[kbd.FuzzyKey].View(),
		decorator.Bold.Render("Open Git Operation  Management"),
		ui.Inputs[kbd.GitOperationsKey].View(),
		decorator.Bold.Render("Open Profile"),
		ui.Inputs[kbd.ProfileKey].View(),
		decorator.Bold.Render("Open Session Management"),
		ui.Inputs[kbd.SessionKey].View(),
		decorator.Bold.Render("Open Docker Management"),
		ui.Inputs[kbd.DockerKey].View(),
		decorator.Bold.Render("Open Maven Management"),
		ui.Inputs[kbd.MvnKey].View(),
		decorator.Bold.Render("Open Jdk Management"),
		ui.Inputs[kbd.JdkKey].View(),
		decorator.Bold.Render("Open Build Screen"),
		ui.Inputs[kbd.BuildKey].View(),
		decorator.Bold.Render("Quit application"),
		ui.Inputs[kbd.QuitKey].View(),
		decorator.Bold.Render("Edit"),
		ui.Inputs[kbd.EditKey].View(),
		decorator.Bold.Render("New Project"),
		ui.Inputs[kbd.NewProjectKey].View(),
		decorator.Bold.Render("Submit action"),
		ui.Inputs[kbd.SubmitKey].View(),
		decorator.Bold.Render("Cancel"),
		ui.Inputs[kbd.CancelKey].View(),
		decorator.Bold.Render("Open Help Menu"),
		ui.Inputs[kbd.HelpKey].View(),
		decorator.Bold.Render("Toggle action"),
		ui.Inputs[kbd.ToggleKey].View(),
		decorator.Bold.Render("Edit shortcuts"),
		ui.Inputs[kbd.EditShortcutsKey].View(),
		installerHint.Render(fmt.Sprintf(
			"[%s] Navigate Fields | [%s] To Save | [%s] Exit",
			installerKey.Render("Tab"),
			installerKey.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			installerKey.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)),
	)

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(ui.WindowWidth-4, 85)).Render(boxContent),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
