package panel

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	selectedProjStyle = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	localProjStyle    = lipgloss.NewStyle().Foreground(color.Blue)
	remoteProjStyle   = lipgloss.NewStyle().Foreground(color.Subtext0)
	localBadgeStyle   = lipgloss.NewStyle().Foreground(color.Teal)
	remoteBadgeStyle  = lipgloss.NewStyle().Foreground(color.Overlay0)
	sessionBadgeStyle = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	treeFolderStyle   = lipgloss.NewStyle().Foreground(color.Blue).Bold(true)
	treeFileStyle     = lipgloss.NewStyle().Foreground(color.Text)
	selectedTreeStyle = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	mutedTextStyle    = lipgloss.NewStyle().Foreground(color.Overlay0).Italic(true)
)

func RenderMainBody(ui *model.UI) string {
	rightPaneMultiplier := 8
	leftPaneMultiplier := 2
	widthDivider := 10
	widthPadding := 2
	paneHeight := max(ui.WindowHeight-7, 5)

	leftWidth := ((ui.WindowWidth / widthDivider) * leftPaneMultiplier) - widthPadding
	centerWidth := ((ui.WindowWidth / widthDivider) * rightPaneMultiplier) - widthPadding
	// Rows available under the title + blank line, keeping one for the
	// "x-y of n" position line.
	listRows := max(paneHeight-3, 1)

	// 1. Render Left Column (Projects)
	var projList strings.Builder
	projList.WriteString(fitRow(decorator.Title.Render("📁 Configured Projects"), leftWidth) + "\n\n")

	start, end := scrollWindow(len(ui.Config.Projects), ui.SelectedProject, listRows)
	for i := start; i < end; i++ {
		p := ui.Config.Projects[i]
		var row string
		sessionLabel := ""
		if build := session.RunningBuild(ui.Shells, p.ID); build != nil {
			sessionLabel = sessionBadgeStyle.Render(fmt.Sprintf(" (#%d ⏳)", build.ID))
		}

		statusBadge := localBadgeStyle.Render("[local]")
		if !p.Fetched {
			statusBadge = remoteBadgeStyle.Render("[remote]")
		}

		switch {
		case i == ui.SelectedProject:
			row = fmt.Sprintf("> %s [%s] %s%s", selectedProjStyle.Render(p.Name), p.Type, statusBadge, sessionLabel)
		case p.Fetched:
			row = fmt.Sprintf("  %s [%s] %s%s", localProjStyle.Render(p.Name), p.Type, statusBadge, sessionLabel)
		default:
			row = fmt.Sprintf("  %s [%s] %s%s", remoteProjStyle.Render(p.Name), p.Type, statusBadge, sessionLabel)
		}
		projList.WriteString(fitRow(row, leftWidth) + "\n")
	}
	projList.WriteString(scrollPosition(start, end, len(ui.Config.Projects)))

	leftBoxStyle := decorator.UnfocusedBorder.Background(color.Base)
	if ui.ActiveFocus == model.FocusProjects && ui.ViewState == model.StateDashboard {
		leftBoxStyle = decorator.FocusedBorder.Background(color.Base)
	}
	leftPanel := leftBoxStyle.
		Width(leftWidth + decorator.BorderFrame).
		Height(paneHeight + decorator.BorderFrame).
		Render(projList.String())

	// 2. Render Directory Tree in the Center Column
	var treeList strings.Builder
	treeList.WriteString(fitRow(decorator.Title.Render("🌿 Workspace Directory Tree (Ctrl+E to edit in external editor)"), centerWidth) + "\n\n")

	if len(ui.TreeNodes) == 0 {
		treeList.WriteString(mutedTextStyle.Render("  (No workspace directories loaded. Use Ctrl+G to clone/checkout projects)"))
	} else {
		start, end := scrollWindow(len(ui.TreeNodes), ui.SelectedFile, listRows)
		for i := start; i < end; i++ {
			node := ui.TreeNodes[i]
			indent := strings.Repeat("  ", node.Depth)

			prefix := "📄 "
			if node.IsDir {
				if node.IsExpanded {
					prefix = "📂 "
				} else {
					prefix = "📁 "
				}
			}

			lineText := fmt.Sprintf("%s%s%s", indent, prefix, node.Name)

			var row string
			switch {
			case i == ui.SelectedFile:
				row = fmt.Sprintf("> %s", selectedTreeStyle.Render(lineText))
			case node.IsDir:
				row = fmt.Sprintf("  %s", treeFolderStyle.Render(lineText))
			default:
				row = fmt.Sprintf("  %s", treeFileStyle.Render(lineText))
			}
			treeList.WriteString(fitRow(row, centerWidth) + "\n")
		}
		treeList.WriteString(scrollPosition(start, end, len(ui.TreeNodes)))
	}

	centerBoxStyle := decorator.UnfocusedBorder.Background(color.Base)
	if ui.ActiveFocus == model.FocusTree && ui.ViewState == model.StateDashboard {
		centerBoxStyle = decorator.FocusedBorder.Background(color.Base)
	}
	centerPanel := centerBoxStyle.
		Width(centerWidth + decorator.BorderFrame).
		Height(paneHeight + decorator.BorderFrame).
		Render(treeList.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerPanel)
}

// scrollWindow returns the [start, end) slice of n rows to show in visible
// rows, keeping the selected row roughly centered.
func scrollWindow(n, selected, visible int) (start, end int) {
	start = max(min(selected-visible/2, n-visible), 0)
	return start, min(start+visible, n)
}

// scrollPosition shows "x-y of n" when the list doesn't fit.
func scrollPosition(start, end, n int) string {
	if start == 0 && end == n {
		return ""
	}
	return mutedTextStyle.Render(fmt.Sprintf("  %d-%d of %d", start+1, end, n))
}

// fitRow cuts a styled row to the pane's inner width so it never wraps
// (a wrapped row pushed the whole dashboard past the bottom of the screen).
func fitRow(row string, width int) string {
	return ansi.Truncate(row, max(width, 1), "…")
}
