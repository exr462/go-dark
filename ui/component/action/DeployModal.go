package componentaction

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/kube"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
)

// TagCluster marks the session that starts the local cluster's container.
const TagCluster = "cluster"

// OpenDeploy shows the local deploy screen: what would be deployed (in
// dependency waves) and whether the local cluster is ready.
func OpenDeploy(ui *model.UI) tea.Cmd {
	ui.ViewState = model.StateDeployModal
	PlanDeploy(ui)
	return tea.Batch(CheckCluster(ui), ScanWorkspace(ui))
}

// PlanDeploy resolves the deploy waves from the current projects.
func PlanDeploy(ui *model.UI) {
	_, waves, err := model.ResolveDeployOrder(ui.Config.Projects)
	ui.DeployPlanErr = err
	ui.DeployWaves, ui.DeploySkipped = deploy.Plan(waves)
	if ui.DeployProgress == nil {
		ui.DeployProgress = deploy.NewProgress()
	}
}

// cluster is the local target: the team environment, or the legacy
// config.json cluster when no manifest declares one.
func cluster(ui *model.UI) kube.Cluster { return ui.DeployTarget().Cluster }

// CheckCluster checks the local cluster and the context guard in the
// background.
func CheckCluster(ui *model.UI) tea.Cmd {
	c := cluster(ui)
	return func() tea.Msg {
		err := kube.Guard(c)
		running := false
		if err == nil {
			running, err = kube.Status(c)
		}
		var scaled []string
		if err == nil && running {
			scaled, _ = kube.ScaledDown(c)
		}
		return model.DeployPreflightMsg{Running: running, Err: err, ScaledDown: scaled}
	}
}

// ClusterChecked stores the preflight result.
func ClusterChecked(ui *model.UI, msg model.DeployPreflightMsg) {
	ui.ClusterChecked = true
	ui.ClusterRunning = msg.Running
	ui.ClusterErr = msg.Err
	ui.ScaledDown = msg.ScaledDown
}

// DeployModal is the ship screen: enter runs the full train (build ->
// images -> deploy), d deploys without rebuilding, v aligns versions, s
// starts the local cluster when it's stopped, x cancels, r re-checks, esc leaves (a
// running train continues in the background).
func DeployModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case binding(ui, action.Escape):
		ui.ViewState = model.StateDashboard
	case "r":
		PlanDeploy(ui)
		return CheckCluster(ui)
	case "s":
		if ui.ClusterChecked && !ui.ClusterRunning && ui.ClusterErr == nil && ui.ClusterStartID == 0 && ui.Shells != nil {
			c := cluster(ui)
			id, err := ui.Shells.Start(shell.StartOptions{
				ProjectName: c.Container, Title: "start " + c.Container, Tag: TagCluster,
				Argv: kube.StartArgv(c),
			})
			if err != nil {
				ui.StatusMsg = fmt.Sprintf("❌ start %s: %v", c.Container, err)
				return nil
			}
			ui.ClusterStartID = id
			ui.StatusMsg = fmt.Sprintf("⏳ Starting %s (k3s takes a minute; r re-checks)...", c.Container)
		}
	case "x":
		cancelShip(ui)
	case "v":
		if !ui.DeployRunning && ui.ShipStage != model.ShipBuilding {
			return OpenAlign(ui, model.StateDeployModal)
		}
	case "d":
		if pipelineRunning(ui) {
			ui.StatusMsg = "⏳ A build is still running - deploy when it's done"
			return nil
		}
		if ui.ShipStage != model.ShipBuilding {
			ui.ShipStage = model.ShipIdle
			return startDeploy(ui)
		}
	case "z":
		if ui.ClusterRunning && ui.ClusterErr == nil && !ui.DeployRunning {
			ui.StatusMsg = "⏳ Scaling the other deployments to 0..."
			return scaleCmd(ui, false)
		}
	case "u":
		if ui.ClusterRunning && ui.ClusterErr == nil && !ui.DeployRunning {
			ui.StatusMsg = "⏳ Restoring scaled-down deployments..."
			return scaleCmd(ui, true)
		}
	case "enter", binding(ui, action.Save):
		return ship(ui)
	}
	return nil
}

func startDeploy(ui *model.UI) tea.Cmd {
	switch {
	case ui.DeployRunning:
		return nil
	case !ui.ClusterChecked || !ui.ClusterRunning || ui.ClusterErr != nil:
		ui.StatusMsg = "❌ The local cluster isn't ready - see the checks above"
		return nil
	case ui.DeployPlanErr != nil || len(ui.DeployWaves) == 0 || ui.Shells == nil:
		ui.StatusMsg = "❌ Nothing to deploy"
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.DeployRunning, ui.DeployFinished, ui.DeployErr = true, false, nil
	ui.DeployCancel = cancel
	ui.DeployProgress = deploy.NewProgress()
	w, h := ui.TerminalPaneSize()
	opts := deploy.Options{
		Sessions: ui.Shells, Cluster: cluster(ui), Parallel: max(ui.MaxParallelism, 1),
		Width: w, Height: h, Progress: ui.DeployProgress,
	}
	envFor := ui.DeployEnv
	waves := ui.DeployWaves
	ui.StatusMsg = "🚢 Deploying to the local cluster..."
	return func() tea.Msg {
		defer cancel()
		arch, err := kube.NodeArch(opts.Cluster)
		if err != nil {
			return model.DeployFinishedMsg{Err: err}
		}
		opts.Env = envFor(arch)
		return model.DeployFinishedMsg{Err: deploy.Run(ctx, waves, opts)}
	}
}

// DeployFinished stores the outcome of a run.
func DeployFinished(ui *model.UI, msg model.DeployFinishedMsg) {
	ui.DeployRunning, ui.DeployFinished, ui.DeployErr, ui.DeployCancel = false, true, msg.Err, nil
	shipFinished(ui, msg.Err)
	if msg.Err != nil {
		ui.StatusMsg = fmt.Sprintf("❌ Deploy failed: %v", msg.Err)
	} else {
		c := cluster(ui)
		ui.StatusMsg = fmt.Sprintf("✅ Deployed to %s/%s", c.Context, c.Namespace)
	}
}

// ClusterStartExited re-checks the cluster after its start session ends.
func ClusterStartExited(ui *model.UI, id int) tea.Cmd {
	if id != ui.ClusterStartID {
		return nil
	}
	ui.ClusterStartID = 0
	return CheckCluster(ui)
}
