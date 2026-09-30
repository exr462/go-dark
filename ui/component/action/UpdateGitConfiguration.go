package componentaction

import (
	"fmt"
	"log"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func UpdateGitConfiguration(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		log.Printf("updateGitConfiguration Previous View State: %v", ui.PreviousViewState)
		ui.ViewState = ui.PreviousViewState
		return nil

	case "tab", "down":
		ui.Inputs[ui.FocusedInput].Blur()
		ui.FocusedInput = (ui.FocusedInput + 1) % 4
		ui.Inputs[ui.FocusedInput].Focus()
		return nil

	case "shift+tab", "up":
		ui.Inputs[ui.FocusedInput].Blur()
		ui.FocusedInput--
		if ui.FocusedInput < 0 {
			ui.FocusedInput = 3 // 🎯 FIX: Wrap cleanly to index 3 (matching your 4 total inputs)
		}
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
		ui.Config.Projects = make([]config.Project, 0)

		for _, project := range config.AvailableProjects {
			ui.Config.Projects = append(ui.Config.Projects, config.Project{
				Deployable: project.Deployable,
				Name:       project.Name,
				// TODO fix this to be magical
				Type:   "java",
				Path:   config.ResolvePath(ui.Config.BasePath, project.Name),
				GitURL: fmt.Sprintf("%s%s.git", config.BaseGitURL, project.Name),
			})
		}

		_ = config.SaveConfig(ui.Config)
		ui.ViewState = ui.PreviousViewState
		return nil
	}

	var cmd tea.Cmd
	ui.Inputs[ui.FocusedInput], cmd = ui.Inputs[ui.FocusedInput].Update(msg)
	return cmd
}
