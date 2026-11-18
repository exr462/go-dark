package componentaction

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/team"
)

// gitConfigurationFields are the inputs of the global preferences form, in
// tab order.
var gitConfigurationFields = []kbd.InputField{
	kbd.GitWorkspace, kbd.GitUsername, kbd.GitEmail, kbd.GitMaxTagListSize, kbd.TeamGitURL,
}

func fieldIndex(field kbd.InputField) int {
	for i, f := range gitConfigurationFields {
		if f == field {
			return i
		}
	}
	return 0
}

//goland:noinspection GoMixedReceiverTypes
func UpdateGitConfiguration(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		log.Printf("updateGitConfiguration Previous View State: %v", ui.PreviousViewState)
		ui.ViewState = ui.PreviousViewState
		return nil

	case "tab", "down":
		ui.Inputs[ui.FocusedInput].Blur()
		ui.FocusedInput = gitConfigurationFields[(fieldIndex(ui.FocusedInput)+1)%len(gitConfigurationFields)]
		ui.Inputs[ui.FocusedInput].Focus()
		return nil

	case "shift+tab", "up":
		ui.Inputs[ui.FocusedInput].Blur()
		i := fieldIndex(ui.FocusedInput) - 1
		if i < 0 {
			i = len(gitConfigurationFields) - 1
		}
		ui.FocusedInput = gitConfigurationFields[i]
		ui.Inputs[ui.FocusedInput].Focus()
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		ui.Config.BasePath = ui.Inputs[kbd.GitWorkspace].Value()
		ui.Config.GitUsername = ui.Inputs[kbd.GitUsername].Value()
		ui.Config.GitEmail = ui.Inputs[kbd.GitEmail].Value()
		ui.Config.MaxListTag, _ = strconv.Atoi(ui.Inputs[kbd.GitMaxTagListSize].Value())
		if ui.Config.BasePath == "" {
			return nil
		}
		// Projects come from team manifests and the user's own local
		// projects; re-resolve their default paths against the base path.
		ui.Config.RefreshProjects()
		_ = config.SaveConfig(ui.Config)
		ui.ViewState = ui.PreviousViewState

		teamURL := strings.TrimSpace(ui.Inputs[kbd.TeamGitURL].Value())
		ui.Inputs[kbd.TeamGitURL].SetValue("")
		if teamURL == "" {
			return nil
		}
		ui.StatusMsg = fmt.Sprintf("🛰 Subscribing to team manifest %s...", teamURL)
		return team.SubscribeCmd(ui.Config.Teams, teamURL)
	}

	var cmd tea.Cmd
	ui.Inputs[ui.FocusedInput], cmd = ui.Inputs[ui.FocusedInput].Update(msg)
	return cmd
}
