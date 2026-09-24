package panels

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-build/model"
)

func RenderTopMenu(m model.UIState) string {
	unfocusedBorder := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	focusedBorder := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205"))

	var topBarStrings []string
	topBarStrings = append(topBarStrings, "\x1b[41m[Enter] Backup & Clean Repo\x1b[0m")

	if len(m.Config.Projects) > 0 && m.SelectedProj < len(m.Config.Projects) && m.Config.Projects[m.SelectedProj].Type == "java" {
		topBarStrings = append(topBarStrings, "\x1b[44mExecute: mvn clean install\x1b[0m")
	}
	topBarStrings = append(topBarStrings, "\x1b[42mExecute: docker build .\x1b[0m")

	topMenuStyle := unfocusedBorder
	if m.ActiveFocus == model.FocusMenu && m.ViewState == model.StateDashboard {
		topMenuStyle = focusedBorder
	}

	return topMenuStyle.Width(m.TerminalW - 2).Render("Portal Operations Menu: " + strings.Join(topBarStrings, "  |  "))
}
