package window

import (
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/model"
)

func Size(ui *model.UI, msg tea.WindowSizeMsg) tea.Cmd {
	ui.WindowWidth = msg.Width
	ui.WindowHeight = msg.Height
	ui.FileViewer.SetWidth((msg.Width / 2) - 4)
	ui.FileViewer.SetHeight(max(msg.Height-8, 5))
	ui.FuzzyViewer.SetWidth((msg.Width / 2) - 4)
	ui.FuzzyViewer.SetHeight(max(msg.Height-12, 5))
	return nil
}
