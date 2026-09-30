package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
)

func BuildModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	if len(ui.Config.Projects) == 0 {
		ui.ViewState = model.StateDashboard
		return nil
	}

	targetProj := ui.Config.Projects[ui.SelectedProject]
	var currentSessionID int
	for id, sess := range ui.Sessions {
		if sess.ProjectName == targetProj.Name && sess.IsRunning {
			currentSessionID = id
			break
		}
	}

	if currentSessionID != 0 {
		if msg.String() == "esc" {
			ui.ViewState = model.StateDashboard
		}
		return nil
	}

	switch msg.String() {
	case "esc":
		ui.ViewState = model.StateDashboard
		return nil
	case "left", "h":
		if ui.SelectedBuildOption > 0 {
			ui.SelectedBuildOption--
		}
	case "right", "l":
		if ui.SelectedBuildOption < len(ui.BuildOptions)-1 {
			ui.SelectedBuildOption++
		}
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		ui.IsBuilding = true
		chosenOpt := ui.BuildOptions[ui.SelectedBuildOption]
		return session.SpawnBackgroundSession(ui, targetProj, chosenOpt)
	}
	return nil
}
