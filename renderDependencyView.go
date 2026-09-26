package main

import (
	"fmt"
	"strings"

	"github.com/exr462/go-dark/config"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) renderDependencyView() string {
	projIdx := m.state.DepScreen.ActiveProjectIndex
	proj := config.AvailableProjects[projIdx]

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔗  Configure Dependencies for: \033[1;36m%s\033[0m\n", proj.Name))
	b.WriteString("Use [↑/↓] to navigate, [Space] to toggle, [Enter] to save, [Esc] to cancel.\n\n")

	// Build a hash map of current active selections for quick lookups
	activeDeps := make(map[string]bool)
	for _, d := range proj.Dependencies {
		activeDeps[d] = true
	}

	for i, option := range m.state.DepScreen.AvailableOptions {
		// Draw cursor point indicator
		cursor := " "
		if m.state.DepScreen.Cursor == i {
			cursor = "❯"
		}

		// Draw selection checkbox status
		checked := " "
		if activeDeps[option] {
			checked = "⬢" // Filled indicator for checked dependency
		} else {
			checked = "⬡" // Empty indicator
		}

		// Format output row
		if m.state.DepScreen.Cursor == i {
			b.WriteString(fmt.Sprintf("%s [%s] \033[1;33m%s\033[0m\n", cursor, checked, option))
		} else {
			b.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, option))
		}
	}

	return b.String()
}
