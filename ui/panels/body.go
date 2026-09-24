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
		// !!! NEW: EXTRACT ACTIVE TRACKING BACKGROUND SESSION IDS FOR THIS SPECIFIC PROJECT !!!
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

	// 2. Render Center Column (File List Tree)
	var treeList strings.Builder
	treeList.WriteString(titleStyle.Render("🌿 File Workspace Tree") + "\n\n")
	if len(m.Files) == 0 {
		treeList.WriteString("  (No files loaded)")
	} else {
		for i, f := range m.Files {
			if i == m.SelectedFile {
				treeList.WriteString(fmt.Sprintf("> \x1b[36m%s\x1b[0m\n", f))
			} else {
				treeList.WriteString(fmt.Sprintf("  %s\n", f))
			}
		}
	}
	centerBoxStyle := unfocusedBorder
	if m.ActiveFocus == model.FocusTree && m.ViewState == model.StateDashboard {
		centerBoxStyle = focusedBorder
	}
	centerPanel := centerBoxStyle.Width(columnWidth).Height(paneHeight).Render(treeList.String())

	// 3. Render Right Column (File Contents)
	rightBoxStyle := unfocusedBorder
	rightPanel := rightBoxStyle.Width((m.TerminalW / 2) - 2).Height(paneHeight).Render(
		titleStyle.Render("🗒 Live File View Context") + "\n\n" + m.FileViewer.View(),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerPanel, rightPanel)
}
