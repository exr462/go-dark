package panel

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

func RenderMenu(ui *model.UI) string {
	var activeSessionTracker string
	for id, sess := range ui.Sessions {
		if ui.SelectedProject < len(ui.Config.Projects) && sess.ProjectName == ui.Config.Projects[ui.SelectedProject].Name && sess.IsRunning {
			activeSessionTracker = sessionAlert.Render(fmt.Sprintf("  ⚡ [Session %d: COMPILING]", id))
			break
		}
	}

	cpuVal := ui.DockerTelemetry.CPU
	if cpuVal == "" {
		cpuVal = "0.0%"
	}
	memVal := ui.DockerTelemetry.Memory
	if memVal == "" {
		memVal = "0B / 0B"
	}

	leftContent := fmt.Sprintf(
		" %s ➜ Active: %s | CPU: %s | Mem: %s",
		dockerLabel.Render("🐳 Docker"),
		activeVal.Render(fmt.Sprintf("%d", ui.DockerTelemetry.Running)),
		cpuValStyle.Render(cpuVal),
		memValStyle.Render(memVal),
	)

	rightContent := menuLabel.Render("Git Username: ") + ui.Config.GitUsername + " " + menuLabel.Render("Git Email: ") + ui.Config.GitEmail + " " + menuLabel.Render("Active Sessions: ") + activeSessionTracker

	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	spaceLen := max(ui.WindowWidth-leftWidth-rightWidth-6, 2)

	unifiedTopBarText := leftContent + strings.Repeat(" ", spaceLen) + rightContent

	topMenuStyle := decorator.UnfocusedBorder.Background(color.Base)
	if ui.ActiveFocus == model.FocusMenu && ui.ViewState == model.StateDashboard {
		topMenuStyle = decorator.FocusedBorder.Background(color.Base)
	}

	return topMenuStyle.Width(ui.WindowWidth - 2).Render(unifiedTopBarText)
}
