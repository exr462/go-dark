package component

import (
	"fmt"
	"strings"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

// RenderDependencyModal renders an interactive configuration checklist overlay
func RenderDependencyModal(ui *model.UI) string {
	projIdx := config.FindProject(ui.Config.Projects, ui.DepScreen.ActiveProjectID)
	if projIdx < 0 {
		return "⚠️ No project active for configuration."
	}

	proj := ui.Config.Projects[projIdx]

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(" 🔗 Configure Upstream Dependencies for: \033[1;36m%s\033[0m\n", proj.Name))
	b.WriteString(" ─────────────────────────────────────────────────────────────\n")
	b.WriteString("  [↑/↓] Navigate   •   [Space] Toggle   •   [Enter] Save   •   [Esc] Cancel\n\n")

	// Convert current chosen dependencies into a hashmap for quick looks
	activeDeps := make(map[string]bool)
	for _, d := range proj.DependsOn {
		activeDeps[d] = true
	}

	if !proj.IsLocal() {
		note := fmt.Sprintf("  Managed by team %s", proj.Source)
		if proj.DepsOverridden {
			note += " (local override of the manifest's dependencies)"
		}
		b.WriteString(note + "\n\n")
	}
	for _, unknown := range config.UnknownDependencies(proj, ui.Config.Projects) {
		b.WriteString(fmt.Sprintf("  ⚠️ Unknown dependency %s (not in any subscribed team or local project)\n", unknown))
	}

	if len(ui.DepScreen.AvailableOptions) == 0 {
		b.WriteString("  ❌ No other workspace projects available to link.\n")
		return b.String()
	}

	for i, option := range ui.DepScreen.AvailableOptions {
		// Calculate the line cursor arrow
		cursor := "  "
		if ui.DepScreen.Cursor == i {
			cursor = " ❯"
		}

		// Calculate the checkbox visual state
		checked := "⬡" // Unchecked circle
		if activeDeps[option] {
			checked = "⬢" // Checked filled circle
		}

		// Render rows highlighting the item currently under the cursor
		if ui.DepScreen.Cursor == i {
			b.WriteString(fmt.Sprintf("%s \033[1;33m[%s] %s\033[0m\n", cursor, checked, option))
		} else {
			b.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, option))
		}
	}

	b.WriteString("\n")
	return b.String()
}
