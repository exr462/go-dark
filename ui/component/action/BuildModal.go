package componentaction

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
)

// BuildModal picks and runs one of the selected project's commands. Builds
// run as sessions: they keep going when the modal is closed, the modal shows
// their output live, and the terminal shortcut attaches to them (where
// ctrl+c cancels).
func BuildModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	if len(ui.Config.Projects) == 0 || ui.SelectedProject >= len(ui.Config.Projects) {
		ui.ViewState = model.StateDashboard
		return nil
	}
	proj := ui.Config.Projects[ui.SelectedProject]
	latest := session.LatestBuild(ui.Shells, proj.ID, session.TagBuild)
	running := latest != nil && latest.Running()

	switch msg.String() {
	case binding(ui, action.Escape):
		ui.ViewState = model.StateDashboard
	case binding(ui, action.OpenTerminal):
		if latest != nil && ui.Shells != nil {
			AttachSession(ui, ui.Shells, latest.ID)
		}
	case "left", "h":
		if !running && ui.SelectedBuildOption > 0 {
			ui.SelectedBuildOption--
		}
	case "right", "l":
		if !running && ui.SelectedBuildOption < len(ui.BuildOptions)-1 {
			ui.SelectedBuildOption++
		}
	case binding(ui, action.Save):
		if running || ui.Shells == nil || ui.SelectedBuildOption >= len(ui.BuildOptions) {
			return nil
		}
		key := ui.BuildOptions[ui.SelectedBuildOption]
		w, h := ui.TerminalPaneSize()
		keep := func(id int) bool { return slices.Contains(ui.TerminalTabs, id) }
		if _, err := session.StartBuild(ui.Shells, ui.Config, proj, key, session.TagBuild, w, h, keep); err != nil {
			ui.StatusMsg = fmt.Sprintf("❌ Build: %v", err)
			return nil
		}
		ui.StatusMsg = fmt.Sprintf("🏗️  Building %s (%s)...", proj.Name, key)
	}
	return nil
}
