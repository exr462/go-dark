package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/renderer"
)

var (
	mvnHintStyle = lipgloss.NewStyle().Foreground(color.Overlay0)
	mvnKeyHint   = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderMvnConfigModal(m *model.UIState) string {
	var modalBody strings.Builder

	modalBody.WriteString(renderer.Title.Render("🛠️ Apache Maven Manager (Ctrl+U)") + "\n\n")

	switch m.MavenStep {
	case model.StepSelectMvnAction:
		modalBody.WriteString(renderer.Section.Render("👉 Select Action to Perform:") + "\n\n")
		options := []string{"Add New Maven Version Profile to Global Pool", "Assign Selected Maven to Active Workspace"}
		for i, opt := range options {
			if i == m.SelectedMenuIndex {
				modalBody.WriteString(renderer.Selected.Render(fmt.Sprintf("> %s", opt)) + "\n")
			} else {
				modalBody.WriteString(renderer.Inactive.Render(fmt.Sprintf("  %s", opt)) + "\n")
			}
		}
		modalBody.WriteString("\n" + mvnHintStyle.Render(fmt.Sprintf(
			"[%s] Navigate | [%s] Select | [%s] Close Menu",
			mvnKeyHint.Render("↑/↓/j/k"),
			mvnKeyHint.Render("Enter"),
			mvnKeyHint.Render("Esc"),
		)))

	case model.StepAddNewMvnVersion:
		modalBody.WriteString(renderer.Section.Render("➕ Step 2: Register a New Maven Environment Context:") + "\n\n")

		nameLabel := renderer.Bold.Render("  Maven Profile Name (e.g., Maven-3.9.6):")
		if m.FocusedInput == 9 {
			nameLabel = renderer.ActiveLabel.Render("> Maven Profile Name (e.g., Maven-3.9.6):")
		}
		modalBody.WriteString(nameLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[9].View() + "\n\n")

		pathLabel := renderer.Bold.Render("  Maven Home Directory (MAVEN_HOME / M2_HOME):")
		if m.FocusedInput == 10 {
			pathLabel = renderer.ActiveLabel.Render("> Maven Home Directory (MAVEN_HOME / M2_HOME):")
		}
		modalBody.WriteString(pathLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[10].View() + "\n")

		modalBody.WriteString("\n" + mvnHintStyle.Render(fmt.Sprintf(
			"[%s] Swap Fields | [%s] Save Profile | [%s] Cancel & Return",
			mvnKeyHint.Render("Tab"),
			mvnKeyHint.Render("Enter"),
			mvnKeyHint.Render("Esc"),
		)))

	case model.StepAssignMvnToProject:
		currentProj := m.Config.Projects[m.SelectedProject]
		currentBound := currentProj.MavenName
		if currentBound == "" {
			currentBound = "System Default (PATH)"
		}
		modalBody.WriteString(fmt.Sprintf("📦 Target Workspace: %s (Current Bound Maven: %s)\n\n", currentProj.Name, currentBound))
		modalBody.WriteString(renderer.Section.Render("👉 Step 2: Select Profile to Bind to Workspace:") + "\n\n")
		renderer.DecorateProfiles(&modalBody, "Maven", m.SelectedMavenIndex, m.Config.Mavens)
		modalBody.WriteString("\n" + mvnHintStyle.Render(fmt.Sprintf(
			"[%s] Browse Maven Installations | [%s] Confirm Binding | [%s] Back",
			mvnKeyHint.Render("↑/↓/j/k"),
			mvnKeyHint.Render("Enter"),
			mvnKeyHint.Render("Esc"),
		)))
	}

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		renderer.ModalBox.Width(min(m.WindowWidth-4, 85)).Render(modalBody.String()),
		renderer.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
