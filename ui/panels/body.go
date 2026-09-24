package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-build/model"
)

func RenderMainBody(m model.UIState) string {
	focusedBorder := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205"))
	unfocusedBorder := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	columnWidth := (m.TerminalW / 4) - 2
	paneHeight := m.TerminalH - 7

	// 1. Render Left Column (Projects)
	var projList strings.Builder
	projList.WriteString(titleStyle.Render("📁 Configured Projects") + "\n\n")
	for i, p := range m.Config.Projects {
		sessionLabel := ""
		for id, sess := range m.Sessions {
			if sess.ProjectName == p.Name && sess.IsRunning {
				sessionLabel = fmt.Sprintf(" \x1b[33;1m(ID:%d ⏳)\x1b[0m", id)
				break
			}
		}
		if i == m.SelectedProj {
			projList.WriteString(fmt.Sprintf("> \x1b[32m%s\x1b[0m [%s]%s\n", p.Name, p.Type, sessionLabel))
		} else {
			projList.WriteString(fmt.Sprintf("  %s [%s]%s\n", p.Name, p.Type, sessionLabel))
		}
	}
	leftBoxStyle := unfocusedBorder
	if m.ActiveFocus == model.FocusProjects && m.ViewState == model.StateDashboard {
		leftBoxStyle = focusedBorder
	}
	leftPanel := leftBoxStyle.Width(columnWidth).Height(paneHeight).Render(projList.String())

	// 2. FIXED: RENDER THE DYNAMIC TREE HIERARCHY IN THE CENTER COLUMN
	var treeList strings.Builder
	treeList.WriteString(titleStyle.Render("🌿 Workspace Directory Tree") + "\n\n")

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
	centerBoxStyle := unfocusedBorder
	if m.ActiveFocus == model.FocusTree && m.ViewState == model.StateDashboard {
		centerBoxStyle = focusedBorder
	}
	centerPanel := centerBoxStyle.Width(columnWidth).Height(paneHeight).Render(treeList.String())

	// 3. Render Right Column (File Contents View)
	rightBoxStyle := unfocusedBorder
	rightPanel := rightBoxStyle.Width((m.TerminalW / 2) - 2).Height(paneHeight).Render(
		titleStyle.Render("🗒 Live File View Context") + "\n\n" + m.FileViewer.View(),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerPanel, rightPanel)
}
