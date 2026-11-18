package componentaction

import (
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

// ApplyProjectChange re-merges team manifests and local projects after
// subscriptions or local projects changed, persists the result and
// refreshes the dashboard tree.
func ApplyProjectChange(ui *model.UI) tea.Cmd {
	ui.Config.RefreshProjects()
	if ui.SelectedProject >= len(ui.Config.Projects) {
		ui.SelectedProject = max(len(ui.Config.Projects)-1, 0)
	}
	_ = config.SaveConfig(ui.Config)
	return initializer.InitializeWorkspace(ui).OnAction()
}
