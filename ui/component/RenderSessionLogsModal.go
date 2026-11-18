package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	sessCompleteStyle = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	sessFailedStyle   = lipgloss.NewStyle().Foreground(color.Red).Bold(true)
	sessRunningStyle  = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	sessCmdStyle      = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	sessHintStyle     = lipgloss.NewStyle().Foreground(color.Overlay0)
	sessKeyHint       = lipgloss.NewStyle().Foreground(color.Yellow)
)

// sessionListRows caps how many sessions are listed above the preview.
const sessionListRows = 8

// RenderSessionLogsModal lists every session (shells, builds, pipeline
// steps) with a live preview of the highlighted one.
func RenderSessionLogsModal(ui *model.UI) string {
	var body strings.Builder
	body.WriteString(decorator.Title.Render(fmt.Sprintf("🛰️ Session Inspector (%s)", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenSession))) + "\n\n")

	var sessions []*shell.Session
	if ui.Shells != nil {
		sessions = ui.Shells.List()
	}
	logWidth := max(ui.WindowWidth-8, 20)

	if len(sessions) == 0 {
		body.WriteString(decorator.Meta.Render("  (No sessions yet - builds, pipeline steps and terminal shells show up here)") + "\n\n")
	} else {
		selected := min(ui.SelectedSession, len(sessions)-1)
		start := max(min(selected-sessionListRows/2, len(sessions)-sessionListRows), 0)
		end := min(start+sessionListRows, len(sessions))
		for i := start; i < end; i++ {
			s := sessions[i]
			row := fmt.Sprintf("#%-3d %-10s %-40s %s", s.ID, sessionKind(s), s.Title, sessionStatus(s))
			if i == selected {
				body.WriteString(decorator.Selected.Render("> "+ansi.Strip(row)) + "\n")
			} else {
				body.WriteString(decorator.InactivePlain.Render("  "+row) + "\n")
			}
		}
		if len(sessions) > sessionListRows {
			body.WriteString(decorator.Meta.Render(fmt.Sprintf("  %d-%d of %d", start+1, end, len(sessions))) + "\n")
		}

		s := sessions[selected]
		body.WriteString("\n" + fmt.Sprintf("📄 %s\n", sessCmdStyle.Render(ansi.Truncate(s.Command, logWidth, "…"))))
		logHeight := max(ui.WindowHeight-sessionListRows-18, 5)
		lines := tailLines(s.Text(), logHeight)
		for i, l := range lines {
			lines[i] = colorizeLogLine(ansi.Truncate(l, logWidth, "…"))
		}
		for len(lines) < logHeight {
			lines = append(lines, "")
		}
		body.WriteString(decorator.LogWindow.Width(logWidth+decorator.BorderFrame).Height(logHeight+decorator.BorderFrame).Render(strings.Join(lines, "\n")) + "\n")
	}

	body.WriteString(sessHintStyle.Render(fmt.Sprintf("[%s] Select | [%s] Open in terminal | [%s] Close session | [%s] Dashboard",
		sessKeyHint.Render("↑/↓"), sessKeyHint.Render("enter"), sessKeyHint.Render("d"),
		sessKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))))

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4+decorator.BorderFrame).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}

func sessionKind(s *shell.Session) string {
	if s.Tag != "" {
		return s.Tag
	}
	if s.Interactive {
		return "shell"
	}
	return "command"
}

func sessionStatus(s *shell.Session) string {
	switch {
	case s.Running():
		return sessRunningStyle.Render("⏳ RUNNING")
	case s.ExitCode() == 0:
		return sessCompleteStyle.Render("✅ DONE")
	default:
		return sessFailedStyle.Render(fmt.Sprintf("❌ EXIT %d", s.ExitCode()))
	}
}
