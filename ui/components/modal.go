package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderModal(m model.UIState) string {
	modalContent := fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n\x1b[90m%s\x1b[0m",
		boldStyle.Foreground(Pink).Render("✨ Add New Project Configuration"),
		boldStyle.Render("1. Project Display Name:"),
		m.Inputs[3].View(),
		boldStyle.Render("2. Relative Folder Name:"),
		m.Inputs[4].View(),
		boldStyle.Render("3. Project Stack Type (java, docker):"),
		m.Inputs[5].View(),
		boldStyle.Render("4. Git Clone URL Reference:"),
		m.Inputs[6].View(),
		"[Tab] Cycle Inputs | [Enter] Confirm Save | [Esc] Cancel",
	)

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		ModalBox.Width(m.WindowWidth-4).Render(modalContent),
		whiteSpace,
		lipgloss.WithWhitespaceForeground(DarkerGrey),
	)
}
