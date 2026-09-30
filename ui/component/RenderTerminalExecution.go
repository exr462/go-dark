package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

// RenderTerminalExecution draws a cross-platform command entry and log monitoring workspace.
func RenderTerminalExecution(ui *model.UI) string {
	// 1. Setup specialized styling for terminal/log streams
	logBoxStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1E1E2E")). // Deep dark terminal backing
		Padding(0, 1).
		MarginBottom(1).
		Width(min(ui.WindowWidth-8, 81)).
		Height(10) // Fixed height block with text boundaries

	// 2. Format and render background command log entries
	var logContent strings.Builder
	if len(ui.TerminalLogs) == 0 {
		logContent.WriteString(decorator.Dim.Render("No active process logs. Enter a command above to begin..."))
	} else {
		for _, log := range ui.TerminalLogs {
			if log.IsErr {
				logContent.WriteString(decorator.Red.Render(log.Text) + "\n")
			} else {
				logContent.WriteString(decorator.White.Render(log.Text) + "\n")
			}
		}
	}

	// 3. Assemble the internal layout content flow
	var boxContent string
	boxContent = fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		decorator.Title.Render("💻 Multi-Platform Console Command Cockpit"),
		decorator.Bold.Render("1. Target Shell Command (Windows/Linux/Mac compatible):"),
		ui.Inputs[kbd.Terminal].View(), // Expected key field definition
		decorator.Bold.Render("2. Real-Time Output Console:"),
		logBoxStyle.Render(logContent.String()),
		installerHint.Render(fmt.Sprintf(
			"[%s] Execute Code | [%s] Abort Process | [%s] Exit Screen",
			installerKey.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Execute)),
			installerKey.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Abort)),
			installerKey.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)),
	)

	// 4. Center-align the workspace using your platform's placement engine
	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(ui.WindowWidth-4, 85)).Render(boxContent),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
