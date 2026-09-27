package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

func RenderShortcutConfigurationModal(uiState *model.UIState) string {
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
		uiState.Inputs[model.FuzzyKey].View(),
		decorator.Bold.Render("Open Git Operation  Management"),
		uiState.Inputs[model.GitOperationsKey].View(),
		decorator.Bold.Render("Open Profile"),
		uiState.Inputs[model.ProfileKey].View(),
		decorator.Bold.Render("Open Session Management"),
		uiState.Inputs[model.SessionKey].View(),
		decorator.Bold.Render("Open Docker Management"),
		uiState.Inputs[model.DockerKey].View(),
		decorator.Bold.Render("Open Maven Management"),
		uiState.Inputs[model.MvnKey].View(),
		decorator.Bold.Render("Open Jdk Management"),
		uiState.Inputs[model.JdkKey].View(),
		decorator.Bold.Render("Open Build Screen"),
		uiState.Inputs[model.BuildKey].View(),
		decorator.Bold.Render("Quit application"),
		uiState.Inputs[model.QuitKey].View(),
		decorator.Bold.Render("Edit"),
		uiState.Inputs[model.EditKey].View(),
		decorator.Bold.Render("New Project"),
		uiState.Inputs[model.NewProjectKey].View(),
		decorator.Bold.Render("Submit action"),
		uiState.Inputs[model.SubmitKey].View(),
		decorator.Bold.Render("Cancel"),
		uiState.Inputs[model.CancelKey].View(),
		decorator.Bold.Render("Open Help Menu"),
		uiState.Inputs[model.HelpKey].View(),
		decorator.Bold.Render("Toggle action"),
		uiState.Inputs[model.ToggleKey].View(),
		decorator.Bold.Render("Edit shortcuts"),
		uiState.Inputs[model.EditShortcutsKey].View(),
		installerHint.Render(fmt.Sprintf(
			"[%s] Navigate Fields | [%s] To Save | [%s] Exit",
			installerKey.Render("Tab"),
			installerKey.Render(config.GetShortcutKeyBinding(uiState.Config.ShortCuts, config.SubmitKeyBind)),
			installerKey.Render(config.GetShortcutKeyBinding(uiState.Config.ShortCuts, config.CancelKeyBind)),
		)),
	)

	return lipgloss.Place(
		uiState.WindowWidth, uiState.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(uiState.WindowWidth-4, 85)).Render(boxContent),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
