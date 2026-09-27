package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	fuzzyHintStyle = lipgloss.NewStyle().Foreground(color.Overlay0)
	fuzzyKeyHint   = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderFuzzyModal(ui *model.UI) string {
	var body strings.Builder

	body.WriteString(decorator.Title.Render("🔍 Workspace Fuzzy Lookup Engine (Ctrl+F)") + "\n\n")

	modeLabel := "📁 [FILES SEARCH MODE]"
	if ui.FuzzyMode == model.FuzzyModeContent {
		modeLabel = "🗒 [DEEP CONTENT SCAN MODE]"
	}
	body.WriteString(fmt.Sprintf(
		"Active Filter: %s   %s\n\n",
		decorator.Mode.Render(modeLabel),
		fuzzyHintStyle.Render(fmt.Sprintf("(Press %s to toggle mode)", fuzzyKeyHint.Render("Ctrl+T"))),
	))

	// Render the query input bar
	body.WriteString("💬 Type Filter Query:  " + ui.FuzzyQueryInput.View() + "\n")

	// Keep separator length safely within the terminal bounds
	sepWidth := max(ui.WindowWidth-8, 20)
	body.WriteString(strings.Repeat("─", sepWidth) + "\n\n")

	// Math bounding variables to prevent pane overflows or distortion loops
	containerWidth := max((ui.WindowWidth-10)/2, 20)
	listHeight := max(ui.WindowHeight-16, 5)

	ui.FuzzyViewer.Width = containerWidth - 4
	ui.FuzzyViewer.Height = listHeight - 2

	// --- 1. BUILD LEFT SIDE PANEL (THE RESULTS LIST) ---
	var leftBody strings.Builder
	leftBody.WriteString(decorator.PanelTitle.Render("📋 Ranked Filter Matching Results:") + "\n\n")

	if len(ui.FuzzyResults) == 0 {
		leftBody.WriteString(decorator.Meta.Render("  (No matches found)") + "\n")
	} else {
		startIdx := 0
		if ui.SelectedFuzzy >= listHeight-2 {
			startIdx = ui.SelectedFuzzy - (listHeight - 3)
		}
		endIdx := min(startIdx+(listHeight-2), len(ui.FuzzyResults))

		for i := startIdx; i < endIdx; i++ {
			res := ui.FuzzyResults[i]
			var lineText string
			if ui.FuzzyMode == model.FuzzyModeFiles {
				lineText = fmt.Sprintf("📄 %s", res.FileName)
			} else {
				lineText = fmt.Sprintf("🗒 %s [%d]: %s", res.FileName, res.LineNum, res.Snippet)
			}

			if len(lineText) > containerWidth-6 {
				lineText = lineText[:containerWidth-9] + "..."
			}

			if i == ui.SelectedFuzzy {
				leftBody.WriteString(decorator.Selected.Render("> "+lineText) + "\n")
			} else {
				leftBody.WriteString(decorator.Inactive.Render("  "+lineText) + "\n")
			}
		}
	}

	leftPanel := decorator.BorderPane.Width(containerWidth).
		Height(listHeight).
		MaxWidth(containerWidth).
		MaxHeight(listHeight).
		Render(leftBody.String())

	// --- 2. BUILD RIGHT SIDE PANEL (THE FILE CODE PREVIEW) ---
	var rightBody strings.Builder
	rightBody.WriteString(decorator.PanelTitle.Render("🗒 Live File View Context Preview:") + "\n\n")
	rightBody.WriteString(ui.FuzzyViewer.View())

	rightPanel := decorator.BorderPane.Width(containerWidth).
		Height(listHeight).
		MaxWidth(containerWidth).
		MaxHeight(listHeight).
		Render(rightBody.String())

	dualPaneGrid := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	body.WriteString(dualPaneGrid + "\n\n")

	body.WriteString(fuzzyHintStyle.Render(fmt.Sprintf(
		"[%s] Navigate | [%s] Open Selection in Workspace | [%s] Dashboard",
		fuzzyKeyHint.Render("↑/↓"),
		fuzzyKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
		fuzzyKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
	)))

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
