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
	fuzzyHintStyle = lipgloss.NewStyle().Foreground(color.Overlay0)
	fuzzyKeyHint   = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderFuzzyModal(m *model.UIState) string {
	var body strings.Builder

	body.WriteString(renderer.Title.Render("🔍 Workspace Fuzzy Lookup Engine (Ctrl+F)") + "\n\n")

	modeLabel := "📁 [FILES SEARCH MODE]"
	if m.FuzzyMode == model.FuzzyModeContent {
		modeLabel = "🗒 [DEEP CONTENT SCAN MODE]"
	}
	body.WriteString(fmt.Sprintf(
		"Active Filter: %s   %s\n\n",
		renderer.Mode.Render(modeLabel),
		fuzzyHintStyle.Render(fmt.Sprintf("(Press %s to toggle mode)", fuzzyKeyHint.Render("Ctrl+T"))),
	))

	// Render the query input bar
	body.WriteString("💬 Type Filter Query:  " + m.FuzzyQueryInput.View() + "\n")

	// Keep separator length safely within the terminal bounds
	sepWidth := max(m.WindowWidth-8, 20)
	body.WriteString(strings.Repeat("─", sepWidth) + "\n\n")

	// Math bounding variables to prevent pane overflows or distortion loops
	containerWidth := max((m.WindowWidth-10)/2, 20)
	listHeight := max(m.WindowHeight-16, 5)

	m.FuzzyViewer.Width = containerWidth - 4
	m.FuzzyViewer.Height = listHeight - 2

	// --- 1. BUILD LEFT SIDE PANEL (THE RESULTS LIST) ---
	var leftBody strings.Builder
	leftBody.WriteString(renderer.PanelTitle.Render("📋 Ranked Filter Matching Results:") + "\n\n")

	if len(m.FuzzyResults) == 0 {
		leftBody.WriteString(renderer.Meta.Render("  (No matches found)") + "\n")
	} else {
		startIdx := 0
		if m.SelectedFuzzy >= listHeight-2 {
			startIdx = m.SelectedFuzzy - (listHeight - 3)
		}
		endIdx := min(startIdx+(listHeight-2), len(m.FuzzyResults))

		for i := startIdx; i < endIdx; i++ {
			res := m.FuzzyResults[i]
			var lineText string
			if m.FuzzyMode == model.FuzzyModeFiles {
				lineText = fmt.Sprintf("📄 %s", res.FileName)
			} else {
				lineText = fmt.Sprintf("🗒 %s [%d]: %s", res.FileName, res.LineNum, res.Snippet)
			}

			if len(lineText) > containerWidth-6 {
				lineText = lineText[:containerWidth-9] + "..."
			}

			if i == m.SelectedFuzzy {
				leftBody.WriteString(renderer.Selected.Render("> "+lineText) + "\n")
			} else {
				leftBody.WriteString(renderer.Inactive.Render("  "+lineText) + "\n")
			}
		}
	}

	leftPanel := renderer.BorderPane.Width(containerWidth).
		Height(listHeight).
		MaxWidth(containerWidth).
		MaxHeight(listHeight).
		Render(leftBody.String())

	// --- 2. BUILD RIGHT SIDE PANEL (THE FILE CODE PREVIEW) ---
	var rightBody strings.Builder
	rightBody.WriteString(renderer.PanelTitle.Render("🗒 Live File View Context Preview:") + "\n\n")
	rightBody.WriteString(m.FuzzyViewer.View())

	rightPanel := renderer.BorderPane.Width(containerWidth).
		Height(listHeight).
		MaxWidth(containerWidth).
		MaxHeight(listHeight).
		Render(rightBody.String())

	dualPaneGrid := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	body.WriteString(dualPaneGrid + "\n\n")

	body.WriteString(fuzzyHintStyle.Render(fmt.Sprintf(
		"[%s] Navigate | [%s] Open Selection in Workspace | [%s] Dashboard",
		fuzzyKeyHint.Render("↑/↓"),
		fuzzyKeyHint.Render("Enter"),
		fuzzyKeyHint.Render("Esc"),
	)))

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		renderer.ModalBox.Width(m.WindowWidth-4).Render(body.String()),
		renderer.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
