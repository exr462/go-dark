package componentaction

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

// DependencyScreen edits the DependsOn list of the project picked on the
// dashboard. Edits are saved to the user's config; on a team project they
// are kept as a local override of the manifest's list.
//
//goland:noinspection GoMixedReceiverTypes
func DependencyScreen(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	projIdx := config.FindProject(ui.Config.Projects, ui.DepScreen.ActiveProjectID)
	if projIdx < 0 {
		ui.ViewState = model.StateDashboard
		return nil
	}
	proj := &ui.Config.Projects[projIdx]

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
		if len(ui.DepScreen.AvailableOptions) == 0 {
			return nil
		}
		target := ui.DepScreen.AvailableOptions[ui.DepScreen.Cursor]
		if i := slices.Index(proj.DependsOn, target); i >= 0 {
			proj.DependsOn = slices.Delete(slices.Clone(proj.DependsOn), i, i+1)
		} else {
			proj.DependsOn = append(slices.Clone(proj.DependsOn), target)
		}

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		if !slices.Equal(proj.DependsOn, ui.DepScreen.Original) && !proj.IsLocal() {
			proj.DepsOverridden = true
		}
		_ = config.SaveConfig(ui.Config)
		ui.ViewState = model.StateDashboard
		return initializer.InitializeWorkspace(ui).OnAction()

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape), action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.QuitApplication):
		proj.DependsOn = ui.DepScreen.Original
		ui.ViewState = model.StateDashboard
		return nil
	}

	return nil
}
