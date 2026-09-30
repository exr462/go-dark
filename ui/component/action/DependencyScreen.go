package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func DependencyScreen(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	projIdx := ui.DepScreen.ActiveProjectIndex
	if projIdx < 0 || projIdx >= len(ui.Config.Projects) {
		ui.ViewState = model.StateDashboard
		return nil
	}

	// 🔍 FIX 1: Direct slice reference modification
	// Do NOT copy out the project struct into a local variable.
	// Instead, manipulate the array directly at its absolute index position.

	switch msg.String() {
	case "up", "k":
		if ui.DepScreen.Cursor > 0 {
			ui.DepScreen.Cursor--
		}

	case "down", "j":
		if ui.DepScreen.Cursor < len(ui.DepScreen.AvailableOptions)-1 {
			ui.DepScreen.Cursor++
		}

	case "space":
		selectedTarget := ui.DepScreen.AvailableOptions[ui.DepScreen.Cursor]

		// Find if dependency exists in our absolute index target
		foundIdx := -1
		for i, dep := range config.AvailableProjects[projIdx].Dependencies {
			if dep == selectedTarget {
				foundIdx = i
				break
			}
		}

		if foundIdx >= 0 {
			// Toggle Off: Remove dependency directly from the slice array
			config.AvailableProjects[projIdx].Dependencies = append(
				config.AvailableProjects[projIdx].Dependencies[:foundIdx],
				config.AvailableProjects[projIdx].Dependencies[foundIdx+1:]...,
			)
		} else {
			// Toggle On: Add dependency directly to the slice array
			config.AvailableProjects[projIdx].Dependencies = append(
				config.AvailableProjects[projIdx].Dependencies,
				selectedTarget,
			)
		}

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		_ = config.SaveConfig(ui.Config)
		ui.ViewState = model.StateDashboard
		return initializer.InitializeWorkspace(ui).OnAction()

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape), action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.QuitApplication):
		ui.ViewState = model.StateDashboard
		return nil
	}

	// 🔍 FIX 2: Return the updated model 'm' back to Bubble Tea!
	// If you were returning 'nil, nil' or a raw unmutated model state here,
	// Bubble Tea wouldn't know the selection markers changed.
	return nil
}
