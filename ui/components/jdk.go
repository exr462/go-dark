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
	jdkHintStyle = lipgloss.NewStyle().Foreground(color.Overlay0)
	jdkKeyHint   = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderJDKConfigModal(m *model.UIState) string {
	var modalBody strings.Builder

	modalBody.WriteString(renderer.Title.Render("☕ Java Environment Manager (Ctrl+J)") + "\n\n")

	switch m.JDKStep {
	case model.StepSelectJDKAction:
		modalBody.WriteString(renderer.Section.Render("👉 Select Action to Perform:") + "\n\n")
		options := []string{"Add New JDK Version Profile to Global Pool", "Assign Selected JDK to Active Workspace"}
		for i, opt := range options {
			if i == m.SelectedMenuIndex {
				modalBody.WriteString(renderer.Selected.Render(fmt.Sprintf("> %s", opt)) + "\n")
			} else {
				modalBody.WriteString(renderer.Inactive.Render(fmt.Sprintf("  %s", opt)) + "\n")
			}
		}
		modalBody.WriteString("\n" + jdkHintStyle.Render(fmt.Sprintf(
			"[%s] Navigate | [%s] Select | [%s] Close Menu",
			jdkKeyHint.Render("↑/↓/j/k"),
			jdkKeyHint.Render("Enter"),
			jdkKeyHint.Render("Esc"),
		)))

	case model.StepAddNewJDKVersion:
		modalBody.WriteString(renderer.Section.Render("➕ Step 2: Register a New Java Environment Build Context:") + "\n\n")

		nameLabel := renderer.Bold.Render("  JDK Profile Logical Name (e.g. OpenJDK-17):")
		if m.FocusedInput == 7 {
			nameLabel = renderer.ActiveLabel.Render("> JDK Profile Logical Name (e.g. OpenJDK-17):")
		}
		modalBody.WriteString(nameLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[7].View() + "\n\n")

		pathLabel := renderer.Bold.Render("  JDK Home Absolute Path (JAVA_HOME):")
		if m.FocusedInput == 8 {
			pathLabel = renderer.ActiveLabel.Render("> JDK Home Absolute Path (JAVA_HOME):")
		}
		modalBody.WriteString(pathLabel + "\n")
		modalBody.WriteString("  " + m.Inputs[8].View() + "\n")

		modalBody.WriteString("\n" + jdkHintStyle.Render(fmt.Sprintf(
			"[%s] Swap Fields | [%s] Save Profile | [%s] Cancel & Return",
			jdkKeyHint.Render("Tab"),
			jdkKeyHint.Render("Enter"),
			jdkKeyHint.Render("Esc"),
		)))

	case model.StepAssignJDKToProject:
		if m.SelectedProject >= len(m.Config.Projects) {
			modalBody.WriteString(renderer.Error.Render("❌ Error: No valid project context active."))
		} else {
			currentProj := m.Config.Projects[m.SelectedProject]
			currentBound := currentProj.JDKName
			if currentBound == "" {
				currentBound = "System Default"
			}
			modalBody.WriteString(fmt.Sprintf("📦 Target Workspace: %s (Current Bound JDK: %s)\n\n", currentProj.Name, currentBound))
			modalBody.WriteString(renderer.Section.Render("👉 Step 2: Select Profile to Bind to Workspace Environment:") + "\n\n")
			renderer.DecorateProfiles(&modalBody, "JDK", m.SelectedJDKIndex, m.Config.JDKs)
		}
		modalBody.WriteString("\n" + jdkHintStyle.Render(fmt.Sprintf(
			"[%s] Browse SDKs | [%s] Confirm Binding | [%s] Back",
			jdkKeyHint.Render("↑/↓/j/k"),
			jdkKeyHint.Render("Enter"),
			jdkKeyHint.Render("Esc"),
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
