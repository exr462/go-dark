package componentaction

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/model"
)

// deployTickInterval controls how fast the simulated rollout "streams" its
// pre-computed transcript into the UI - fast enough to feel responsive, slow
// enough to actually read as it scrolls by.
const deployTickInterval = 120 * time.Millisecond

// DeployModal drives the simulated Rancher/Kubernetes workspace rollout
// screen: Save (ctrl+s) computes the dependency-ordered deploy plan and
// streams it line-by-line; Escape cancels/leaves.
//
//goland:noinspection GoMixedReceiverTypes
func DeployModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.IsDeploying = false
		ui.ViewState = model.StateDashboard
		return nil
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		if ui.IsDeploying {
			return nil
		}
		return startDeploy(ui)
	}
	return nil
}

// startDeploy computes the full simulated rollout transcript up-front (pure,
// deterministic, unit-testable via deploy.Plan) and kicks off the streamed
// reveal.
func startDeploy(ui *model.UI) tea.Cmd {
	namespace := ui.DeployNamespace
	if namespace == "" {
		namespace = deploy.DefaultNamespace
	}

	entries, err := deploy.Plan(ui.Config.Projects, config.AvailableProjects, namespace)
	if err != nil {
		ui.DeployError = err
		ui.DeployDone = true
		ui.StatusMsg = "❌ Deploy: " + err.Error()
		return nil
	}

	ui.DeployNamespace = namespace
	ui.DeployScript = deploy.Lines(entries)
	ui.DeployLogs = nil
	ui.DeployError = nil
	ui.DeployDone = false
	ui.IsDeploying = true
	return deployTick()
}

// DeployTick reveals the next pre-computed line of the simulated rollout and
// schedules the next tick, or marks the simulation as finished once the
// whole transcript has been streamed.
func DeployTick(ui *model.UI) tea.Cmd {
	if !ui.IsDeploying {
		return nil
	}
	if len(ui.DeployLogs) >= len(ui.DeployScript) {
		ui.IsDeploying = false
		ui.DeployDone = true
		ui.StatusMsg = "🚢 Simulated workspace rollout complete."
		return nil
	}

	ui.DeployLogs = append(ui.DeployLogs, ui.DeployScript[len(ui.DeployLogs)])
	return deployTick()
}

func deployTick() tea.Cmd {
	return tea.Tick(deployTickInterval, func(time.Time) tea.Msg { return deploy.TickMsg{} })
}
