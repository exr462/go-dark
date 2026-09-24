package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderJDKConfigModal(m model.UIState) string {
	var modalBody strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	labelStyle := lipgloss.NewStyle().Bold(true)
	activeLabelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	modalBody.WriteString(titleStyle.Render("☕ Java Environment Manager (Ctrl+J)") + "\n\n")

	switch m.JDKStep {
	case model.StepSelectJDKAction:
		modalBody.WriteString("👉 Select Dashboard Action to Perform:\n\n")
		options := []string{"Add New JDK Version Profile to Global Pool", "Assign Selected JDK to Active Workspace"}
		for i, opt := range options {
			if i == m.SelectedMenuIdx {
				modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", opt)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  %s", opt)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate Options | [Enter] Select | [Esc] Close Menu\x1b[0m")

	case model.StepAddNewJDKVersion:
		modalBody.WriteString("➕ Step 2: Register a New Java Environment Build Context:\n\n")

		// 1. DYNAMICALLY HIGHLIGHT LABELS TO SHOW WHICH COMPONENT CAPTURES FOCUS
		nameLabel := labelStyle.Render("  JDK Profile Logical Name (e.g. OpenJDK-17):")
		if m.FocusedInput == 7 {
			nameLabel = activeLabelStyle.Render("> JDK Profile Logical Name (e.g. OpenJDK-17):")
		}
		modalBody.WriteString(nameLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[7].View() + "\n\n") // Render field 7 explicitly

		pathLabel := labelStyle.Render("  JDK Home Absolute Paths System Directory (JAVA_HOME):")
		if m.FocusedInput == 8 {
			pathLabel = activeLabelStyle.Render("> JDK Home Absolute Paths System Directory (JAVA_HOME):")
		}
		modalBody.WriteString(pathLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[8].View() + "\n") // Render field 8 explicitly

		modalBody.WriteString("\n\x1b[90m[Tab] Swap Fields | [Enter] Save Profile Entry | [Esc] Cancel and Return\x1b[0m")

	case model.StepAssignJDKToProject:
		if m.SelectedProj >= len(m.Config.Projects) {
			modalBody.WriteString("❌ Error: No valid project context active.")
		} else {
			currentProj := m.Config.Projects[m.SelectedProj]
			currentBound := currentProj.JDKName
			if currentBound == "" {
				currentBound = "System Default"
			}
			modalBody.WriteString(fmt.Sprintf("📦 Bound Target Workspace: %s (Current Bound JDK: %s)\n\n", currentProj.Name, currentBound))
			modalBody.WriteString("👉 Step 2: Select Profile to Bind to Workspace Environment:\n\n")

			if len(m.Config.JDKs) == 0 {
				modalBody.WriteString("  ❌ No configured JDK profiles found in database.\n  Go back and add one first.")
			} else {
				for i, jdk := range m.Config.JDKs {
					if i == m.SelectedJDKIdx {
						modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> %s (%s)", jdk.Name, jdk.Path)) + "\n")
					} else {
						modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  %s", jdk.Name)) + "\n")
					}
				}
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Browse SDKs | [Enter] Confirm Workspace Override Bindings | [Esc] Back\x1b[0m")
	}

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("220")).
		Background(lipgloss.Color("234")).
		Padding(1, 4, 1, 4).
		Width(75).
		Render(modalBody.String())

	return lipgloss.Place(
		m.TerminalW, m.TerminalH,
		lipgloss.Center, lipgloss.Center,
		modalBox,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}
