package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-build/model"
)

func RenderTopMenu(m model.UIState) string {
	unfocusedBorder := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	focusedBorder := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205"))

	var topBarStrings []string
	topBarStrings = append(topBarStrings, "\x1b[41m[Enter] Backup & Clean Repo\x1b[0m")

	if len(m.Config.Projects) > 0 && m.SelectedProj < len(m.Config.Projects) {
		proj := m.Config.Projects[m.SelectedProj]
		if proj.Type == "java" {
			topBarStrings = append(topBarStrings, "\x1b[44mExecute: mvn clean install\x1b[0m")
		}
	}
	topBarStrings = append(topBarStrings, "\x1b[42mExecute: docker build .\x1b[0m")

	// !!! NEW: DYNAMICALLY ADVERTISE RUNNING BACKGROUND SESSION ID IN THE HEADER MENU !!!
	var activeSessionTracker string
	for id, sess := range m.Sessions {
		if sess.ProjectName == m.Config.Projects[m.SelectedProj].Name && sess.IsRunning {
			activeSessionTracker = fmt.Sprintf("  ⚡ [\x1b[33;1mSession %d\x1b[0m: COMPILING]", id)
			break
		}
	}

	topMenuStyle := unfocusedBorder
	if m.ActiveFocus == model.FocusMenu && m.ViewState == model.StateDashboard {
		topMenuStyle = focusedBorder
	}

	return topMenuStyle.Width(m.TerminalW - 2).Render("Portal Operations Menu: " + strings.Join(topBarStrings, "  |  ") + activeSessionTracker)
}
