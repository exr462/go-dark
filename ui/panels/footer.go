package panels

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
)

var (
	footerKeyStyle    = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	footerActiveStyle = lipgloss.NewStyle().Foreground(color.Peach).Bold(true)
	footerMutedStyle  = lipgloss.NewStyle().Foreground(color.Overlay0)
	footerStatusStyle = lipgloss.NewStyle().Foreground(color.Teal)
	footerBoundStyle  = lipgloss.NewStyle().Foreground(color.Lavender)
)

func CreateFooter(state *model.UIState) string {
	var boundJDK string
	if len(state.Config.Projects) > 0 && state.SelectedProject < len(state.Config.Projects) {
		boundJDK = state.Config.Projects[state.SelectedProject].JDKName
	}
	if boundJDK == "" {
		boundJDK = "System Default"
	}

	var activeCount int
	for _, s := range state.Sessions {
		if s.IsRunning {
			activeCount++
		}
	}

	var sessionStatusStr = footerMutedStyle.Render("Idle (no sessions)")
	if activeCount > 0 {
		sessionStatusStr = footerActiveStyle.Render(fmt.Sprintf("⚡ Active: %d running", activeCount))
	}

	statusDisplay := ""
	if state.StatusMsg != "" {
		statusDisplay = fmt.Sprintf(" | %s", footerStatusStyle.Render(state.StatusMsg))
	}

	footerText := fmt.Sprintf(
		" %s Help | %s Git Ops | %s Fuzzy | %s Config | Bound: %s | %s%s",
		footerKeyStyle.Render("[?]"),
		footerKeyStyle.Render("[Ctrl+G]"),
		footerKeyStyle.Render("[Ctrl+F]"),
		footerKeyStyle.Render("[Ctrl+Y]"),
		footerBoundStyle.Render(boundJDK),
		sessionStatusStr,
		statusDisplay,
	)

	return lipgloss.NewStyle().
		Background(color.Mantle).
		Foreground(color.Subtext0).
		Width(state.WindowWidth).
		Render(footerText)
}
