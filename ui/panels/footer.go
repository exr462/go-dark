package panels

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/components"
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
	var sessionStatusStr = "\x1b[90mNo active background sessions\x1b[0m"
	if activeCount > 0 {
		sessionStatusStr = fmt.Sprintf("⚡ \x1b[33;1mBackground Active: %d Running\x1b[0m", activeCount)
	}
	footerText := fmt.Sprintf(" Press [?] for Help | [Ctrl+F] Fuzzy Find | Bound: %s | %s | Status: %s", boundJDK, sessionStatusStr, state.StatusMsg)
	return lipgloss.NewStyle().Background(components.DarkerGrey).Foreground(components.Grey).Width(state.WindowWidth).Render(footerText)
}
