package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func GitStatusLoaded(ui *model.UI, msg config.GitStatusLoadedMsg) tea.Cmd {
	ui.GitStatusOutput = string(msg)
	if ui.GitStatusOutput == "" {
		ui.GitStatusOutput = "✨ Working tree clean."
	}
	return nil
}
