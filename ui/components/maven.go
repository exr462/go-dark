package components

import (
	Fmt "fmt"
	Strings "strings"

	Lipgloss "github.com/charmbracelet/lipgloss"
	Modal "github.com/exr462/go-dark/model"
)

func RenderMvnConfigModal(m *Modal.UIState) string {
	var modalBody Strings.Builder

	modalBody.WriteString(TitleStyle.Render("🛠️ Apache Maven Manager (Ctrl+U)") + "\n\n")

	switch m.MavenStep {
	case Modal.StepSelectMvnAction:
		modalBody.WriteString("👉 Select Dashboard Action to Perform:\n\n")
		options := []string{"Add New Maven Version Profile to Global Pool", "Assign Selected Maven to Active Workspace"}
		for i, opt := range options {
			if i == m.SelectedMenuIndex {
				modalBody.WriteString(selectedStyle.Render(Fmt.Sprintf("> %s", opt)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(Fmt.Sprintf("  %s", opt)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate Options | [Enter] Select | [Esc] Close Menu\x1b[0m")

	case Modal.StepAddNewMvnVersion:
		modalBody.WriteString("➕ Step 2: Register a New Maven Environment Context:\n\n")

		nameLabel := boldStyle.Render("  Maven Profile Name (e.g., Maven-3.9.6):")
		if m.FocusedInput == 9 {
			nameLabel = activeLabelStyle.Render("> Maven Profile Name (e.g., Maven-3.9.6):")
		}
		modalBody.WriteString(nameLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[9].View() + "\n\n")

		pathLabel := boldStyle.Render("  Maven Home Directory (MAVEN_HOME / M2_HOME):")
		if m.FocusedInput == 10 {
			pathLabel = activeLabelStyle.Render("> Maven Home Directory (MAVEN_HOME / M2_HOME):")
		}
		modalBody.WriteString(pathLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[10].View() + "\n")

		modalBody.WriteString("\n\x1b[90m[Tab] Swap Fields | [Enter] Save Profile Entry | [Esc] Cancel and Return\x1b[0m")

	case Modal.StepAssignMvnToProject:
		currentProj := m.Config.Projects[m.SelectedProject]
		currentBound := currentProj.MavenName
		if currentBound == "" {
			currentBound = "System Default (PATH)"
		}
		modalBody.WriteString(Fmt.Sprintf("📦 Target Workspace: %s (Current Bound Maven: %s)\n\n", currentProj.Name, currentBound))
		modalBody.WriteString("👉 Step 2: Select Profile to Bind to Workspace:\n\n")
		decorateProfiles(modalBody, "Maven", m.SelectedMavenIndex, m.Config.Mavens)
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Browse Maven Installations | [Enter] Confirm Bindings | [Esc] Back\x1b[0m")
	}

	return Lipgloss.Place(m.WindowWidth, m.WindowHeight, Lipgloss.Center, Lipgloss.Center, ModalBox.
		Width(m.WindowWidth-4).
		Render(modalBody.String()), whiteSpace, Lipgloss.WithWhitespaceForeground(DarkerGrey))
}
