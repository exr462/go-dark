package commandfile

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

func FileLoad(ui *model.UI, cmds []tea.Cmd) []tea.Cmd {
	if len(ui.TreeNodes) > 0 {
		if ui.SelectedFile >= len(ui.TreeNodes) {
			ui.SelectedFile = 0
		}
		cmds = append(cmds, ReadFileContent(ui))
	} else {
		ui.FileViewer.SetContent("Empty or uncloned project repository.")
	}
	return cmds
}
