package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	addProjHint = lipgloss.NewStyle().Foreground(color.Overlay0)
	addProjKey  = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderModal(m *model.UIState) string {
	//goland:noinspection GoPrintFunctions
	modalContent := fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		decorator.Title.Render("✨ Add New Project Configuration"),
		decorator.Bold.Render("1. Project Display Name:"),
		m.Inputs[model.ProjectName].View(),
		decorator.Bold.Render("2. Relative Folder Name:"),
		m.Inputs[model.RelativeFolder].View(),
		decorator.Bold.Render("3. Project Stack Type (java, docker):"),
		m.Inputs[model.StackType].View(),
		decorator.Bold.Render("4. Git Clone URL Reference:"),
		m.Inputs[model.GitCloneURL].View(),
		m.Inputs[model.ProjectName].Focus(),
		addProjHint.Render(fmt.Sprintf(
			"[%s] Cycle Inputs | [%s] Confirm Save | [%s] Cancel",
			addProjKey.Render("Tab"),
			addProjKey.Render("Enter"),
			addProjKey.Render("Esc"),
		)),
	)

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(m.WindowWidth-4, 85)).Render(modalContent),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
