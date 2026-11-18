package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

// RenderDeployModal shows the local deployment: cluster checks, then each
// service with its image and Helm step, grouped in dependency waves.
func RenderDeployModal(ui *model.UI) string {
	width := max(ui.WindowWidth-8, 20)
	c := ui.Config.Cluster()
	var b strings.Builder
	b.WriteString(decorator.Title.Render(fmt.Sprintf("🚢 Deploy to the local cluster (%s)", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenDeploy))) + "\n")
	b.WriteString(decorator.Description.Render(fmt.Sprintf(
		"Builds each service's image into minikube's docker daemon and installs its chart with %s into %s/%s.", deploy.LocalValuesFile, c.Context, c.Namespace)) + "\n\n")

	b.WriteString(clusterLine(ui) + "\n\n")

	switch {
	case ui.DeployPlanErr != nil:
		b.WriteString(decorator.Error.Render("❌ "+ui.DeployPlanErr.Error()) + "\n")
	case len(ui.DeployWaves) == 0:
		b.WriteString(decorator.Meta.Render("  (No cloned, deployable project has a "+deploy.LocalValuesFile+")") + "\n")
	default:
		rows := max(ui.WindowHeight-18, 4)
		shown := 0
		for i, wave := range ui.DeployWaves {
			for j, t := range wave {
				if shown == rows {
					b.WriteString(decorator.Meta.Render("  ...") + "\n")
					break
				}
				label := "   "
				if j == 0 {
					label = fmt.Sprintf("W%-2d", i+1)
				}
				st := ui.DeployProgress.Get(t.Project.ID)
				line := fmt.Sprintf("%s %-24s %-44s image %s  deploy %s", graphWaveStyle.Render(label), t.Project.Name,
					t.Image+":"+t.Tag, stepBadge(st.Image), stepBadge(st.Deploy))
				if st.SessionID != 0 && (st.Image == deploy.StepFailed || st.Deploy == deploy.StepFailed) {
					line += decorator.Meta.Render(fmt.Sprintf("  log: session #%d", st.SessionID))
				}
				b.WriteString(ansi.Truncate(line, width, "…") + "\n")
				shown++
			}
		}
	}
	if n := len(ui.DeploySkipped); n > 0 {
		names := make([]string, 0, n)
		for _, t := range ui.DeploySkipped {
			names = append(names, t.Project.Name)
		}
		b.WriteString(ansi.Truncate(decorator.Meta.Render(fmt.Sprintf("  Skipped (no local chart values): %s", strings.Join(names, ", "))), width, "…") + "\n")
	}

	b.WriteString("\n")
	switch {
	case ui.DeployRunning:
		b.WriteString(decorator.Warning.Render("⏳ Deploying... the steps run as sessions (watch them with "+action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenSession)+")") + "\n")
	case ui.DeployFinished && ui.DeployErr != nil:
		b.WriteString(ansi.Truncate(decorator.Error.Render("❌ "+ui.DeployErr.Error()), width, "…") + "\n")
	case ui.DeployFinished:
		b.WriteString(decorator.Success.Render(fmt.Sprintf("✅ Deployed. kubectl --context %s -n %s get pods", c.Context, c.Namespace)) + "\n")
	}

	var hints []string
	if ui.ClusterChecked && !ui.ClusterRunning && ui.MinikubeStartID == 0 {
		hints = append(hints, fmt.Sprintf("[%s] Start minikube", sessKeyHint.Render("s")))
	}
	if ui.DeployRunning {
		hints = append(hints, fmt.Sprintf("[%s] Cancel", sessKeyHint.Render("x")))
	} else {
		hints = append(hints, fmt.Sprintf("[%s] Deploy", sessKeyHint.Render("enter")))
	}
	hints = append(hints, fmt.Sprintf("[%s] Re-check", sessKeyHint.Render("r")),
		fmt.Sprintf("[%s] Back (a deployment keeps running)", sessKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape))))
	b.WriteString(sessHintStyle.Render(strings.Join(hints, " | ")))

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4+decorator.BorderFrame).Render(b.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}

func clusterLine(ui *model.UI) string {
	c := ui.Config.Cluster()
	switch {
	case !ui.ClusterChecked:
		return decorator.Meta.Render("⏳ Checking the local cluster...")
	case ui.ClusterErr != nil:
		return decorator.Error.Render("❌ " + ui.ClusterErr.Error())
	case ui.MinikubeStartID != 0:
		return decorator.Warning.Render(fmt.Sprintf("⏳ Starting minikube (session #%d)...", ui.MinikubeStartID))
	case !ui.ClusterRunning:
		return decorator.Warning.Render("⏸ minikube is stopped - press s to start it")
	}
	return decorator.Success.Render(fmt.Sprintf("✅ %s is running and local (namespace %s)", c.Context, c.Namespace))
}

func stepBadge(s deploy.StepState) string {
	switch s {
	case deploy.StepRunning:
		return decorator.Warning.Render("⏳")
	case deploy.StepDone:
		return decorator.Success.Render("✅")
	case deploy.StepFailed:
		return decorator.Error.Render("❌")
	case deploy.StepCancelled:
		return decorator.Meta.Render("⏹")
	}
	return decorator.Meta.Render("·")
}
