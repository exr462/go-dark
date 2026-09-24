package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-build/model"
)

func RenderHelpModal(m model.UIState) string {
	var body strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Width(14)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	body.WriteString(titleStyle.Render("📖 RepoDeck Command & Shortcuts Cheat Sheet") + "\n\n")

	// Helper wrapper function to draw unified lines
	row := func(key, desc string) string {
		return keyStyle.Render(" "+key) + descStyle.Render(desc) + "\n"
	}

	body.WriteString(sectionStyle.Render("🌐 GLOBAL PANE CONTROLS") + "\n")
	body.WriteString(row("Tab", "Cycle active panel focus (Projects ➜ Tree ➜ Menu)"))
	body.WriteString(row("↑ / ↓ / j / k", "Navigate items in the currently focused panel"))
	body.WriteString(row("Enter", "Confirm selected element or execute focused action"))
	body.WriteString(row("?", "Toggle this Help utility layout on/off"))
	body.WriteString(row("q / Ctrl+C", "Quit application immediately"))
	body.WriteString("\n")

	body.WriteString(sectionStyle.Render("🎛️ MODAL INTERFACE TOGGLES (From Dashboard)"))
	body.WriteString(row("Ctrl+N", "Open 'Add New Project' configuration modal form"))
	body.WriteString(row("Ctrl+G", "Open 'Git Hub Operations Center' control module"))
	body.WriteString(row("Ctrl+J", "Open 'Java Environment SDK Manager' (JDK) module"))
	body.WriteString("\n")

	body.WriteString(sectionStyle.Render("✍️ DATA ENTRY FORMS (Installer / Add Project / Add JDK)"))
	body.WriteString(row("Tab / Down", "Move cursor focus forward to next input field"))
	body.WriteString(row("Shift+Tab / Up", "Move cursor focus backward to previous input field"))
	body.WriteString(row("Enter", "Submit form data, save configurations, and commit files"))
	body.WriteString(row("Esc", "Safely exit active wizard without saving records"))

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("86")).
		Background(lipgloss.Color("234")).
		Padding(1, 4, 1, 4).
		Width(75).
		Render(body.String())

	return lipgloss.Place(
		m.TerminalW, m.TerminalH,
		lipgloss.Center, lipgloss.Center,
		helpBox,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}
