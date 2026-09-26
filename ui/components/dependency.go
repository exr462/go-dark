package components

import (
	"fmt"
	"strings"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

// RenderDependencyModal renders an interactive configuration checklist overlay
func RenderDependencyModal(state *model.UIState) string {
	projIdx := state.DepScreen.ActiveProjectIndex
	if projIdx < 0 || projIdx >= len(state.Config.Projects) {
		return "⚠️ No project active for configuration."
	}

	proj := state.Config.Projects[projIdx]
	availableProject := config.AvailableProjects[projIdx]

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(" 🔗 Configure Upstream Dependencies for: \033[1;36m%s\033[0m\n", proj.Name))
	b.WriteString(" ─────────────────────────────────────────────────────────────\n")
	b.WriteString("  [↑/↓] Navigate   •   [Space] Toggle   •   [Enter] Save   •   [Esc] Cancel\n\n")

	// Convert current chosen dependencies into a hashmap for quick looks
	activeDeps := make(map[string]bool)
	for _, d := range availableProject.Dependencies {
		activeDeps[d] = true
	}

	if len(state.DepScreen.AvailableOptions) == 0 {
		b.WriteString("  ❌ No other workspace projects available to link.\n")
		return b.String()
	}

	for i, option := range state.DepScreen.AvailableOptions {
		// Calculate the line cursor arrow
		cursor := "  "
		if state.DepScreen.Cursor == i {
			cursor = " ❯"
		}

		// Calculate the checkbox visual state
		checked := "⬡" // Unchecked circle
		if activeDeps[option] {
			checked = "⬢" // Checked filled circle
		}

		// Render rows highlighting the item currently under the cursor
		if state.DepScreen.Cursor == i {
			b.WriteString(fmt.Sprintf("%s \033[1;33m[%s] %s\033[0m\n", cursor, checked, option))
		} else {
			b.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, option))
		}
	}

	b.WriteString("\n")
	return b.String()
}
