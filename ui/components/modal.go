package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-build/model"
)

func RenderModal(m model.UIState) string {
	modalContent := fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n\x1b[90m%s\x1b[0m",
		lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("✨ Add New Project Configuration"),
		lipgloss.NewStyle().Bold(true).Render("1. Project Display Name:"),
		m.Inputs[3].View(),
		lipgloss.NewStyle().Bold(true).Render("2. Relative Folder Name:"),
		m.Inputs[4].View(),
		lipgloss.NewStyle().Bold(true).Render("3. Project Stack Type (java, docker):"),
		m.Inputs[5].View(),
		lipgloss.NewStyle().Bold(true).Render("4. Git Clone URL Reference:"),
		m.Inputs[6].View(),
		"[Tab] Cycle Inputs | [Enter] Confirm Save | [Esc] Cancel",
	)

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("205")).
		Background(lipgloss.Color("234")).
		Padding(1, 4, 1, 4).
		Width(75).
		Render(modalContent)

	return lipgloss.Place(
		m.TerminalW, m.TerminalH,
		lipgloss.Center, lipgloss.Center,
		modalBox,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}
