package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateGitOpsModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.state.GitOperationStep == model.StepSelectGitBranch {
			m.state.GitOperationStep = model.StepSelectGitCommand
		} else if m.state.GitOperationStep == model.StepSelectGitCommand {
			m.state.GitOperationStep = model.StepSelectGitProject
		} else if m.state.GitOperationStep == 3 {
			m.state.GitOperationStep = model.StepSelectGitCommand
		} else {
			m.state.ViewState = model.StateDashboard
		}
		return m, nil

	case "up", "k":
		switch m.state.GitOperationStep {
		case model.StepSelectGitProject:
			if m.state.SelectedGitProject > 0 {
				m.state.SelectedGitProject--
			}
		case model.StepSelectGitCommand:
			if m.state.SelectedGitCommand > 0 {
				m.state.SelectedGitCommand--
			}
		case model.StepSelectGitBranch:
			if m.state.SelectedGitBranch > 0 {
				m.state.SelectedGitBranch--
			}
		}
		return m, nil

	case "down", "j":
		switch m.state.GitOperationStep {
		case model.StepSelectGitProject:
			if m.state.SelectedGitProject < len(m.state.Config.Projects)-1 {
				m.state.SelectedGitProject++
			}
		case model.StepSelectGitCommand:
			if m.state.SelectedGitCommand < len(m.state.GitCommands)-1 {
				m.state.SelectedGitCommand++
			}
		case model.StepSelectGitBranch:
			if m.state.SelectedGitBranch < len(m.state.AvailableBranches)-1 {
				m.state.SelectedGitBranch++
			}
		}
		return m, nil

	case "enter":
		switch m.state.GitOperationStep {
		case model.StepSelectGitProject:
			m.state.GitOperationStep = model.StepSelectGitCommand
			m.state.SelectedGitCommand = 0
			return m, m.loadGitBranchesCmd()

		case model.StepSelectGitCommand:
			chosenCmd := m.state.GitCommands[m.state.SelectedGitCommand]
			targetProj := m.state.Config.Projects[m.state.SelectedGitProject]

			if strings.HasPrefix(chosenCmd, "checkout") {
				m.state.GitOperationStep = model.StepSelectGitBranch
				m.state.AvailableBranches = []string{}
				m.state.SelectedGitBranch = 0
				return m, m.loadGitBranchesCmd()
			}

			if chosenCmd == "status" {
				if !targetProj.Fetched {
					m.state.StatusMsg = fmt.Sprintf("⚠️ %s is not cloned yet.", targetProj.Name)
					return m, nil
				}
				m.state.GitOperationStep = 3
				m.state.GitStatusOutput = "⏳ Querying workspace parameters..."
				return m, m.runGitCommand(targetProj, "status")
			}

			if chosenCmd == "reset" {
				chosenCmd = "reset --hard"
			}

			if !targetProj.Fetched && (chosenCmd == "pull" || chosenCmd == "fetch" || strings.HasPrefix(chosenCmd, "reset")) {
				m.state.StatusMsg = fmt.Sprintf("⚠️ %s is not cloned yet. Use checkout or clone first.", targetProj.Name)
				return m, nil
			}

			m.state.ViewState = model.StateDashboard
			m.state.StatusMsg = fmt.Sprintf("🔄 Executing git %s on %s...", chosenCmd, targetProj.Name)
			return m, m.runGitCommand(targetProj, chosenCmd)

		case model.StepSelectGitBranch:
			if len(m.state.AvailableBranches) > 0 && m.state.SelectedGitBranch < len(m.state.AvailableBranches) {
				targetProj := m.state.Config.Projects[m.state.SelectedGitProject]
				targetBranch := m.state.AvailableBranches[m.state.SelectedGitBranch]
				m.state.StatusMsg = fmt.Sprintf("🔄 Checking out %s on %s...", targetBranch, targetProj.Name)
				return m, m.executeGitCheckoutCmd(targetProj, targetBranch)
			}
			return m, nil
		}
	}
	return m, nil
}
