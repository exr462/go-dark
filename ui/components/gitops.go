package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderGitOpsModal(m model.UIState) string {
	var modalBody strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	modalBody.WriteString(titleStyle.Render("⚡ Git Hub Central Control (Ctrl+G)") + "\n\n")

	if m.GitOpsStep == model.StepSelectGitProject {
		modalBody.WriteString("👉 Step 1: Select Target Project Repository:\n\n")
		for i, p := range m.Config.Projects {
			if i == m.SelectedGitProj {
				modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> %s (%s)", p.Name, p.GitURL)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  %s", p.Name)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate | [Enter] Select Command | [Esc] Exit\x1b[0m")
	} else {
		targetProj := m.Config.Projects[m.SelectedGitProj]
		modalBody.WriteString(fmt.Sprintf("📦 Project Workspace: %s\n", targetProj.Name))
		modalBody.WriteString("👉 Step 2: Choose Git Operation to Execute:\n\n")

		for i, cmd := range m.GitCommands {
			if i == m.SelectedGitCmd {
				modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> git %s", cmd)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  git %s", cmd)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate | [Enter] Execute | [Esc] Back to Projects\x1b[0m")
	}

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ModalBorderColor).
		Background(ModalBackground).
		Padding(1, 2, 1, 2).
		Width(m.TerminalW - 4).
		Render(modalBody.String())

	return lipgloss.Place(
		m.TerminalW, m.TerminalH,
		lipgloss.Center, lipgloss.Center,
		modalBox,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}
