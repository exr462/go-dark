package componentaction

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	commandgit "github.com/exr462/go-dark/command/git"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func GitOpsModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		if ui.GitOperationStep == model.StepSelectGitBranch {
			ui.GitOperationStep = model.StepSelectGitCommand
		} else if ui.GitOperationStep == model.StepSelectGitCommand {
			ui.GitOperationStep = model.StepSelectGitProject
		} else if ui.GitOperationStep == 3 {
			ui.GitOperationStep = model.StepSelectGitCommand
		} else {
			ui.ViewState = model.StateDashboard
		}
		return nil

	case "up", "k":
		switch ui.GitOperationStep {
		case model.StepSelectGitProject:
			if ui.SelectedGitProject > 0 {
				ui.SelectedGitProject--
			}
		case model.StepSelectGitCommand:
			if ui.SelectedGitCommand > 0 {
				ui.SelectedGitCommand--
			}
		case model.StepSelectGitBranch:
			if ui.SelectedGitBranch > 0 {
				ui.SelectedGitBranch--
			}
		}
		return nil

	case "down", "j":
		switch ui.GitOperationStep {
		case model.StepSelectGitProject:
			if ui.SelectedGitProject < len(ui.Config.Projects)-1 {
				ui.SelectedGitProject++
			}
		case model.StepSelectGitCommand:
			if ui.SelectedGitCommand < len(ui.GitCommands)-1 {
				ui.SelectedGitCommand++
			}
		case model.StepSelectGitBranch:
			if ui.SelectedGitBranch < len(ui.AvailableBranches)-1 {
				ui.SelectedGitBranch++
			}
		}
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		switch ui.GitOperationStep {
		case model.StepSelectGitProject:
			ui.GitOperationStep = model.StepSelectGitCommand
			ui.SelectedGitCommand = 0
			return LoadGitBranches(ui)

		case model.StepSelectGitCommand:
			chosenCmd := ui.GitCommands[ui.SelectedGitCommand]
			targetProj := ui.Config.Projects[ui.SelectedGitProject]

			if strings.HasPrefix(chosenCmd, "checkout") {
				ui.GitOperationStep = model.StepSelectGitBranch
				ui.AvailableBranches = []string{}
				ui.SelectedGitBranch = 0
				return LoadGitBranches(ui)
			}

			if chosenCmd == "status" {
				if !targetProj.Fetched {
					ui.StatusMsg = fmt.Sprintf("⚠️ %s is not cloned yet.", targetProj.Name)
					return nil
				}
				ui.GitOperationStep = 3
				ui.GitStatusOutput = "⏳ Querying workspace parameters..."
				return commandgit.GitCommand(ui, targetProj, "status")
			}

			if chosenCmd == "reset" {
				chosenCmd = "reset --hard"
			}

			if !targetProj.Fetched && (chosenCmd == "pull" || chosenCmd == "fetch" || strings.HasPrefix(chosenCmd, "reset")) {
				ui.StatusMsg = fmt.Sprintf("⚠️ %s is not cloned yet. Use checkout or clone first.", targetProj.Name)
				return nil
			}

			ui.ViewState = model.StateDashboard
			ui.StatusMsg = fmt.Sprintf("🔄 Executing git %s on %s...", chosenCmd, targetProj.Name)
			return commandgit.GitCommand(ui, targetProj, chosenCmd)

		case model.StepSelectGitBranch:
			if len(ui.AvailableBranches) > 0 && ui.SelectedGitBranch < len(ui.AvailableBranches) {
				targetProj := ui.Config.Projects[ui.SelectedGitProject]
				targetBranch := ui.AvailableBranches[ui.SelectedGitBranch]
				ui.StatusMsg = fmt.Sprintf("🔄 Checking out %s on %s...", targetBranch, targetProj.Name)
				return commandgit.GitCheckoutCmd(targetProj, targetBranch)
			}
			return nil
		}
	}
	return nil
}
