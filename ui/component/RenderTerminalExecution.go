package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
)

var (
	// termGlassStyle is the "glass pane" of the emulator: a near-black
	// background with a rounded border, styled to look like a real Linux
	// terminal window rather than a plain modal box.
	termGlassStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#11111B")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color.Mauve).
			Padding(0, 1)

	termChromeStyle = lipgloss.NewStyle().Foreground(color.Overlay0)
	termPromptStyle = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	termOutStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
	termErrStyle    = lipgloss.NewStyle().Foreground(color.Red)
	termHintStyle   = lipgloss.NewStyle().Foreground(color.Overlay0)
	termKeyStyle    = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	termWelcomeMsg  = lipgloss.NewStyle().Foreground(color.Overlay0).Italic(true)
)

// RenderTerminalExecution draws a large, real-terminal-style interactive
// console: previously run commands and their output scroll by above a live
// shell prompt at the bottom, just like a genuine Linux terminal emulator -
// instead of two small, separately-labelled "input" and "output" boxes.
func RenderTerminalExecution(ui *model.UI) string {
	ui.Inputs[kbd.Terminal].Focus()

	projectName := "go-dark"
	if ui.SelectedGitProject >= 0 && ui.SelectedGitProject < len(ui.Config.Projects) {
		projectName = ui.Config.Projects[ui.SelectedGitProject].Name
	}

	promptPrefix := fmt.Sprintf("dev@%s:~$ ", projectName)
	ui.Inputs[kbd.Terminal].Prompt = ""

	// Use almost the entire available viewport so the console feels like a
	// real, maximized terminal window rather than a cramped popup.
	glassWidth := max(ui.WindowWidth-6, 20)
	glassHeight := max(ui.WindowHeight-4, 10)
	innerWidth := max(glassWidth-4, 16)  // minus border(2) + horizontal padding(2)
	innerHeight := max(glassHeight-2, 6) // minus border(2)

	titleBar := termChromeStyle.Render(fmt.Sprintf("● ● ●  %s — bash — %dx%d", projectName, innerWidth, innerHeight))
	hint := termHintStyle.Render(fmt.Sprintf(
		"[%s] run command   [%s] exit terminal",
		termKeyStyle.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Enter)),
		termKeyStyle.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
	))

	// Reserve rows for: title bar, blank separator, live prompt line.
	scrollbackHeight := max(innerHeight-3, 3)

	var rendered []string
	if len(ui.TerminalLogs) == 0 {
		rendered = append(rendered, termWelcomeMsg.Render("Welcome to the go-dark shell. Type a command below and press Enter to run it in the selected project's directory."))
	}
	for _, log := range ui.TerminalLogs {
		for _, wrapped := range wrapTerminalLine(log.Text, innerWidth) {
			switch {
			case log.IsPrompt:
				rendered = append(rendered, termPromptStyle.Render(wrapped))
			case log.IsErr:
				rendered = append(rendered, termErrStyle.Render(wrapped))
			default:
				rendered = append(rendered, termOutStyle.Render(wrapped))
			}
		}
	}

	startIdx := 0
	if len(rendered) > scrollbackHeight {
		startIdx = len(rendered) - scrollbackHeight
	}
	visible := append([]string{}, rendered[startIdx:]...)
	for len(visible) < scrollbackHeight {
		visible = append([]string{""}, visible...)
	}

	livePrompt := termPromptStyle.Render(promptPrefix) + ui.Inputs[kbd.Terminal].View()

	content := strings.Join([]string{
		titleBar,
		strings.Join(visible, "\n"),
		livePrompt,
	}, "\n")

	termBox := termGlassStyle.Width(innerWidth).Height(innerHeight).Render(content)

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, termBox, hint),
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}

// wrapTerminalLine hard-wraps a single log line to the console width so long
// output (stack traces, long paths, etc.) doesn't get silently truncated,
// matching how a real terminal emulator soft-wraps overflowing lines.
func wrapTerminalLine(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	if text == "" {
		return []string{""}
	}

	var lines []string
	for len(text) > width {
		lines = append(lines, text[:width])
		text = text[width:]
	}
	lines = append(lines, text)
	return lines
}
