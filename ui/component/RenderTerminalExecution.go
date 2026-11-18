package component

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	termBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color.Mauve).
			Padding(0, 1)
	termTabStyle       = lipgloss.NewStyle().Foreground(color.Subtext0)
	termActiveTabStyle = lipgloss.NewStyle().Foreground(color.Crust).Background(color.Mauve).Bold(true)
	termHintStyle      = lipgloss.NewStyle().Foreground(color.Overlay0)
	termKeyStyle       = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	termExitedStyle    = lipgloss.NewStyle().Foreground(color.Red).Bold(true)
)

// RenderTerminalExecution draws the embedded terminal: a tab bar and the
// active session's screen inside a bordered pane, plus a hint line. It
// returns where the real terminal cursor belongs (nil to hide it). The
// layout matches model.TerminalPaneLeft/Top and model.UI.TerminalPaneSize.
func RenderTerminalExecution(ui *model.UI, sessions *shell.Manager) (string, *tea.Cursor) {
	paneW, paneH := ui.TerminalPaneSize()
	active := sessions.Get(ui.ActiveTerminalSession())

	var body []string
	var cursor *tea.Cursor
	switch {
	case ui.TerminalPicking:
		body = renderTerminalPicker(ui, paneH)
	case active != nil:
		body = strings.Split(active.Render(), "\n")
		if x, y, visible := active.Cursor(); visible && active.Running() && x < paneW && y < paneH {
			cursor = tea.NewCursor(model.TerminalPaneLeft+x, model.TerminalPaneTop+y)
		}
	default:
		body = []string{decorator.Meta.Render("No terminal open.")}
	}

	lines := make([]string, 0, paneH+1)
	lines = append(lines, fitLine(renderTerminalTabs(ui, sessions), paneW))
	for i := range paneH {
		line := ""
		if i < len(body) {
			line = body[i]
		}
		lines = append(lines, fitLine(line, paneW))
	}

	box := termBoxStyle.
		Width(paneW + 2 + decorator.BorderFrame).
		Height(paneH + 1 + decorator.BorderFrame).
		Render(strings.Join(lines, "\n"))

	margin := strings.Repeat(" ", model.TerminalPaneLeft-2)
	var out strings.Builder
	out.WriteString("\n")
	for _, l := range strings.Split(box, "\n") {
		out.WriteString(margin + l + "\n")
	}
	out.WriteString(margin + ansi.Truncate(renderTerminalHint(ui, active), max(ui.WindowWidth-len(margin), 0), "…"))
	return out.String(), cursor
}

// fitLine truncates or pads an ANSI-styled line to exactly width cells.
func fitLine(line string, width int) string {
	line = ansi.Truncate(line, width, "")
	if pad := width - ansi.StringWidth(line); pad > 0 {
		line += "\x1b[m" + strings.Repeat(" ", pad)
	}
	return line + "\x1b[m"
}

func renderTerminalTabs(ui *model.UI, sessions *shell.Manager) string {
	var tabs []string
	for i, id := range ui.TerminalTabs {
		s := sessions.Get(id)
		if s == nil {
			continue
		}
		label := fmt.Sprintf(" %d %s ", i+1, s.Title)
		switch {
		case !s.Running():
			label = fmt.Sprintf(" %d %s ✗ ", i+1, s.Title)
		case s.Unseen() && i != ui.ActiveTerminalTab:
			label = fmt.Sprintf(" %d %s* ", i+1, s.Title)
		}
		if i == ui.ActiveTerminalTab && !ui.TerminalPicking {
			tabs = append(tabs, termActiveTabStyle.Render(label))
		} else {
			tabs = append(tabs, termTabStyle.Render(label))
		}
	}
	if ui.TerminalPicking {
		tabs = append(tabs, termActiveTabStyle.Render(" + new tab "))
	}
	return strings.Join(tabs, termTabStyle.Render("│"))
}

func renderTerminalPicker(ui *model.UI, height int) []string {
	lines := []string{decorator.Title.Render("Open a new terminal in:"), ""}
	if len(ui.Config.Projects) == 0 {
		return append(lines, decorator.Meta.Render("  (no projects - enter opens a shell in your home directory)"))
	}
	visible := max(height-len(lines), 1)
	start := max(min(ui.TerminalPickIndex-visible/2, len(ui.Config.Projects)-visible), 0)
	end := min(start+visible, len(ui.Config.Projects))
	for i := start; i < end; i++ {
		p := ui.Config.Projects[i]
		row := fmt.Sprintf("%s  %s", p.ID, p.Path)
		if i == ui.TerminalPickIndex {
			lines = append(lines, decorator.Selected.Render("> "+row))
		} else {
			lines = append(lines, decorator.InactivePlain.Render("  "+row))
		}
	}
	return lines
}

func renderTerminalHint(ui *model.UI, active *shell.Session) string {
	key := func(a action.Action) string {
		return termKeyStyle.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, a))
	}
	if ui.TerminalPicking {
		return termHintStyle.Render(fmt.Sprintf("[%s/%s] choose  [%s] open  [%s] cancel",
			termKeyStyle.Render("↑"), termKeyStyle.Render("↓"), termKeyStyle.Render("enter"), termKeyStyle.Render("esc")))
	}
	hint := termHintStyle.Render(fmt.Sprintf("[%s] dashboard  [%s] new tab  [%s/%s] switch  [%s] close tab",
		key(action.TerminalDetach), key(action.TerminalNewTab),
		key(action.TerminalPrevTab), key(action.TerminalNextTab), key(action.TerminalCloseTab)))
	if active != nil && !active.Running() {
		status := "process exited"
		if err := active.ExitErr(); err != nil {
			status = fmt.Sprintf("process exited: %v", err)
		}
		hint = termExitedStyle.Render(status+" - enter closes the tab") + "  " + hint
	}
	return hint
}
