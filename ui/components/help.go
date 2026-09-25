package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/renderer"
)

func RenderHelpModal(m *model.UIState) string {
	var body strings.Builder
	body.WriteString(renderer.Title.Render("📖 Go-Dark Command & Shortcuts Cheat Sheet") + "\n\n")
	body.WriteString(renderer.Section.Render("🌐 GLOBAL PANE CONTROLS") + "\n")
	body.WriteString(renderer.Row("Tab", "Cycle active panel focus (Projects ➜ Tree ➜ Menu)"))
	body.WriteString(renderer.Row("↑ / ↓ / j / k", "Navigate items in the currently focused panel"))
	body.WriteString(renderer.Row("Enter", "Confirm selected element or execute focused action"))
	body.WriteString(renderer.Row("?", "Toggle this Help utility layout on/off"))
	body.WriteString(renderer.Row("q", "Quit application immediately"))
	body.WriteString("\n")

	body.WriteString(renderer.Section.Render("🎛️ MODAL INTERFACE TOGGLES (From Dashboard)" + "\n"))
	body.WriteString(renderer.Row("Ctrl+P", "Open 'Add New Project' configuration modal form"))
	body.WriteString(renderer.Row("Ctrl+G", "Open 'Git Hub Operations Center' control module"))
	body.WriteString(renderer.Row("Ctrl+J", "Open 'Java Environment SDK Manager' (JDK) module"))
	body.WriteString(renderer.Row("Ctrl+M", "Open 'Maven Manager' (MVN) module"))
	body.WriteString(renderer.Row("Ctrl+D", "Open 'Docker Manager' (DOCKER) module"))
	body.WriteString(renderer.Row("Ctrl+B", "Open 'Build' (JDK) module"))
	body.WriteString(renderer.Row("Ctrl+F", "Open 'Fuzzy finder"))
	body.WriteString(renderer.Row("Ctrl+E", "Opens a terminal and nvim to edit the file"))
	body.WriteString("\n")

	body.WriteString(renderer.Section.Render("✍️ DATA ENTRY FORMS (Installer / Add Project / Add JDK / Add MVN)"))
	body.WriteString(renderer.Row("Tab / Down", "Move cursor focus forward to next input field"))
	body.WriteString(renderer.Row("Shift+Tab / Up", "Move cursor focus backward to previous input field"))
	body.WriteString(renderer.Row("Enter", "Submit form data, save configurations, and commit files"))
	body.WriteString(renderer.Row("Esc", "Safely exit active wizard without saving records") + "\n\n")

	body.WriteString(renderer.Author.Render("🤖 Author: Daniel Noulet © 2026"))

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		renderer.ModalBox.
			Width(m.WindowWidth-4).
			Render(body.String()),
		renderer.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.DarkerGrey),
	)
}
