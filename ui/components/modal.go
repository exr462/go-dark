package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/renderer"
)

func RenderModal(m *model.UIState) string {
	modalContent := fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n\x1b[90m%s\x1b[0m",
		renderer.Bold.Foreground(color.Pink).Render("✨ Add New Project Configuration"),
		renderer.Bold.Render("1. Project Display Name:"),
		m.Inputs[3].View(),
		renderer.Bold.Render("2. Relative Folder Name:"),
		m.Inputs[4].View(),
		renderer.Bold.Render("3. Project Stack Type (java, docker):"),
		m.Inputs[5].View(),
		renderer.Bold.Render("4. Git Clone URL Reference:"),
		m.Inputs[6].View(),
		"[Tab] Cycle Inputs | [Enter] Confirm Save | [Esc] Cancel",
	)

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		renderer.ModalBox.Width(m.WindowWidth-4).Render(modalContent),
		renderer.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.DarkerGrey),
	)
}
