package panel

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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

	statusDisplay := ""
	if ui.StatusMsg != "" {
		statusDisplay = fmt.Sprintf(" | %s", footerStatusStyle.Render(ui.StatusMsg))
	}

	footerText := fmt.Sprintf(" %s Help | %s Git Ops | %s Fuzzy | %s Config | %s Shortcuts | SDK: %s | Build: %s | %s%s",
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
