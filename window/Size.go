package window

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

func Size(ui *model.UI, msg tea.WindowSizeMsg) tea.Cmd {
	ui.WindowWidth = msg.Width
	ui.WindowHeight = msg.Height
	ui.FileViewer.Width = (msg.Width / 2) - 4
	ui.FileViewer.Height = max(msg.Height-8, 5)
	ui.FuzzyViewer.Width = (msg.Width / 2) - 4
	ui.FuzzyViewer.Height = max(msg.Height-12, 5)
	return nil
}
