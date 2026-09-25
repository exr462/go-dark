package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/renderer"
)

func RenderMainBody(m *model.UIState) string {
	rightPaneMultiplier := 8
	leftPaneMultiplier := 2
	widthDivider := 10
	widthPadding := 2
	paneHeight := m.WindowHeight - 7
	// 1. Render Left Column (Projects)
	var projList strings.Builder
	projList.WriteString(renderer.Title.Render("📁 Configured Projects") + "\n\n")
	for i, p := range m.Config.Projects {
		sessionLabel := ""
		for id, sess := range m.Sessions {
			if sess.ProjectName == p.Name && sess.IsRunning {
				sessionLabel = fmt.Sprintf(" \x1b[33;1m(ID:%d ⏳)\x1b[0m", id)
				break
			}
		}
		if i == m.SelectedProject {
			projList.WriteString(fmt.Sprintf("> \x1b[36m%s\x1b[0m [%s]%s\n", p.Name, p.Type, sessionLabel))
		} else {
			if p.Fetched {
				projList.WriteString(fmt.Sprintf("  \x1b[34m%s\x1b[0m [%s]%s\n", p.Name, p.Type, sessionLabel))
			} else {
				projList.WriteString(fmt.Sprintf("  %s [%s]%s\n", p.Name, p.Type, sessionLabel))
			}
		}
	}
	leftBoxStyle := renderer.UnfocusedBorder
	if m.ActiveFocus == model.FocusProjects && m.ViewState == model.StateDashboard {
		leftBoxStyle = renderer.FocusedBorder
	}
	leftPanel := leftBoxStyle.Width(((m.WindowWidth / widthDivider) * leftPaneMultiplier) - widthPadding).Height(paneHeight).Render(projList.String())

	// 2. FIXED: RENDER THE DYNAMIC TREE HIERARCHY IN THE CENTER COLUMN
	var treeList strings.Builder
	treeList.WriteString(renderer.Title.Render("🌿 Workspace Directory Tree (ctrl+E to edit the file)") + "\n\n")

	if len(m.TreeNodes) == 0 {
		treeList.WriteString("  \x1b[90m(No workspace directories loaded)\x1b[0m")
	} else {
		for i, node := range m.TreeNodes {
			// Calculate indent spacing based on multi-level depth offsets
			indent := strings.Repeat("  ", node.Depth)

			// Map out branch symbols indicators context cues
			prefix := "📄 "
			if node.IsDir {
				if node.IsExpanded {
					prefix = "📂 "
				} else {
					prefix = "📁 "
				}
			}

			lineText := fmt.Sprintf("%s%s%s", indent, prefix, node.Name)

			if i == m.SelectedFile {
				// Highlight highlighted tree lines active selector indexes rows
				treeList.WriteString(fmt.Sprintf("> \x1b[36;1m%s\x1b[0m\n", lineText))
			} else {
				if node.IsDir {
					treeList.WriteString(fmt.Sprintf("  \x1b[34;1m%s\x1b[0m\n", lineText)) // Folders render blue
				} else {
					treeList.WriteString(fmt.Sprintf("  %s\n", lineText))
				}
			}
		}
	}
	centerBoxStyle := renderer.UnfocusedBorder
	if m.ActiveFocus == model.FocusTree && m.ViewState == model.StateDashboard {
		centerBoxStyle = renderer.FocusedBorder
	}
	centerPanel := centerBoxStyle.Width(((m.WindowWidth / widthDivider) * rightPaneMultiplier) - widthPadding).Height(paneHeight).Render(treeList.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerPanel)
}
