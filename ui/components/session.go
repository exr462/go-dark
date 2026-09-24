package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderSessionLogsModal(m model.UIState) string {
	var body strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	logWindowStyle := lipgloss.NewStyle().Background(lipgloss.Color("233")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))

	body.WriteString(titleStyle.Render("🛰️ Global Background Session Inspector Panel (Ctrl+S)") + "\n\n")

	body.WriteString("📋 Active Background Tracking Registry Sessions List:\n")
	if len(m.Sessions) == 0 {
		body.WriteString("  \x1b[90m(No background tracking compiler sessions active on history stacks)\x1b[0m\n")
	} else {
		for id, sess := range m.Sessions {
			status := "\x1b[32mCOMPLETE\x1b[0m"
			if sess.IsRunning {
				status = "⏳ \x1b[33mRUNNING\x1b[0m"
			}

			// Highlight the active session line if the user is currently viewing it
			lineText := fmt.Sprintf("  [%d] Project: %s ➜ Command: %s [Status: %s]", id, sess.ProjectName, sess.Command, status)
			if id == m.ViewingSessionID {
				body.WriteString(selectedStyle.Render("> "+lineText) + "\n")
			} else {
				body.WriteString(inactiveStyle.Render(lineText) + "\n")
			}
		}
	}
	body.WriteString("\n")

	if m.ViewingSessionID == 0 {
		body.WriteString("👉 \x1b[226;1mPress a digit key number button [1-9] to select and unpack history logs context...\x1b[0m\n\n")
		body.WriteString(strings.Repeat("\n", 6))
	} else {
		sess, exists := m.Sessions[m.ViewingSessionID]
		if !exists {
			body.WriteString(fmt.Sprintf("❌ Error: Session profile ID index [%d] could not be found or has been scrubbed.\n\n", m.ViewingSessionID))
			body.WriteString(strings.Repeat("\n", 6))
		} else {
			body.WriteString(fmt.Sprintf("誠 Session Logs Output Trail targeting ID [%d]: \x1b[35;1m%s\x1b[0m\n", m.ViewingSessionID, m.Sessions[m.ViewingSessionID].Command))

			logHeight := max(m.TerminalH-16, 5)
			logWidth := max(m.TerminalW-8, 20)

			var lines []string
			startIdx := 0
			if len(sess.Logs) > logHeight {
				startIdx = len(sess.Logs) - logHeight
			}
			for i := startIdx; i < len(sess.Logs); i++ {
				l := sess.Logs[i]
				if len(l) > logWidth {
					l = l[:logWidth-3] + "..."
				}
				lines = append(lines, l)
			}
			for len(lines) < logHeight {
				lines = append(lines, strings.Repeat(" ", logWidth))
			}

			consoleBox := logWindowStyle.Width(logWidth).Height(logHeight).Render(strings.Join(lines, "\n"))
			body.WriteString(consoleBox + "\n")
		}
	}

	body.WriteString("\x1b[90m[Esc] Close Inspector View Panel and return safely to dashboard structures portal\x1b[0m")

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ModalBorderColor).
		Background(ModalBackground).
		Padding(1, 2, 1, 2).
		Width(m.TerminalW - 4).
		Render(body.String())

	return lipgloss.Place(m.TerminalW, m.TerminalH, lipgloss.Center, lipgloss.Center, modalBox)
}
