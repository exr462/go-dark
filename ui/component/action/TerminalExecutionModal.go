package componentaction

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
)

// TerminalExecutionModal routes key signals for the interactive terminal workspace.
func TerminalExecutionModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	// 1. Detect if a process is already running in this view context
	var isRunning bool
	if currentProc, ok := ui.Sessions[ui.ActiveTerminalSessionID]; ok {
		isRunning = currentProc.IsRunning
	}

	// 2. Handle global escaping safely back to the master dashboard
	if msg.String() == action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape) {
		// If running, we block dropping out to prevent losing track of logs, or allow it based on preference
		if !isRunning {
			ui.ViewState = model.StateDashboard
			return nil
		}
	}

	// 3. If running, intercept inputs to allow process abortion or text viewing only
	if isRunning {
		if msg.String() == action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Abort) {
			// Trigger background process cancellation context
			return session.AbortTerminalSession(ui, ui.ActiveTerminalSessionID)
		}
		return nil
	}

	// 4. Execution Router & Form Input Typing Phase
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Execute):
		// Get raw terminal command string from the inputs dictionary matrix
		cmdStr := ui.Inputs[kbd.Terminal].Value()
		if cmdStr == "" {
			return nil
		}

		// Set building/executing state tracking flags
		ui.IsBuilding = true

		// Spawn cross-platform shell process relative to your core workspace base path
		return session.SpawnTerminalSession(ui, ui.Inputs[kbd.GitWorkspace].Value(), cmdStr)

	default:
		// Forward typing keystrokes directly into the dedicated textinput model component
		var cmd tea.Cmd
		var ti textinput.Model = ui.Inputs[kbd.Terminal]

		ti, cmd = ti.Update(msg)
		ui.Inputs[kbd.Terminal] = ti

		return cmd
	}
}
