package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/components"
)

func RenderTopMenu(m *model.UIState) string {

	// Left Side Content: Action Links
	var leftBarStrings []string
	topBarStrings := append(leftBarStrings, "\x1b[41m[Enter] Backup & Clean\x1b[0m")

	if len(m.Config.Projects) > 0 && m.SelectedProject < len(m.Config.Projects) {
		proj := m.Config.Projects[m.SelectedProject]
		if proj.Type == "java" {
			topBarStrings = append(topBarStrings, "\x1b[44mExecute: mvn install\x1b[0m")
		}
	}
	topBarStrings = append(topBarStrings, "\x1b[42mExecute: docker build\x1b[0m")

	var activeSessionTracker string
	for id, sess := range m.Sessions {
		if sess.ProjectName == m.Config.Projects[m.SelectedProject].Name && sess.IsRunning {
			activeSessionTracker = fmt.Sprintf("  ⚡ [\x1b[33;1mSession %d\x1b[0m: COMPILING]", id)
			break
		}
	}
	leftContent := "Operations Menu: " + strings.Join(topBarStrings, " | ") + activeSessionTracker

	// !!! FIXED: COMPOSING THE TOP-RIGHT DOCKER REAL-TIME TELEMETRY PANEL HEADER !!!
	cpuVal := m.DockerTelemetry.CPU
	if cpuVal == "" {
		cpuVal = "0.0%"
	}
	memVal := m.DockerTelemetry.Memory
	if memVal == "" {
		memVal = "0B / 0B"
	}

	rightContent := fmt.Sprintf(
		"🐳 \x1b[36;1mDocker\x1b[0m ➜ Active: \x1b[32m%d\x1b[0m | CPU: \x1b[33m%s\x1b[0m | Mem: \x1b[35m%s\x1b[0m",
		m.DockerTelemetry.Running,
		cpuVal,
		memVal,
	)

	// Math calculation to compute dynamic spacing based on terminal dimensions width bounds
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	// Factor margins and padding
	spaceLen := max(m.WindowWidth-leftWidth-rightWidth-6, 2)

	unifiedTopBarText := leftContent + strings.Repeat(" ", spaceLen) + rightContent

	topMenuStyle := components.UnfocusedBorder
	if m.ActiveFocus == model.FocusMenu && m.ViewState == model.StateDashboard {
		topMenuStyle = components.FocusedBorder
	}

	return topMenuStyle.Width(m.WindowWidth - 2).Render(unifiedTopBarText)
}
