package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	deckPathVal   = lipgloss.NewStyle().Foreground(color.Sky).Bold(true)
	deckJDKVal    = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	deckMvnVal    = lipgloss.NewStyle().Foreground(color.Blue).Bold(true)
	deckProjVal   = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	deckHintStyle = lipgloss.NewStyle().Foreground(color.Overlay0)
	deckKeyHint   = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderConfigDeckModal(m *model.UIState) string {
	var body strings.Builder

	body.WriteString(decorator.Title.Render("⚙️ Master Configuration & Environment Deck (Ctrl+Y)") + "\n")
	body.WriteString(decorator.Description.Render("Review, expand, or adjust your system parameters across all modules below.") + "\n")
	body.WriteString(strings.Repeat("─", max(m.WindowWidth-8, 20)) + "\n\n")

	// 1. DYNAMICALLY DISPLAY CURRENT ACTIVE CONFIG RECORD VALUES
	body.WriteString(decorator.Section.Render("📊 ACTIVE GLOBAL CONFIGURATION SETTINGS SUMMARY:") + "\n")
	body.WriteString(fmt.Sprintf("  • Global Workspace Base Path : %s\n", deckPathVal.Render(m.Config.BasePath)))
	body.WriteString(fmt.Sprintf("  • Global Git Committer User  : %s\n", m.Config.GitUsername))
	body.WriteString(fmt.Sprintf("  • Global Git Committer Email : %s\n", m.Config.GitEmail))

	body.WriteString(fmt.Sprintf("  • Registered Java SDK Pools  : %s\n", deckJDKVal.Render(fmt.Sprintf("%d profiles loaded", len(m.Config.JDKs)))))
	body.WriteString(fmt.Sprintf("  • Registered Maven Engines   : %s\n", deckMvnVal.Render(fmt.Sprintf("%d profiles loaded", len(m.Config.Mavens)))))
	body.WriteString(fmt.Sprintf("  • Total Tracked Workspaces   : %s\n\n", deckProjVal.Render(fmt.Sprintf("%d projects configured", len(m.Config.Projects)))))

	body.WriteString(decorator.Section.Render("👉 Choose Category Option to Modify / Add Entries:") + "\n\n")

	options := []string{
		"[1] Edit Global Core Credentials (BasePath, Git Profile Parameters)",
		"[2] Manage Java SDK Pools (Add / Register JAVA_HOME Paths)",
		"[3] Manage Apache Maven Engines (Add / Register MAVEN_HOME Paths)",
		"[4] Add New Tracked Project Workspace Environment",
	}

	for i, opt := range options {
		if i == m.SelectedConfigOption {
			body.WriteString(decorator.Selected.Render("> "+opt) + "\n")
		} else {
			body.WriteString(decorator.Inactive.Render("  "+opt) + "\n")
		}
	}

	body.WriteString("\n" + strings.Repeat("─", max(m.WindowWidth-8, 20)) + "\n")
	body.WriteString(deckHintStyle.Render(fmt.Sprintf(
		"[%s] Navigate Options | [%s] Open Selected Management Modal | [%s] Close Deck",
		deckKeyHint.Render("↑/↓/j/k"),
		deckKeyHint.Render("Enter"),
		deckKeyHint.Render("Esc"),
	)) + "\n")

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(m.WindowWidth-4).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
