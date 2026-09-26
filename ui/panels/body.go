package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/renderer"
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

func RenderMainBody(m *model.UIState) string {
	rightPaneMultiplier := 8
	leftPaneMultiplier := 2
	widthDivider := 10
	widthPadding := 2
	paneHeight := max(m.WindowHeight-7, 5)

	// 1. Render Left Column (Projects)
	var projList strings.Builder
	projList.WriteString(renderer.Title.Render("📁 Configured Projects") + "\n\n")

	for i, p := range m.Config.Projects {
		sessionLabel := ""
		for id, sess := range m.Sessions {
			if sess.ProjectName == p.Name && sess.IsRunning {
				sessionLabel = sessionBadgeStyle.Render(fmt.Sprintf(" (ID:%d ⏳)", id))
				break
			}
		}

		statusBadge := localBadgeStyle.Render("[local]")
		if !p.Fetched {
			statusBadge = remoteBadgeStyle.Render("[remote]")
		}

		if i == m.SelectedProject {
			projList.WriteString(fmt.Sprintf("> %s [%s] %s%s\n", selectedProjStyle.Render(p.Name), p.Type, statusBadge, sessionLabel))
		} else {
			if p.Fetched {
				projList.WriteString(fmt.Sprintf("  %s [%s] %s%s\n", localProjStyle.Render(p.Name), p.Type, statusBadge, sessionLabel))
			} else {
				projList.WriteString(fmt.Sprintf("  %s [%s] %s%s\n", remoteProjStyle.Render(p.Name), p.Type, statusBadge, sessionLabel))
			}
		}
	}

	leftBoxStyle := renderer.UnfocusedBorder.Background(color.Base)
	if m.ActiveFocus == model.FocusProjects && m.ViewState == model.StateDashboard {
		leftBoxStyle = renderer.FocusedBorder.Background(color.Base)
	}
	leftPanel := leftBoxStyle.
		Width(((m.WindowWidth / widthDivider) * leftPaneMultiplier) - widthPadding).
		Height(paneHeight).
		Render(projList.String())

	// 2. Render Directory Tree in the Center Column
	var treeList strings.Builder
	treeList.WriteString(renderer.Title.Render("🌿 Workspace Directory Tree (Ctrl+E to edit in external editor)") + "\n\n")

	if len(m.TreeNodes) == 0 {
		treeList.WriteString(mutedTextStyle.Render("  (No workspace directories loaded. Use Ctrl+G to clone/checkout projects)"))
	} else {
		for i, node := range m.TreeNodes {
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

			if i == m.SelectedFile {
				treeList.WriteString(fmt.Sprintf("> %s\n", selectedTreeStyle.Render(lineText)))
			} else {
				if node.IsDir {
					treeList.WriteString(fmt.Sprintf("  %s\n", treeFolderStyle.Render(lineText)))
				} else {
					treeList.WriteString(fmt.Sprintf("  %s\n", treeFileStyle.Render(lineText)))
				}
			}
		}
	}

	centerBoxStyle := renderer.UnfocusedBorder.Background(color.Base)
	if m.ActiveFocus == model.FocusTree && m.ViewState == model.StateDashboard {
		centerBoxStyle = renderer.FocusedBorder.Background(color.Base)
	}
	centerPanel := centerBoxStyle.
		Width(((m.WindowWidth / widthDivider) * rightPaneMultiplier) - widthPadding).
		Height(paneHeight).
		Render(treeList.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerPanel)
}
