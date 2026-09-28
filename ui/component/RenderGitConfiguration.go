package component

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	installerHint = lipgloss.NewStyle().Foreground(color.Overlay0)
	installerKey  = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderGitConfiguration(ui *model.UI) string {
	var boxContent string
	boxContent = fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		decorator.Title.Render("🚀 Global Preferences Setup (Step 1 of 2)"),
		decorator.Bold.Render("1. Global Workspace Base Path:"),
		ui.Inputs[model.GitWorkspace].View(),
		decorator.Bold.Render("2. Global Git Username (For your code commits):"),
		ui.Inputs[model.GitUsername].View(),
		decorator.Bold.Render("3. Global Git Email Address:"),
		ui.Inputs[model.GitEmail].View(),
		decorator.Bold.Render("4. Max Tag List Size:"),
		ui.Inputs[model.GitMaxTagListSize].View(),
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
