package panel

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
)

func RenderFooter(ui *model.UI) string {
	var boundJDK string
	var boundMaven string
	if len(ui.Config.Projects) > 0 && ui.SelectedProject < len(ui.Config.Projects) {
		project := ui.Config.Projects[ui.SelectedProject]
		boundJDK = project.JDKName
		boundMaven = project.MavenName
	}
	if boundJDK == "" {
		boundJDK = "N/A"
	}
	if boundMaven == "" {
		boundMaven = "N/A"
	}

	var activeCount int
	for _, s := range ui.Sessions {
		if s.IsRunning {
			activeCount++
		}
	}

	var sessionStatusStr = footerMutedStyle.Render("Idle (no sessions)")
	if activeCount > 0 {
		sessionStatusStr = footerActiveStyle.Render(fmt.Sprintf("⚡ Active: %d running", activeCount))
	}
	// Shown first: the footer gets cut off on narrow windows, and running
	// shells are easy to forget about.
	shellsStr := ""
	if shells := len(ui.TerminalTabs); shells > 0 {
		shellsStr = footerActiveStyle.Render(fmt.Sprintf(" 🖥 %d shell%s [%s] |", shells, plural(shells),
			action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenTerminal)))
	}

	statusDisplay := ""
	if ui.StatusMsg != "" {
		statusDisplay = fmt.Sprintf(" | %s", footerStatusStyle.Render(ui.StatusMsg))
	}

	footerText := shellsStr + fmt.Sprintf(" %s Help | %s Git Ops | %s Fuzzy | %s Config | %s Shortcuts | SDK: %s | Build: %s | %s%s",
		footerKeyStyle.Render(fmt.Sprintf("[%s]", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenHelp))),
		footerKeyStyle.Render(fmt.Sprintf("[%s]", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenGitOperations))),
		footerKeyStyle.Render(fmt.Sprintf("[%s]", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenFuzzy))),
		footerKeyStyle.Render(fmt.Sprintf("[%s]", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenConfiguration))),
		footerKeyStyle.Render(fmt.Sprintf("[%s]", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenEditShortcuts))),
		footerBoundStyle.Render(boundJDK),
		footerBoundStyle.Render(boundMaven),
		sessionStatusStr,
		statusDisplay,
	)

	return lipgloss.NewStyle().
		Background(color.Mantle).
		Foreground(color.Subtext0).
		Width(ui.WindowWidth).
		Render(footerText)
}

var (
	footerKeyStyle    = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	footerActiveStyle = lipgloss.NewStyle().Foreground(color.Peach).Bold(true)
	footerMutedStyle  = lipgloss.NewStyle().Foreground(color.Overlay0)
	footerStatusStyle = lipgloss.NewStyle().Foreground(color.Teal)
	footerBoundStyle  = lipgloss.NewStyle().Foreground(color.Lavender)
)

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
