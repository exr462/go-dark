package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	mvnHintStyle = lipgloss.NewStyle().Foreground(color.Overlay0)
	mvnKeyHint   = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderMvnConfigModal(ui *model.UI) string {
	var modalBody strings.Builder

	modalBody.WriteString(decorator.Title.Render("🛠️ Apache Maven Manager (Ctrl+U)") + "\n\n")

	switch ui.MavenStep {
	case model.StepSelectMvnAction:
		modalBody.WriteString(decorator.Section.Render("👉 Select Action to Perform:") + "\n\n")
		options := []string{"Add New Maven Version Profile to Global Pool", "Assign Selected Maven to Active Workspace"}
		for i, opt := range options {
			if i == ui.SelectedMenuIndex {
				modalBody.WriteString(decorator.Selected.Render(fmt.Sprintf("> %s", opt)) + "\n")
			} else {
				modalBody.WriteString(decorator.Inactive.Render(fmt.Sprintf("  %s", opt)) + "\n")
			}
		}
		modalBody.WriteString("\n" + mvnHintStyle.Render(fmt.Sprintf(
			"[%s] Navigate | [%s] Select | [%s] Close Menu",
			mvnKeyHint.Render("↑/↓/j/k"),
			mvnKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			mvnKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)))

	case model.StepAddNewMvnVersion:
		modalBody.WriteString(decorator.Section.Render("➕ Step 2: Register a New Maven Environment Context:") + "\n\n")

		nameLabel := decorator.Bold.Render("  Maven Profile Name (e.g., Maven-3.9.6):")
		if ui.FocusedInput == kbd.MvnName {
			nameLabel = decorator.ActiveLabel.Render("> Maven Profile Name (e.g., Maven-3.9.6):")
		}
		modalBody.WriteString(nameLabel + "\n")
		modalBody.WriteString("  " + ui.Inputs[kbd.MvnName].View() + "\n\n")

		pathLabel := decorator.Bold.Render("  Maven Home Directory (MAVEN_HOME / M2_HOME):")
		if ui.FocusedInput == kbd.MvnPath {
			pathLabel = decorator.ActiveLabel.Render("> Maven Home Directory (MAVEN_HOME / M2_HOME):")
		}
		modalBody.WriteString(pathLabel + "\n")
		modalBody.WriteString("  " + ui.Inputs[kbd.MvnPath].View() + "\n\n")

		modalBody.WriteString("\n" + mvnHintStyle.Render(fmt.Sprintf(
			"[%s] Swap Fields | [%s] Save Profile | [%s] Cancel & Return",
			mvnKeyHint.Render("Tab"),
			mvnKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			mvnKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)))

	case model.StepAssignMvnToProject:
		currentProj := ui.Config.Projects[ui.SelectedProject]
		currentBound := currentProj.MavenName
		if currentBound == "" {
			currentBound = "System Default (PATH)"
		}
		modalBody.WriteString(fmt.Sprintf("📦 Target Workspace: %s (Current Bound Maven: %s)\n\n", currentProj.Name, currentBound))
		modalBody.WriteString(decorator.Section.Render("👉 Step 2: Select Profile to Bind to Workspace:") + "\n\n")
		decorator.DecorateProfiles(&modalBody, "Maven", ui.SelectedMavenIndex, ui.Config.Mavens)
		modalBody.WriteString("\n" + mvnHintStyle.Render(fmt.Sprintf(
			"[%s] Browse Maven Installations | [%s] Confirm Binding | [%s] Back",
			mvnKeyHint.Render("↑/↓/j/k"),
			mvnKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			mvnKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)))
	}

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(ui.WindowWidth-4, 85)).Render(modalBody.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
