package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

func RenderGitConfiguration(m *model.UIState) string {
	var boxContent string
	boxContent = fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		decorator.Title.Render("🚀 Global Preferences Setup (Step 1 of 2)"),
		decorator.Bold.Render("1. Global Workspace Base Path:"),
		m.Inputs[model.GitWorkspace].View(),
		decorator.Bold.Render("2. Global Git Username (For your code commits):"),
		m.Inputs[model.GitUsername].View(),
		decorator.Bold.Render("3. Global Git Email Address:"),
		m.Inputs[model.GitEmail].View(),
		decorator.Bold.Render("4. Max Tag List Size:"),
		m.Inputs[model.GitMaxTagListSize].View(),
		installerHint.Render(fmt.Sprintf(
			"[%s] Navigate Fields | [%s] To Save | [%s] Exit",
			installerKey.Render("Tab"),
			installerKey.Render("Enter"),
			installerKey.Render("Esc"),
		)),
	)

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(m.WindowWidth-4, 85)).Render(boxContent),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
