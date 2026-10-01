package componentaction

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
	"github.com/exr462/go-dark/terminal"
)

// TerminalExecutionModal routes key signals for the interactive terminal workspace.
func TerminalExecutionModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		if !ui.IsBuilding {
			ui.ViewState = model.StateDashboard
		}
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Enter):
		if ui.IsBuilding {
			return nil
		}

		cmdStr := ui.Inputs[kbd.Terminal].Value()
		if cmdStr == "" {
			return nil
		}

		projectName := "go-dark"
		projectPath := "."
		if ui.SelectedGitProject >= 0 && ui.SelectedGitProject < len(ui.Config.Projects) {
			targetProj := ui.Config.Projects[ui.SelectedGitProject]
			projectName = targetProj.Name
			projectPath = targetProj.Path
		}

		// Echo the command like a real shell would, appending to the running
		// transcript instead of wiping prior history on every run.
		ui.TerminalLogs = append(ui.TerminalLogs, terminal.LogLine{
			Text:     fmt.Sprintf("dev@%s:~$ %s", projectName, cmdStr),
			IsPrompt: true,
		})

		ui.IsBuilding = true

		// Clear the input value upon submission so it's clean for the next command!
		ui.Inputs[kbd.Terminal].SetValue("")

		return session.SpawnTerminalSession(projectPath, cmdStr)
	}

	return nil
}
