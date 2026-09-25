package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderGitOpsModal(m model.UIState) string {
	var modalBody strings.Builder

	modalBody.WriteString(TitleStyle.Render("⚡ Git Hub Central Control (Ctrl+G)") + "\n\n")

	switch m.GitOperationStep {
	case model.StepSelectGitProject:
		modalBody.WriteString("👉 Step 1: Select Target Project Repository:\n\n")
		for i, p := range m.Config.Projects {
			if i == m.SelectedGitProject {
				modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> %s (%s)", p.Name, p.GitURL)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  %s", p.Name)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate | [Enter] Select Command | [Esc] Exit\x1b[0m")

	case model.StepSelectGitCommand:
		targetProj := m.Config.Projects[m.SelectedGitProject]
		modalBody.WriteString(fmt.Sprintf("📦 Project Workspace: %s\n", targetProj.Name))
		modalBody.WriteString("👉 Step 2: Choose Git Operation to Execute:\n\n")

		for i, cmd := range m.GitCommands {
			if i == m.SelectedGitCommand {
				modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> git %s", cmd)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  git %s", cmd)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate | [Enter] Select | [Esc] Back to Projects\x1b[0m")

	case model.StepSelectGitBranch: // 👈 New branch rendering layout
		targetProj := m.Config.Projects[m.SelectedGitProject]
		modalBody.WriteString(fmt.Sprintf("📦 Project Workspace: %s\n", targetProj.Name))
		modalBody.WriteString("👉 Step 3: Select Branch to Checkout:\n\n")

		if len(m.AvailableBranches) == 0 {
			modalBody.WriteString("  \x1b[91mNo branches found or loading...\x1b[0m\n")
		} else {
			for i, branch := range m.AvailableBranches {
				if i == m.SelectedGitBranch {
					modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", branch)) + "\n")
				} else {
					modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  %s", branch)) + "\n")
				}
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate | [Enter] Checkout Branch | [Esc] Back to Commands\x1b[0m")
	}

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		ModalBox.Width(m.WindowWidth-4).Render(modalBody.String()),
		whiteSpace,
		lipgloss.WithWhitespaceForeground(DarkerGrey),
	)
}
