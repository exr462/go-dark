package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
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
		BorderForeground(ModalBorderColor).
		Background(ModalBackground).
		Padding(1, 2, 1, 2).
		Width(m.TerminalW - 4).
		Render(modalContent)

	return lipgloss.Place(
		m.TerminalW, m.TerminalH,
		lipgloss.Center, lipgloss.Center,
		modalBox,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}
