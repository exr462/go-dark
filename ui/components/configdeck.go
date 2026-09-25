package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/renderer"
)

func RenderConfigDeckModal(m *model.UIState) string {
	var body strings.Builder

	// Styling tokens
	body.WriteString(renderer.Title.Render("⚙️ Master Configuration & Environment Deck (Ctrl+Y)") + "\n")
	body.WriteString("Review, expand, or adjust your system parameters across all modules below.\n")
	body.WriteString(strings.Repeat("─", max(m.WindowWidth-8, 20)) + "\n\n")

	// 1. DYNAMICALLY DISPLAY CURRENT ACTIVE CONFIG RECORD VALUES
	body.WriteString(renderer.Section.Render("📊 ACTIVE GLOBAL CONFIGURATION SETTINGS SUMMARY:") + "\n")
	body.WriteString(fmt.Sprintf("  • Global Workspace Base Path : \x1b[36m%s\x1b[0m\n", m.Config.BasePath))
	body.WriteString(fmt.Sprintf("  • Global Git Committer User  : %s\n", m.Config.GitUsername))
	body.WriteString(fmt.Sprintf("  • Global Git Committer Email : %s\n", m.Config.GitEmail))

	// Extract counts
	body.WriteString(fmt.Sprintf("  • Registered Java SDK Pools  : \x1b[32m%d profiles loaded\x1b[0m\n", len(m.Config.JDKs)))
	body.WriteString(fmt.Sprintf("  • Registered Maven Engines   : \x1b[34m%d profiles loaded\x1b[0m\n", len(m.Config.Mavens)))
	body.WriteString(fmt.Sprintf("  • Total Tracked Workspaces   : %d projects configured\n\n", len(m.Config.Projects)))

	body.WriteString(renderer.Section.Render("👉 Choose Category Option to Modify / Add Entries:") + "\n\n")

	// 2. CATEGORY LIST OPTIONS SELECTION MENU BLOCK
	options := []string{
		"[1] Edit Global Core Credentials (BasePath, Git Profile Parameters)",
		"[2] Manage Java SDK Pools (Add / Register JAVA_HOME Paths)",
		"[3] Manage Apache Maven Engines (Add / Register MAVEN_HOME Paths)",
		"[4] Add New Tracked Project Workspace Environment",
	}

	for i, opt := range options {
		if i == m.SelectedConfigOption {
			body.WriteString(renderer.Selected.Render("> "+opt) + "\n")
		} else {
			body.WriteString(renderer.Inactive.Render("  "+opt) + "\n")
		}
	}

	body.WriteString("\n" + strings.Repeat("─", max(m.WindowWidth-8, 20)) + "\n")
	body.WriteString(renderer.Meta.Render(" [↑/↓/j/k] Navigate Options  |  [Enter] Open Target Management Modal  |  [Esc] Close Deck") + "\n")

	return lipgloss.Place(m.WindowWidth, m.WindowHeight, lipgloss.Center, lipgloss.Center, renderer.ModalBox.
		Width(m.WindowWidth-4).
		Render(body.String()))
}
