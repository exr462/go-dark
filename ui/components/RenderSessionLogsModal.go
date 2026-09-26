package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	sessCompleteStyle = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	sessRunningStyle  = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	sessCmdStyle      = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	sessHintStyle     = lipgloss.NewStyle().Foreground(color.Overlay0)
	sessKeyHint       = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderSessionLogsModal(m *model.UIState) string {
	var body strings.Builder

	body.WriteString(decorator.Title.Render("🛰️ Global Background Session Inspector Panel (Ctrl+S)") + "\n\n")
	body.WriteString(decorator.Section.Render("📋 Active Background Tracking Registry Sessions List:") + "\n")

	if len(m.Sessions) == 0 {
		body.WriteString(decorator.Meta.Render("  (No background tracking compiler sessions active on history stacks)") + "\n")
	} else {
		for id, sess := range m.Sessions {
			status := sessCompleteStyle.Render("COMPLETE")
			if sess.IsRunning {
				status = sessRunningStyle.Render("⏳ RUNNING")
			}

			lineText := fmt.Sprintf("  [%d] Project: %s ➜ Command: %s [Status: %s]", id, sess.ProjectName, sess.Command, status)
			if id == m.ViewingSessionID {
				body.WriteString(decorator.Selected.Render("> "+lineText) + "\n")
			} else {
				body.WriteString(decorator.Inactive.Render(lineText) + "\n")
			}
		}
	}
	body.WriteString("\n")

	if m.ViewingSessionID == 0 {
		body.WriteString(sessKeyHint.Render("👉 Press a digit key number [1-9] to select and view history logs context...") + "\n\n")
		body.WriteString(strings.Repeat("\n", 6))
	} else {
		sess, exists := m.Sessions[m.ViewingSessionID]
		if !exists {
			body.WriteString(decorator.Error.Render(fmt.Sprintf("❌ Error: Session profile ID index [%d] could not be found or has been scrubbed.\n\n", m.ViewingSessionID)))
			body.WriteString(strings.Repeat("\n", 6))
		} else {
			body.WriteString(fmt.Sprintf("📄 Session Logs Output targeting ID [%d]: %s\n", m.ViewingSessionID, sessCmdStyle.Render(sess.Command)))

			logHeight := max(m.WindowHeight-16, 5)
			logWidth := max(m.WindowWidth-8, 20)

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

			consoleBox := decorator.LogWindow.Width(logWidth).Height(logHeight).Render(strings.Join(lines, "\n"))
			body.WriteString(consoleBox + "\n")
		}
	}

	body.WriteString(sessHintStyle.Render(fmt.Sprintf("[%s] Return to Dashboard", sessKeyHint.Render("Esc"))))

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(m.WindowWidth-4).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
