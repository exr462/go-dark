package componentaction

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/kube"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
	"github.com/exr462/go-dark/task"
)

// ship runs the full train: build every buildable project in dependency
// waves (mvn install), then build the images and deploy. Version drift is
// pointed out first; a second enter ships anyway.
func ship(ui *model.UI) tea.Cmd {
	switch {
	case ui.DeployRunning || ui.ShipStage == model.ShipBuilding:
		return nil
	case pipelineRunning(ui):
		// e.g. the startup build: a second mvn run in the same directories
		// would break both.
		ui.StatusMsg = "⏳ A build is still running (startup pipeline) - wait for it, see " + binding(ui, action.OpenSession)
		return nil
	case !ui.ClusterChecked || !ui.ClusterRunning || ui.ClusterErr != nil:
		// Check before spending minutes on builds.
		ui.StatusMsg = "❌ The local cluster isn't ready - see the checks above"
		return nil
	case ui.DeployPlanErr != nil || len(ui.DeployWaves) == 0:
		ui.StatusMsg = "❌ Nothing to deploy"
		return nil
	}
	if drift := len(model.AlignRows(ui.Workspace, ui.ReleaseTags)); drift > 0 && !ui.ShipDriftAck {
		ui.ShipDriftAck = true
		ui.StatusMsg = fmt.Sprintf("⚠ %d project%s referenced at mixed versions - v to align first, enter again to ship anyway", drift, plural(drift, "", "s"))
		return nil
	}
	return func() tea.Msg { return model.ShipStartMsg{} }
}

// ShipStart begins the build stage.
func ShipStart(ui *model.UI) tea.Cmd {
	ui.ShipStage = model.ShipBuilding
	ui.DeployFinished, ui.DeployErr = false, nil
	ui.StatusMsg = "🏗️ Ship: building the workspace in dependency waves..."
	return initializer.InitializeTriggerPipeline(ui).OnAction()
}

// ShipBuilt continues the train when its build stage ends (the pipeline
// also runs on its own at startup; that run is ignored here).
func ShipBuilt(ui *model.UI, msg task.PipelineCompleteMsg) tea.Cmd {
	if ui.ShipStage != model.ShipBuilding {
		return nil
	}
	if !msg.Success {
		ui.ShipStage = model.ShipFailed
		ui.StatusMsg = "❌ Ship stopped: the build failed - " + msg.Log + buildFailureHint(ui)
		return nil
	}
	ui.ShipStage = model.ShipDeploying
	return startDeploy(ui)
}

// shipFinished closes the train after its deploy stage.
func shipFinished(ui *model.UI, err error) {
	if ui.ShipStage != model.ShipDeploying {
		return
	}
	if err != nil {
		ui.ShipStage = model.ShipFailed
		return
	}
	ui.ShipStage = model.ShipDone
	ui.ShipDriftAck = false
}

// cancelShip stops whatever stage is running: the deployment through its
// context, the build by killing the running pipeline sessions (which makes
// the pipeline abort).
func cancelShip(ui *model.UI) {
	switch {
	case ui.DeployRunning && ui.DeployCancel != nil:
		ui.DeployCancel()
		ui.StatusMsg = "⏹ Cancelling deployment..."
	case ui.ShipStage == model.ShipBuilding && ui.Shells != nil:
		for _, s := range ui.Shells.List() {
			if s.Tag == session.TagPipeline && s.Running() {
				_ = ui.Shells.Kill(s.ID)
			}
		}
		ui.StatusMsg = "⏹ Cancelling the build..."
	}
}

// pipelineRunning reports whether a pipeline build session is running.
func pipelineRunning(ui *model.UI) bool {
	if ui.Shells == nil {
		return false
	}
	for _, s := range ui.Shells.List() {
		if s.Tag == session.TagPipeline && s.Running() {
			return true
		}
	}
	return false
}

// jdkMismatch are compiler errors that mean "this JDK is too new/old for the
// project" (e.g. old Lombok on a newer JDK fails with TypeTag :: UNKNOWN).
var jdkMismatch = []string{"TypeTag :: UNKNOWN", "invalid target release", "release version", "Unsupported class file major version"}

// buildFailureHint points at the JDK profile when a failed pipeline step
// looks like a JDK mismatch.
func buildFailureHint(ui *model.UI) string {
	if ui.Shells == nil {
		return ""
	}
	for _, s := range ui.Shells.List() {
		if s.Tag != session.TagPipeline || s.Running() || s.ExitCode() == 0 {
			continue
		}
		text := s.Text()
		for _, m := range jdkMismatch {
			if strings.Contains(text, m) {
				return fmt.Sprintf(" (%s looks like a JDK mismatch - assign the project's JDK with %s)", s.ProjectName, binding(ui, action.OpenJdk))
			}
		}
	}
	return ""
}

// scaleCmd scales the other deployments in the namespace down (or restores
// them) in the background.
func scaleCmd(ui *model.UI, restore bool) tea.Cmd {
	c := cluster(ui)
	var keep []string
	for _, wave := range ui.DeployWaves {
		for _, t := range wave {
			keep = append(keep, t.ChartName)
		}
	}
	return func() tea.Msg {
		var names []string
		var err error
		if restore {
			names, err = kube.RestoreScaled(c)
		} else {
			names, err = kube.ScaleDownOthers(c, keep)
		}
		left, _ := kube.ScaledDown(c)
		return model.ScaledMsg{Restore: restore, Names: names, Left: left, Err: err}
	}
}

// Scaled stores the outcome of a scale-down / restore.
func Scaled(ui *model.UI, msg model.ScaledMsg) {
	ui.ScaledDown = msg.Left
	switch {
	case msg.Err != nil:
		ui.StatusMsg = "❌ Scaling: " + msg.Err.Error()
	case msg.Restore:
		ui.StatusMsg = fmt.Sprintf("✅ Restored %d deployment%s", len(msg.Names), plural(len(msg.Names), "", "s"))
	default:
		ui.StatusMsg = fmt.Sprintf("✅ Scaled %d other deployment%s to 0 (u restores them)", len(msg.Names), plural(len(msg.Names), "", "s"))
	}
}
