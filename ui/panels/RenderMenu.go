package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	menuLabel    = lipgloss.NewStyle().Foreground(color.Subtext1).Bold(true)
	dockerLabel  = lipgloss.NewStyle().Foreground(color.Rosewater).Bold(true)
	activeVal    = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	cpuValStyle  = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	memValStyle  = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	sessionAlert = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
)

func RenderMenu(m *model.UIState) string {
	var activeSessionTracker string
	for id, sess := range m.Sessions {
		if m.SelectedProject < len(m.Config.Projects) && sess.ProjectName == m.Config.Projects[m.SelectedProject].Name && sess.IsRunning {
			activeSessionTracker = sessionAlert.Render(fmt.Sprintf("  ⚡ [Session %d: COMPILING]", id))
			break
		}
	}

	cpuVal := m.DockerTelemetry.CPU
	if cpuVal == "" {
		cpuVal = "0.0%"
	}
	memVal := m.DockerTelemetry.Memory
	if memVal == "" {
		memVal = "0B / 0B"
	}

	leftContent := fmt.Sprintf(
		" %s ➜ Active: %s | CPU: %s | Mem: %s",
		dockerLabel.Render("🐳 Docker"),
		activeVal.Render(fmt.Sprintf("%d", m.DockerTelemetry.Running)),
		cpuValStyle.Render(cpuVal),
		memValStyle.Render(memVal),
	)

	rightContent := menuLabel.Render("Git Username: ") + m.Config.GitUsername + " " + menuLabel.Render("Git Email: ") + m.Config.GitEmail + " " + menuLabel.Render("Active Sessions: ") + activeSessionTracker

	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	spaceLen := max(m.WindowWidth-leftWidth-rightWidth-6, 2)

	unifiedTopBarText := leftContent + strings.Repeat(" ", spaceLen) + rightContent

	topMenuStyle := decorator.UnfocusedBorder.Background(color.Base)
	if m.ActiveFocus == model.FocusMenu && m.ViewState == model.StateDashboard {
		topMenuStyle = decorator.FocusedBorder.Background(color.Base)
	}

	return topMenuStyle.Width(m.WindowWidth - 2).Render(unifiedTopBarText)
}
