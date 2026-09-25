package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderDockerModal(m model.UIState) string {
	var body strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("36")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	thStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)

	body.WriteString(titleStyle.Render("🐳 Docker Infrastructure Control Center (Ctrl+D)") + "\n")
	body.WriteString("Monitor and orchestrate local microservice containers across your daemon runtime layers.\n")
	body.WriteString(strings.Repeat("─", max(m.TerminalW-8, 20)) + "\n\n")

	// Print Table Grid Headers
	body.WriteString(fmt.Sprintf(
		"  %-12s %-25s %-20s %-20s\n",
		thStyle.Render("CONTAINER ID"), thStyle.Render("NAMES"), thStyle.Render("IMAGE"), thStyle.Render("STATUS"),
	))
	body.WriteString(strings.Repeat("╌", max(m.TerminalW-8, 20)) + "\n")

	if len(m.DockerContainers) == 0 {
		body.WriteString("  \x1b[90m(No docker container contexts active or discovered on your machine daemon)\x1b[0m\n")
	} else {
		for i, c := range m.DockerContainers {
			statusColor := "\x1b[31m" // Default Red for stopped
			if strings.HasPrefix(strings.ToLower(c.Status), "up") {
				statusColor = "\x1b[32m" // Green for Running
			}

			rowText := fmt.Sprintf(
				"  %-12s %-25s %-20s %s%-20s\x1b[0m",
				c.ID, c.Names, c.Image, statusColor, c.Status,
			)

			// Restrict horizontal lengths to stay within bounds gracefully
			if len(rowText) > m.TerminalW-6 {
				rowText = rowText[:m.TerminalW-9] + "..."
			}

			if i == m.SelectedDockerRow {
				body.WriteString(selectedStyle.Render("> "+rowText) + "\n")
			} else {
				body.WriteString(inactiveStyle.Render(rowText) + "\n")
			}
		}
	}

	body.WriteString("\n" + strings.Repeat("─", max(m.TerminalW-8, 20)) + "\n")
	body.WriteString("\x1b[226;1m[s] Start Container  |  [t] Stop Container  |  [r] Restart  |  [Esc] Dashboard\x1b[0m\n")

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("36")).
		Background(lipgloss.Color("234")).
		Padding(1, 2, 1, 2).
		Width(m.TerminalW - 4).
		Render(body.String())

	return lipgloss.Place(m.TerminalW, m.TerminalH, lipgloss.Center, lipgloss.Center, modalBox)
}
