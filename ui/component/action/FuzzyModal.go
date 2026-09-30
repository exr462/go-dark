package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	commandfile "github.com/exr462/go-dark/command/file"
	"github.com/exr462/go-dark/fuzzy"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func FuzzyModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.ViewState = model.StateDashboard
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Toggle):
		if ui.FuzzyMode == model.FuzzyModeFiles {
			ui.FuzzyMode = model.FuzzyModeContent
		} else {
			ui.FuzzyMode = model.FuzzyModeFiles
		}
		ui.SelectedFuzzy = 0
		fuzzy.FuzzySearchEngine(ui)
		fuzzy.FuzzyPreviewPane(ui)
		return nil

	case "up", "k":
		if ui.SelectedFuzzy > 0 {
			ui.SelectedFuzzy--
			fuzzy.FuzzyPreviewPane(ui)
		}
		return nil

	case "down", "j":
		if ui.SelectedFuzzy < len(ui.FuzzyResults)-1 {
			ui.SelectedFuzzy++
			fuzzy.FuzzyPreviewPane(ui)
		}
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		if len(ui.FuzzyResults) > 0 && ui.SelectedFuzzy < len(ui.FuzzyResults) {
			target := ui.FuzzyResults[ui.SelectedFuzzy]
			ui.ViewState = model.StateDashboard

			for idx, node := range ui.TreeNodes {
				if node.FullPath == target.FullPath {
					ui.SelectedFile = idx
					ui.ActiveFocus = model.FocusTree
					break
				}
			}
			return commandfile.ReadFileContent(ui)
		}
		ui.ViewState = model.StateDashboard
		return nil
	}

	var cmd tea.Cmd
	oldVal := ui.FuzzyQueryInput.Value()
	var genericMsg tea.Msg = msg
	ui.FuzzyQueryInput, cmd = ui.FuzzyQueryInput.Update(genericMsg)

	if ui.FuzzyQueryInput.Value() != oldVal {
		ui.SelectedFuzzy = 0
		fuzzy.FuzzySearchEngine(ui)
		fuzzy.FuzzyPreviewPane(ui)
	}

	return cmd
}
