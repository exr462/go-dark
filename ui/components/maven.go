package components

import (
	Fmt "fmt"
	Strings "strings"

	Lipgloss "github.com/charmbracelet/lipgloss"
	Modal "github.com/exr462/go-build/model"
)

func RenderMvnConfigModal(m Modal.UIState) string {
	var modalBody Strings.Builder

	titleStyle := Lipgloss.NewStyle().Foreground(Lipgloss.Color("33")).Bold(true)
	selectedStyle := Lipgloss.NewStyle().Foreground(Lipgloss.Color("46")).Bold(true)
	inactiveStyle := Lipgloss.NewStyle().Foreground(Lipgloss.Color("250"))
	labelStyle := Lipgloss.NewStyle().Bold(true)
	activeLabelStyle := Lipgloss.NewStyle().Foreground(Lipgloss.Color("33")).Bold(true)

	modalBody.WriteString(titleStyle.Render("🛠️ Apache Maven Manager (Ctrl+U)") + "\n\n")

	switch m.MvnStep {
	case Modal.StepSelectMvnAction:
		modalBody.WriteString("👉 Select Dashboard Action to Perform:\n\n")
		options := []string{"Add New Maven Version Profile to Global Pool", "Assign Selected Maven to Active Workspace"}
		for i, opt := range options {
			if i == m.SelectedMenuIdx {
				modalBody.WriteString(selectedStyle.Render(Fmt.Sprintf("> %s", opt)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(Fmt.Sprintf("  %s", opt)) + "\n")
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Navigate Options | [Enter] Select | [Esc] Close Menu\x1b[0m")

	case Modal.StepAddNewMvnVersion:
		modalBody.WriteString("➕ Step 2: Register a New Maven Environment Context:\n\n")

		nameLabel := labelStyle.Render("  Maven Profile Name (e.g., Maven-3.9.6):")
		if m.FocusedInput == 9 {
			nameLabel = activeLabelStyle.Render("> Maven Profile Name (e.g., Maven-3.9.6):")
		}
		modalBody.WriteString(nameLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[9].View() + "\n\n")

		pathLabel := labelStyle.Render("  Maven Home Directory (MAVEN_HOME / M2_HOME):")
		if m.FocusedInput == 10 {
			pathLabel = activeLabelStyle.Render("> Maven Home Directory (MAVEN_HOME / M2_HOME):")
		}
		modalBody.WriteString(pathLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[10].View() + "\n")

		modalBody.WriteString("\n\x1b[90m[Tab] Swap Fields | [Enter] Save Profile Entry | [Esc] Cancel and Return\x1b[0m")

	case Modal.StepAssignMvnToProject:
		currentProj := m.Config.Projects[m.SelectedProj]
		currentBound := currentProj.MavenName
		if currentBound == "" {
			currentBound = "System Default (PATH)"
		}
		modalBody.WriteString(Fmt.Sprintf("📦 Target Workspace: %s (Current Bound Maven: %s)\n\n", currentProj.Name, currentBound))
		modalBody.WriteString("👉 Step 2: Select Profile to Bind to Workspace:\n\n")

		if len(m.Config.Mavens) == 0 {
			modalBody.WriteString("  ❌ No configured Maven profiles found in database.\n  Go back and add one first.")
		} else {
			for i, mvn := range m.Config.Mavens {
				if i == m.SelectedMvnIdx {
					modalBody.WriteString(selectedStyle.Render(Fmt.Sprintf("> %s (%s)", mvn.Name, mvn.Path)) + "\n")
				} else {
					modalBody.WriteString(inactiveStyle.Render(Fmt.Sprintf("  %s", mvn.Name)) + "\n")
				}
			}
		}
		modalBody.WriteString("\n\x1b[90m[↑/↓/j/k] Browse Maven Installations | [Enter] Confirm Bindings | [Esc] Back\x1b[0m")
	}

	modalBox := Lipgloss.NewStyle().
		Border(Lipgloss.DoubleBorder()).
		BorderForeground(Lipgloss.Color("33")).
		Background(Lipgloss.Color("234")).
		Padding(1, 4, 1, 4).
		Width(75).
		Render(modalBody.String())

	return Lipgloss.Place(m.TerminalW, m.TerminalH, Lipgloss.Center, Lipgloss.Center, modalBox, Lipgloss.WithWhitespaceChars("░"), Lipgloss.WithWhitespaceForeground(Lipgloss.Color("236")))
}
