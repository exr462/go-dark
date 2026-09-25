package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderGitOpsModal(m *model.UIState) string {
	var modalBody strings.Builder

	modalBody.WriteString(TitleStyle.Render("⚡ Git Hub Central Control (Ctrl+G)") + "\n\n")

	switch m.GitOperationStep {
	case 0:
		modalBody.WriteString("👉 Step 1: Select Target Project Repository:\n\n")
		for i, p := range m.Config.Projects {
			if i == m.SelectedGitProject {
				modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> %s (%s)", p.Name, p.GitURL)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  %s", p.Name)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate | [Enter] Select Command | [Esc] Exit\x1b[0m")

	case 1:
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

	case 2: // 👈 New branch rendering layout
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
	case 3: // 🟢 Render Step 4: Git Status Details Screen
		targetProj := m.Config.Projects[m.SelectedGitProject]
		modalBody.WriteString(fmt.Sprintf("📦 Project: %s\n", targetProj.Name))
		modalBody.WriteString("📊 Current Working Tree Status:\n\n")

		// Parse lines and add syntax color highlighting to status codes
		lines := strings.Split(m.GitStatusOutput, "\n")
		for _, line := range lines {
			if len(line) < 3 {
				modalBody.WriteString(cleanStyle.Render(line) + "\n")
				continue
			}

			code := line[:2]
			file := line[2:]

			switch {
			case strings.Contains(code, "M"):
				modalBody.WriteString(modifiedStyle.Render(" 📝 M ") + file + "\n")
			case strings.Contains(code, "??"):
				modalBody.WriteString(untrackedStyle.Render(" ❓ ?? ") + file + "\n")
			case code == "A " || code == "M ":
				modalBody.WriteString(stagedStyle.Render(" 🟩 Staged: ") + file + "\n")
			default:
				modalBody.WriteString("  " + line + "\n")
			}
		}

		modalBody.WriteString("\n\x1b[90m[Esc] Back to Operational Selection Commands Menu\x1b[0m")
	}

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		ModalBox.Width(m.WindowWidth-4).Render(modalBody.String()),
		whiteSpace,
		lipgloss.WithWhitespaceForeground(DarkerGrey),
	)
}
