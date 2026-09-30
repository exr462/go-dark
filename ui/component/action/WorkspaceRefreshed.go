package componentaction

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/node"
	componentmessage "github.com/exr462/go-dark/ui/component/message"
)

func WorkspaceRefreshed(ui *model.UI, msg componentmessage.WorkspaceRefreshedMsg) tea.Cmd {
	ui.Files = msg.Files
	if ui.SelectedProject >= 0 && ui.SelectedProject < len(ui.Config.Projects) {
		proj := ui.Config.Projects[ui.SelectedProject]
		ui.TreeNodes = []model.FileNode{}
		if _, err := os.Stat(proj.Path); err == nil {
			node.BuildTreeNodes(ui, proj.Path, 0)
		}
	}
	return func() tea.Msg { return model.FileLoadMsg("sync") }
}
