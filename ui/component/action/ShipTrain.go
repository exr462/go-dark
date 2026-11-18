package componentaction

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/initializer"
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
		ui.StatusMsg = "❌ Ship stopped: the build failed - " + msg.Log
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
