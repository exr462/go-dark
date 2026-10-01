package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	deployExecStyle   = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	deployAlertStyle  = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	deployDoneStyle   = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	deployHintStyle   = lipgloss.NewStyle().Foreground(color.Overlay0)
	deployKeyHint     = lipgloss.NewStyle().Foreground(color.Yellow)
	deployLogBoxStyle = lipgloss.NewStyle().
				Background(color.Mantle).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(color.Surface1)
)

// RenderDeployModal renders the simulated Rancher/Kubernetes workspace
// rollout screen: a dependency-aware transcript streaming into a console box.
func RenderDeployModal(ui *model.UI) string {
	var body strings.Builder

	namespace := ui.DeployNamespace
	if namespace == "" {
		namespace = "(not started)"
	}

	body.WriteString(decorator.Title.Render("🚢 Workspace Deploy: Simulated Rancher / Kubernetes Rollout") + "\n")
	body.WriteString(fmt.Sprintf("📛 Namespace: %s\n\n", deployExecStyle.Render(namespace)))

	switch {
	case ui.DeployError != nil:
		body.WriteString(deployAlertStyle.Render("❌ "+ui.DeployError.Error()) + "\n\n")
	case ui.IsDeploying:
		body.WriteString(deployAlertStyle.Render(fmt.Sprintf("⏳ Rolling out... (%d/%d steps)", len(ui.DeployLogs), len(ui.DeployScript))) + "\n\n")
	case ui.DeployDone:
		body.WriteString(deployDoneStyle.Render("✅ Rollout complete.") + "\n\n")
	default:
		body.WriteString(deployHintStyle.Render(fmt.Sprintf(
			"[%s] Start simulated rollout | [%s] Dashboard",
			deployKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			deployKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)) + "\n\n")
	}

	body.WriteString(decorator.Section.Render("📋 Rollout Transcript:") + "\n")

	logHeight := max(ui.WindowHeight-14, 5)
	logWidth := max(ui.WindowWidth-8, 20)

	logLen := len(ui.DeployLogs)
	startIdx := 0
	if logLen > logHeight {
		startIdx = logLen - logHeight
	}

	var historyLines []string
	for i := startIdx; i < logLen; i++ {
		line := ui.DeployLogs[i]
		if len(line) > logWidth {
			line = line[:logWidth-3] + "..."
		}
		historyLines = append(historyLines, line)
	}
	for len(historyLines) < logHeight {
		historyLines = append(historyLines, strings.Repeat(" ", logWidth))
	}

	consoleBox := deployLogBoxStyle.
		Width(logWidth).
		Height(logHeight).
		MaxWidth(logWidth).
		MaxHeight(logHeight).
		Render(strings.Join(historyLines, "\n"))
	body.WriteString(consoleBox)

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
