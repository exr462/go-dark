package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateGitOpsModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
		if m.ui.GitOperationStep == model.StepSelectGitBranch {
			m.ui.GitOperationStep = model.StepSelectGitCommand
		} else if m.ui.GitOperationStep == model.StepSelectGitCommand {
			m.ui.GitOperationStep = model.StepSelectGitProject
		} else if m.ui.GitOperationStep == 3 {
			m.ui.GitOperationStep = model.StepSelectGitCommand
		} else {
			m.ui.ViewState = model.StateDashboard
		}
		return m, nil

	case "up", "k":
		switch m.ui.GitOperationStep {
		case model.StepSelectGitProject:
			if m.ui.SelectedGitProject > 0 {
				m.ui.SelectedGitProject--
			}
		case model.StepSelectGitCommand:
			if m.ui.SelectedGitCommand > 0 {
				m.ui.SelectedGitCommand--
			}
		case model.StepSelectGitBranch:
			if m.ui.SelectedGitBranch > 0 {
				m.ui.SelectedGitBranch--
			}
		}
		return m, nil

	case "down", "j":
		switch m.ui.GitOperationStep {
		case model.StepSelectGitProject:
			if m.ui.SelectedGitProject < len(m.ui.Config.Projects)-1 {
				m.ui.SelectedGitProject++
			}
		case model.StepSelectGitCommand:
			if m.ui.SelectedGitCommand < len(m.ui.GitCommands)-1 {
				m.ui.SelectedGitCommand++
			}
		case model.StepSelectGitBranch:
			if m.ui.SelectedGitBranch < len(m.ui.AvailableBranches)-1 {
				m.ui.SelectedGitBranch++
			}
		}
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
		switch m.ui.GitOperationStep {
		case model.StepSelectGitProject:
			m.ui.GitOperationStep = model.StepSelectGitCommand
			m.ui.SelectedGitCommand = 0
			return m, m.loadGitBranchesCmd()

		case model.StepSelectGitCommand:
			chosenCmd := m.ui.GitCommands[m.ui.SelectedGitCommand]
			targetProj := m.ui.Config.Projects[m.ui.SelectedGitProject]

			if strings.HasPrefix(chosenCmd, "checkout") {
				m.ui.GitOperationStep = model.StepSelectGitBranch
				m.ui.AvailableBranches = []string{}
				m.ui.SelectedGitBranch = 0
				return m, m.loadGitBranchesCmd()
			}

			if chosenCmd == "status" {
				if !targetProj.Fetched {
					m.ui.StatusMsg = fmt.Sprintf("⚠️ %s is not cloned yet.", targetProj.Name)
					return m, nil
				}
				m.ui.GitOperationStep = 3
				m.ui.GitStatusOutput = "⏳ Querying workspace parameters..."
				return m, m.runGitCommand(targetProj, "status")
			}

			if chosenCmd == "reset" {
				chosenCmd = "reset --hard"
			}

			if !targetProj.Fetched && (chosenCmd == "pull" || chosenCmd == "fetch" || strings.HasPrefix(chosenCmd, "reset")) {
				m.ui.StatusMsg = fmt.Sprintf("⚠️ %s is not cloned yet. Use checkout or clone first.", targetProj.Name)
				return m, nil
			}

			m.ui.ViewState = model.StateDashboard
			m.ui.StatusMsg = fmt.Sprintf("🔄 Executing git %s on %s...", chosenCmd, targetProj.Name)
			return m, m.runGitCommand(targetProj, chosenCmd)

		case model.StepSelectGitBranch:
			if len(m.ui.AvailableBranches) > 0 && m.ui.SelectedGitBranch < len(m.ui.AvailableBranches) {
				targetProj := m.ui.Config.Projects[m.ui.SelectedGitProject]
				targetBranch := m.ui.AvailableBranches[m.ui.SelectedGitBranch]
				m.ui.StatusMsg = fmt.Sprintf("🔄 Checking out %s on %s...", targetBranch, targetProj.Name)
				return m, m.executeGitCheckoutCmd(targetProj, targetBranch)
			}
			return m, nil
		}
	}
	return m, nil
}
