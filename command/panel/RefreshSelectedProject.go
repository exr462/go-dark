package commandpanel

import (
	"os"

	commandfile "github.com/exr462/go-dark/command/file"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/node"
)

//goland:noinspection GoMixedReceiverTypes
func RefreshSelectedProject(ui *model.UI) {
	if len(ui.Config.Projects) == 0 || ui.SelectedProject >= len(ui.Config.Projects) {
		return
	}

	ui.CurrentDirectory = ""
	proj := ui.Config.Projects[ui.SelectedProject]
	ui.TreeNodes = []model.FileNode{}

	if _, err := os.Stat(proj.Path); os.IsNotExist(err) {
		ui.FileViewer.SetContent("Project repository has not been cloned yet. Press Ctrl+G to clone/checkout.")
		ui.SelectedFile = 0
		return
	}
	node.BuildTreeNodes(ui, proj.Path, 0)
	ui.SelectedFile = 0
	if len(ui.TreeNodes) > 0 {
		_ = commandfile.ReadFileContent(ui)
	} else {
		ui.FileViewer.SetContent("Empty project directory root.")
	}
}
