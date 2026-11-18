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

// TagMinikube marks the `minikube start` session.
const TagMinikube = "minikube"

// OpenDeploy shows the local deploy screen: what would be deployed (in
// dependency waves) and whether the local cluster is ready.
func OpenDeploy(ui *model.UI) tea.Cmd {
	ui.ViewState = model.StateDeployModal
	PlanDeploy(ui)
	return CheckCluster(ui)
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

func cluster(ui *model.UI) kube.Cluster {
	c := ui.Config.Cluster()
	return kube.Cluster{Context: c.Context, Namespace: c.Namespace}
}

// CheckCluster checks minikube and the context guard in the background.
func CheckCluster(ui *model.UI) tea.Cmd {
	c := cluster(ui)
	return func() tea.Msg {
		running, err := kube.MinikubeStatus()
		if err == nil {
			err = kube.Guard(c)
		}
		return model.DeployPreflightMsg{Running: running, Err: err}
	}
}

// ClusterChecked stores the preflight result.
func ClusterChecked(ui *model.UI, msg model.DeployPreflightMsg) {
	ui.ClusterChecked = true
	ui.ClusterRunning = msg.Running
	ui.ClusterErr = msg.Err
}

// DeployModal: enter deploys, s starts minikube when it's stopped, x
// cancels a running deployment, r re-checks the cluster, esc leaves (a
// running deployment continues in the background).
func DeployModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case binding(ui, action.Escape):
		ui.ViewState = model.StateDashboard
	case "r":
		PlanDeploy(ui)
		return CheckCluster(ui)
	case "s":
		if ui.ClusterChecked && !ui.ClusterRunning && ui.MinikubeStartID == 0 && ui.Shells != nil {
			id, err := ui.Shells.Start(shell.StartOptions{
				ProjectName: "minikube", Title: "minikube start", Tag: TagMinikube,
				Argv: []string{"minikube", "start"},
			})
			if err != nil {
				ui.StatusMsg = fmt.Sprintf("❌ minikube start: %v", err)
				return nil
			}
			ui.MinikubeStartID = id
			ui.StatusMsg = "⏳ Starting minikube (watch it in the session inspector)..."
		}
	case "x":
		if ui.DeployRunning && ui.DeployCancel != nil {
			ui.DeployCancel()
			ui.StatusMsg = "⏹ Cancelling deployment..."
		}
	case "enter", binding(ui, action.Save):
		return startDeploy(ui)
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
	waves := ui.DeployWaves
	ui.StatusMsg = "🚢 Deploying to the local cluster..."
	return func() tea.Msg {
		defer cancel()
		env, err := kube.DockerEnv()
		if err != nil {
			return model.DeployFinishedMsg{Err: err}
		}
		opts.DockerEnv = env
		return model.DeployFinishedMsg{Err: deploy.Run(ctx, waves, opts)}
	}
}

// DeployFinished stores the outcome of a run.
func DeployFinished(ui *model.UI, msg model.DeployFinishedMsg) {
	ui.DeployRunning, ui.DeployFinished, ui.DeployErr, ui.DeployCancel = false, true, msg.Err, nil
	if msg.Err != nil {
		ui.StatusMsg = fmt.Sprintf("❌ Deploy failed: %v", msg.Err)
	} else {
		ui.StatusMsg = fmt.Sprintf("✅ Deployed to %s/%s", ui.Config.Cluster().Context, ui.Config.Cluster().Namespace)
	}
}

// MinikubeExited re-checks the cluster after `minikube start` ends.
func MinikubeExited(ui *model.UI, id int) tea.Cmd {
	if id != ui.MinikubeStartID {
		return nil
	}
	ui.MinikubeStartID = 0
	return CheckCluster(ui)
}
