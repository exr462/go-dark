package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
	"github.com/exr462/go-dark/terminal"
)

// TerminalExecutionModal routes key signals for the interactive terminal workspace.
func TerminalExecutionModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	var isRunning bool
	if currentProc, ok := ui.Sessions[ui.ActiveTerminalSessionID]; ok {
		isRunning = currentProc.IsRunning
	}

	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		if !isRunning {
			ui.ViewState = model.StateDashboard
		}
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Enter):
		cmdStr := ui.Inputs[kbd.Terminal].Value()
		if cmdStr == "" {
			return nil
		}

		ui.IsBuilding = true
		ui.TerminalLogs = make([]terminal.LogLine, 0)
		targetProj := ui.Config.Projects[ui.SelectedGitProject]

		// Clear the input value upon submission so it's clean for the next command!
		ui.Inputs[kbd.Terminal].SetValue("")

		return session.SpawnTerminalSession(targetProj.Path, cmdStr)
	}

	return nil
}
