package componentaction

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
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
	u := ui.Umbrella()
	targets := ui.ShipTargets()
	return func() tea.Msg {
		err := kube.Guard(c)
		running := false
		if err == nil {
			running, err = kube.Status(c)
		}
		var scaled, migrate []string
		if err == nil && running {
			scaled, _ = kube.ScaledDown(c)
			if u != nil {
				migrate, _ = deploy.PerServiceReleases(c, targets, u.Application)
			}
		}
		return model.DeployPreflightMsg{Running: running, Err: err, ScaledDown: scaled, Migrate: migrate}
	}
}

// ClusterChecked stores the preflight result.
func ClusterChecked(ui *model.UI, msg model.DeployPreflightMsg) {
	ui.ClusterChecked = true
	ui.ClusterRunning = msg.Running
	ui.ClusterErr = msg.Err
	ui.ScaledDown = msg.ScaledDown
	ui.DeployMigrate = msg.Migrate
}

// DeployModal is the ship screen: enter runs the full train (build ->
// images -> deploy), d deploys without rebuilding, up/down and space pick
// which services go into the umbrella, i rebuilds and restarts the
// highlighted one, m uninstalls old per-service releases, v aligns versions,
// s starts the local cluster when it's stopped, x cancels, r re-checks, esc
// leaves (a running train continues in the background).
func DeployModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	targets := ui.ShipTargets()
	ui.SelectedDeploy = min(max(ui.SelectedDeploy, 0), max(len(targets)-1, 0))
	switch msg.String() {
	case binding(ui, action.Escape):
		ui.ViewState = model.StateDashboard
	case "r":
		PlanDeploy(ui)
		return CheckCluster(ui)
	case "up", "k":
		ui.SelectedDeploy = max(ui.SelectedDeploy-1, 0)
	case "down", "j":
		ui.SelectedDeploy = min(ui.SelectedDeploy+1, max(len(targets)-1, 0))
	case "space":
		if ui.Umbrella() != nil && len(targets) > 0 && !ui.DeployRunning {
			toggleShip(ui, targets[ui.SelectedDeploy])
		}
	case "i":
		if len(targets) > 0 {
			return startIterate(ui, targets[ui.SelectedDeploy])
		}
	case "m":
		return migrateReleases(ui)
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

// toggleShip switches a service on or off for the umbrella and remembers
// it in config.json.
func toggleShip(ui *model.UI, t deploy.Target) {
	id := t.Project.ID
	if ui.ShipEnabled(t) {
		ui.Config.ShipDisabled = append(ui.Config.ShipDisabled, id)
	} else {
		ui.Config.ShipDisabled = slices.DeleteFunc(ui.Config.ShipDisabled, func(x string) bool { return x == id })
	}
	_ = config.SaveConfig(ui.Config)
}

// clusterReady is the common gate of every deployment.
func clusterReady(ui *model.UI) bool {
	switch {
	case ui.DeployRunning || ui.DeployMigrating:
		return false
	case !ui.ClusterChecked || !ui.ClusterRunning || ui.ClusterErr != nil:
		ui.StatusMsg = "❌ The local cluster isn't ready - see the checks above"
		return false
	case ui.DeployPlanErr != nil || len(ui.DeployWaves) == 0 || ui.Shells == nil:
		ui.StatusMsg = "❌ Nothing to deploy"
		return false
	}
	return true
}

func startDeploy(ui *model.UI) tea.Cmd {
	if !clusterReady(ui) {
		return nil
	}
	umbrella := ui.Umbrella()
	if umbrella != nil && len(ui.DeployMigrate) > 0 {
		ui.StatusMsg = fmt.Sprintf("❌ Old per-service releases are in the way (%s) - m uninstalls them first", strings.Join(ui.DeployMigrate, ", "))
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.DeployRunning, ui.DeployFinished, ui.DeployErr = true, false, nil
	ui.DeployCancel = cancel
	ui.DeployProgress = deploy.NewProgress()
	w, h := ui.TerminalPaneSize()
	opts := deploy.Options{
		Sessions: ui.Shells, Cluster: cluster(ui), Parallel: max(ui.MaxParallelism, 1),
		Width: w, Height: h, Progress: ui.DeployProgress, Umbrella: umbrella,
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

// startIterate rebuilds one service's image and restarts its deployment,
// the quick loop after a code change (no helm upgrade).
func startIterate(ui *model.UI, t deploy.Target) tea.Cmd {
	if !clusterReady(ui) {
		return nil
	}
	if ui.Umbrella() != nil && !ui.ShipEnabled(t) {
		ui.StatusMsg = "❌ " + t.Project.Name + " is switched off (space switches it on)"
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.DeployRunning, ui.DeployFinished, ui.DeployErr = true, false, nil
	ui.DeployCancel = cancel
	ui.DeployProgress = deploy.NewProgress()
	w, h := ui.TerminalPaneSize()
	opts := deploy.Options{Sessions: ui.Shells, Cluster: cluster(ui), Width: w, Height: h, Progress: ui.DeployProgress}
	ui.StatusMsg = fmt.Sprintf("🔁 %s: rebuilding the image and restarting its deployment...", t.Project.Name)
	return func() tea.Msg {
		defer cancel()
		return model.DeployFinishedMsg{Err: deploy.Iterate(ctx, t, opts)}
	}
}

// migrateReleases uninstalls the per-service releases the umbrella would
// clash with, one session each, then re-checks.
func migrateReleases(ui *model.UI) tea.Cmd {
	if len(ui.DeployMigrate) == 0 || ui.DeployMigrating || ui.DeployRunning || ui.Shells == nil || !ui.ClusterRunning {
		return nil
	}
	c := cluster(ui)
	releases := slices.Clone(ui.DeployMigrate)
	sessions := ui.Shells
	w, h := ui.TerminalPaneSize()
	ui.DeployMigrating = true
	ui.StatusMsg = fmt.Sprintf("⏳ Uninstalling %s (their resources come back with the umbrella release)...", strings.Join(releases, ", "))
	return func() tea.Msg {
		for _, r := range releases {
			id, err := sessions.Start(shell.StartOptions{ProjectName: r, Title: r + " · helm uninstall", Tag: deploy.TagShip, Width: w, Height: h, Argv: deploy.UninstallArgv(c, r)})
			if err != nil {
				return model.DeployMigratedMsg{Err: err}
			}
			s := sessions.Get(id)
			<-s.Done()
			if s.ExitCode() != 0 {
				return model.DeployMigratedMsg{Err: fmt.Errorf("helm uninstall %s exited with %d (see session #%d)", r, s.ExitCode(), id)}
			}
		}
		return model.DeployMigratedMsg{}
	}
}

// DeployMigrated reports the uninstall and re-checks the cluster.
func DeployMigrated(ui *model.UI, msg model.DeployMigratedMsg) tea.Cmd {
	ui.DeployMigrating = false
	if msg.Err != nil {
		ui.StatusMsg = "❌ " + msg.Err.Error()
	} else {
		ui.StatusMsg = "✅ Old releases uninstalled - deploy the umbrella with d or enter"
	}
	return CheckCluster(ui)
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
