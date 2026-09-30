package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/docker"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func DockerModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.ViewState = model.StateDashboard
		return nil
	case "up", "k":
		if ui.SelectedDockerRow > 0 {
			ui.SelectedDockerRow--
		}
	case "down", "j":
		if ui.SelectedDockerRow < len(ui.DockerContainers)-1 {
			ui.SelectedDockerRow++
		}
	case "s", "t", "r":
		if len(ui.DockerContainers) == 0 || ui.SelectedDockerRow >= len(ui.DockerContainers) {
			return nil
		}
		target := ui.DockerContainers[ui.SelectedDockerRow]
		a := "start"
		if msg.String() == "t" {
			a = "stop"
		}
		if msg.String() == "r" {
			a = "restart"
		}
		return tea.Batch(docker.DockerAction(target.ID, a), docker.FetchDockerContainers())
	}
	return nil
}
