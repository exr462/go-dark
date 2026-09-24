package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderFuzzyModal(m *model.UIState) string {
	var body strings.Builder

	// Style tokens
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("99")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	modeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	panelTitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)

	// FIXED: Layout containers with strict alignment boundaries
	borderPaneStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)

	body.WriteString(titleStyle.Render("🔍 Workspace Fuzzy Lookup Engine (Ctrl+F)") + "\n\n")

	modeLabel := "📁 [FILES SEARCH MODE]"
	if m.FuzzyMode == model.FuzzyModeContent {
		modeLabel = "🗒 [DEEP CONTENT SCAN MODE]"
	}
	body.WriteString(fmt.Sprintf("Active Filter: %s   \x1b[90m(Press Ctrl+T to toggle mode context)\x1b[0m\n\n", modeStyle.Render(modeLabel)))

	// Render the query input bar
	body.WriteString("💬 Type Filter Query:  " + m.FuzzyQueryInput.View() + "\n")

	// Keep separator length safely within the terminal bounds
	sepWidth := max(m.TerminalW-8, 20)
	body.WriteString(strings.Repeat("─", sepWidth) + "\n\n")

	// FIXED: Math bounding variables to prevent pane overflows or distortion loops
	containerWidth := max((m.TerminalW-10)/2, 20)
	listHeight := max(m.TerminalH-16, 5)

	// Update live viewport sub-component parameters to match new geometry shifts
	m.FuzzyViewer.Width = containerWidth - 4
	m.FuzzyViewer.Height = listHeight - 2

	// --- 1. BUILD LEFT SIDE PANEL (THE RESULTS LIST) ---
	var leftBody strings.Builder
	leftBody.WriteString(panelTitleStyle.Render("📋 Ranked Filter Matching Results:") + "\n\n")

	if len(m.FuzzyResults) == 0 {
		leftBody.WriteString("  \x1b[90m(No matches found)\x1b[0m\n")
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

			// Clip single long text lines safely inside left panel boundaries
			if len(lineText) > containerWidth-6 {
				lineText = lineText[:containerWidth-9] + "..."
			}

			if i == m.SelectedFuzzy {
				leftBody.WriteString(selectedStyle.Render("> "+lineText) + "\n")
			} else {
				leftBody.WriteString(inactiveStyle.Render("  "+lineText) + "\n")
			}
		}
	}

	// FIXED: Enforce absolute panel limits on LipGloss style instead of manual loop padding
	leftPanel := borderPaneStyle.Width(containerWidth).
		Height(listHeight).
		MaxWidth(containerWidth).
		MaxHeight(listHeight).
		Render(leftBody.String())

	// --- 2. BUILD RIGHT SIDE PANEL (THE FILE CODE PREVIEW) ---
	var rightBody strings.Builder
	rightBody.WriteString(panelTitleStyle.Render("🗒 Live File View Context Preview:") + "\n\n")
	rightBody.WriteString(m.FuzzyViewer.View())

	rightPanel := borderPaneStyle.Width(containerWidth).
		Height(listHeight).
		MaxWidth(containerWidth).
		MaxHeight(listHeight).
		Render(rightBody.String())

	// Stitch both panel strings together horizontally side-by-side cleanly
	dualPaneGrid := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	body.WriteString(dualPaneGrid + "\n\n")

	body.WriteString("\x1b[90m[↑/↓] Navigate Options | [Enter] Open Selection and Expand to Workspace | [Esc] Dashboard\x1b[0m")

	// Outer main container frame
	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ModalBorderColor).
		Background(ModalBackground).
		Padding(1, 2, 1, 2).
		Width(m.TerminalW - 4).
		Render(body.String())

	return lipgloss.Place(m.TerminalW, m.TerminalH, lipgloss.Center, lipgloss.Center, modalBox)
}
