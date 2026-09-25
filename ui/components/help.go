package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderHelpModal(m model.UIState) string {
	var body strings.Builder
	body.WriteString(TitleStyle.Render("📖 Go-Dark Command & Shortcuts Cheat Sheet") + "\n\n")
	body.WriteString(sectionStyle.Render("🌐 GLOBAL PANE CONTROLS") + "\n")
	body.WriteString(row("Tab", "Cycle active panel focus (Projects ➜ Tree ➜ Menu)"))
	body.WriteString(row("↑ / ↓ / j / k", "Navigate items in the currently focused panel"))
	body.WriteString(row("Enter", "Confirm selected element or execute focused action"))
	body.WriteString(row("?", "Toggle this Help utility layout on/off"))
	body.WriteString(row("q", "Quit application immediately"))
	body.WriteString("\n")

	body.WriteString(sectionStyle.Render("🎛️ MODAL INTERFACE TOGGLES (From Dashboard)" + "\n"))
	body.WriteString(row("Ctrl+P", "Open 'Add New Project' configuration modal form"))
	body.WriteString(row("Ctrl+G", "Open 'Git Hub Operations Center' control module"))
	body.WriteString(row("Ctrl+J", "Open 'Java Environment SDK Manager' (JDK) module"))
	body.WriteString(row("Ctrl+M", "Open 'Maven Manager' (MVN) module"))
	body.WriteString(row("Ctrl+D", "Open 'Docker Manager' (DOCKER) module"))
	body.WriteString(row("Ctrl+B", "Open 'Build' (JDK) module"))
	body.WriteString(row("Ctrl+F", "Open 'Fuzzy finder"))
	body.WriteString(row("Ctrl+E", "Opens a terminal and nvim to edit the file"))
	body.WriteString("\n")

	body.WriteString(sectionStyle.Render("✍️ DATA ENTRY FORMS (Installer / Add Project / Add JDK / Add MVN)"))
	body.WriteString(row("Tab / Down", "Move cursor focus forward to next input field"))
	body.WriteString(row("Shift+Tab / Up", "Move cursor focus backward to previous input field"))
	body.WriteString(row("Enter", "Submit form data, save configurations, and commit files"))
	body.WriteString(row("Esc", "Safely exit active wizard without saving records") + "\n\n")

	body.WriteString(authorStyle.Render("🤖 Author: Daniel Noulet © 2026"))

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		ModalBox.
			Width(m.WindowWidth-4).
			Render(body.String()),
		whiteSpace,
		lipgloss.WithWhitespaceForeground(DarkerGrey),
	)
}
