package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/renderer"
)

var (
	badgeClean   = lipgloss.NewStyle().Foreground(color.Crust).Background(color.Peach).Padding(0, 1).Bold(true)
	badgeMaven   = lipgloss.NewStyle().Foreground(color.Crust).Background(color.Lavender).Padding(0, 1).Bold(true)
	badgeDocker  = lipgloss.NewStyle().Foreground(color.Crust).Background(color.Sapphire).Padding(0, 1).Bold(true)
	menuLabel    = lipgloss.NewStyle().Foreground(color.Subtext1).Bold(true)
	dockerLabel  = lipgloss.NewStyle().Foreground(color.Sapphire).Bold(true)
	activeVal    = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	cpuValStyle  = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	memValStyle  = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	sessionAlert = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
)

func RenderTopMenu(m *model.UIState) string {
	var actionBadges []string
	actionBadges = append(actionBadges, badgeClean.Render("[Enter] Backup & Clean"))

	if len(m.Config.Projects) > 0 && m.SelectedProject < len(m.Config.Projects) {
		proj := m.Config.Projects[m.SelectedProject]
		if proj.Type == "java" {
			actionBadges = append(actionBadges, badgeMaven.Render("mvn install"))
		}
	}
	actionBadges = append(actionBadges, badgeDocker.Render("docker build"))

	var activeSessionTracker string
	for id, sess := range m.Sessions {
		if m.SelectedProject < len(m.Config.Projects) && sess.ProjectName == m.Config.Projects[m.SelectedProject].Name && sess.IsRunning {
			activeSessionTracker = sessionAlert.Render(fmt.Sprintf("  ⚡ [Session %d: COMPILING]", id))
			break
		}
	}

	leftContent := menuLabel.Render("Actions: ") + strings.Join(actionBadges, " ") + activeSessionTracker

	cpuVal := m.DockerTelemetry.CPU
	if cpuVal == "" {
		cpuVal = "0.0%"
	}
	memVal := m.DockerTelemetry.Memory
	if memVal == "" {
		memVal = "0B / 0B"
	}

	rightContent := fmt.Sprintf(
		"%s ➜ Active: %s | CPU: %s | Mem: %s",
		dockerLabel.Render("🐳 Docker"),
		activeVal.Render(fmt.Sprintf("%d", m.DockerTelemetry.Running)),
		cpuValStyle.Render(cpuVal),
		memValStyle.Render(memVal),
	)

	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	spaceLen := max(m.WindowWidth-leftWidth-rightWidth-6, 2)

	unifiedTopBarText := leftContent + strings.Repeat(" ", spaceLen) + rightContent

	topMenuStyle := renderer.UnfocusedBorder.Background(color.Base)
	if m.ActiveFocus == model.FocusMenu && m.ViewState == model.StateDashboard {
		topMenuStyle = renderer.FocusedBorder.Background(color.Base)
	}

	return topMenuStyle.Width(m.WindowWidth - 2).Render(unifiedTopBarText)
}
