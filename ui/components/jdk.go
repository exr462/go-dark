package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderJDKConfigModal(m *model.UIState) string {
	var modalBody strings.Builder

	modalBody.WriteString(TitleStyle.Render("☕ Java Environment Manager (Ctrl+J)") + "\n\n")

	switch m.JDKStep {
	case model.StepSelectJDKAction:
		modalBody.WriteString("👉 Select Dashboard Action to Perform:\n\n")
		options := []string{"Add New JDK Version Profile to Global Pool", "Assign Selected JDK to Active Workspace"}
		for i, opt := range options {
			if i == m.SelectedMenuIndex {
				modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", opt)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  %s", opt)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate Options | [Enter] Select | [Esc] Close Menu\x1b[0m")

	case model.StepAddNewJDKVersion:
		modalBody.WriteString("➕ Step 2: Register a New Java Environment Build Context:\n\n")

		// 1. DYNAMICALLY HIGHLIGHT LABELS TO SHOW WHICH COMPONENT CAPTURES FOCUS
		nameLabel := boldStyle.Render("  JDK Profile Logical Name (e.g. OpenJDK-17):")
		if m.FocusedInput == 7 {
			nameLabel = activeLabelStyle.Render("> JDK Profile Logical Name (e.g. OpenJDK-17):")
		}
		modalBody.WriteString(nameLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[7].View() + "\n\n") // Render field 7 explicitly

		pathLabel := boldStyle.Render("  JDK Home Absolute Paths System Directory (JAVA_HOME):")
		if m.FocusedInput == 8 {
			pathLabel = activeLabelStyle.Render("> JDK Home Absolute Paths System Directory (JAVA_HOME):")
		}
		modalBody.WriteString(pathLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[8].View() + "\n") // Render field 8 explicitly

		modalBody.WriteString("\n\x1b[90m[Tab] Swap Fields | [Enter] Save Profile Entry | [Esc] Cancel and Return\x1b[0m")

	case model.StepAssignJDKToProject:
		if m.SelectedProject >= len(m.Config.Projects) {
			modalBody.WriteString("❌ Error: No valid project context active.")
		} else {
			currentProj := m.Config.Projects[m.SelectedProject]
			currentBound := currentProj.JDKName
			if currentBound == "" {
				currentBound = "System Default"
			}
			modalBody.WriteString(fmt.Sprintf("📦 Bound Target Workspace: %s (Current Bound JDK: %s)\n\n", currentProj.Name, currentBound))
			modalBody.WriteString("👉 Step 2: Select Profile to Bind to Workspace Environment:\n\n")
			decorateProfiles(modalBody, "JDK", m.SelectedJDKIndex, m.Config.JDKs)
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Browse SDKs | [Enter] Confirm Workspace Override Bindings | [Esc] Back\x1b[0m")
	}

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		ModalBox.
			Width(m.WindowWidth-4).
			Render(modalBody.String()),
		whiteSpace,
		lipgloss.WithWhitespaceForeground(DarkerGrey),
	)
}
