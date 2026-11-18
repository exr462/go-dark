package componentaction

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/team"
)

// TeamsModal lists the subscribed team manifests: "a" subscribes to a new
// one by git URL, "s" syncs all, "d" unsubscribes the highlighted team.
func TeamsModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	if ui.AddingTeam {
		return addTeamForm(ui, msg)
	}

	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.ViewState = model.StateConfigDeckModal
	case "up", "k":
		if ui.SelectedTeam > 0 {
			ui.SelectedTeam--
		}
	case "down", "j":
		if ui.SelectedTeam < len(ui.Config.Teams)-1 {
			ui.SelectedTeam++
		}
	case "a":
		ui.AddingTeam = true
		ui.FocusedInput = kbd.TeamGitURL
		ui.Inputs[kbd.TeamGitURL].SetValue("")
		ui.Inputs[kbd.TeamGitURL].Focus()
		return textinput.Blink
	case "s":
		if len(ui.Config.Teams) == 0 {
			return nil
		}
		ui.StatusMsg = "🛰 Syncing team manifests..."
		return team.SyncCmd(ui.Config.Teams)
	case "d":
		if ui.SelectedTeam >= len(ui.Config.Teams) {
			return nil
		}
		removed := ui.Config.Teams[ui.SelectedTeam]
		ui.Config.Teams = team.Unsubscribe(ui.Config.Teams, removed.Name)
		ui.SelectedTeam = max(min(ui.SelectedTeam, len(ui.Config.Teams)-1), 0)
		ui.StatusMsg = fmt.Sprintf("🗑 Unsubscribed from %s", removed.Name)
		return ApplyProjectChange(ui)
	}
	return nil
}

func addTeamForm(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.AddingTeam = false
		ui.Inputs[kbd.TeamGitURL].Blur()
		return nil
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Enter), action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		url := strings.TrimSpace(ui.Inputs[kbd.TeamGitURL].Value())
		if url == "" {
			return nil
		}
		ui.AddingTeam = false
		ui.Inputs[kbd.TeamGitURL].Blur()
		ui.StatusMsg = fmt.Sprintf("🛰 Subscribing to team manifest %s...", url)
		return team.SubscribeCmd(ui.Config.Teams, url)
	}
	var cmd tea.Cmd
	ui.Inputs[kbd.TeamGitURL], cmd = ui.Inputs[kbd.TeamGitURL].Update(msg)
	return cmd
}
