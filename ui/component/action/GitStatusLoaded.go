package componentaction

import (
	tea "charm.land/bubbletea/v2"
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
