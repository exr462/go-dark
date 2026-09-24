package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderBuildModal(m model.UIState) string {
	var body strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("46")).Padding(0, 1).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("237")).Padding(0, 1)
	logWindowStyle := lipgloss.NewStyle().Background(lipgloss.Color("233")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))

	targetProj := m.Config.Projects[m.SelectedProj]

	// !!! NEW: DETERMINE IF THIS SPECIFIC PROJECT HAS A RUNNING RECOGNIZED SESSION ID !!!
	var currentSessionID int
	var isRunningBackground bool
	for id, sess := range m.Sessions {
		if sess.ProjectName == targetProj.Name && sess.IsRunning {
			currentSessionID = id
			isRunningBackground = true
			break
		}
	}

	var mvnBin = "mvn"
	if targetProj.Type != "java" {
		mvnBin = "docker"
	}
	var activeCmdString = fmt.Sprintf("%s %s", mvnBin, m.BuildOptions[m.SelectedBuildOpt])
	if targetProj.Type != "java" {
		activeCmdString = fmt.Sprintf("docker build -t %s:latest .", strings.ToLower(targetProj.Name))
	}
	body.WriteString(fmt.Sprintf("🚀 Target Executable Execution: \x1b[35;1m%s\x1b[0m\n\n", activeCmdString))

	// DISPLAY ASSIGNED BOUND SESSION ID IN THE COCKPIT TITLE
	sessionTitleStr := "Detached State"
	if currentSessionID != 0 {
		sessionTitleStr = fmt.Sprintf("Active Session #%d", currentSessionID)
	}
	body.WriteString(titleStyle.Render(fmt.Sprintf("🔨 Build Flight Deck: %s [%s]", targetProj.Name, sessionTitleStr)) + "\n")
	body.WriteString(fmt.Sprintf("☕ Env: %s  |  🛠️ Engine: %s\n\n", targetProj.JDKName, targetProj.MavenName))

	// 1. OPTIONS SELECTION MENU BLOCK (Evaluate true background activity instead of transient state variables)
	if !isRunningBackground {
		body.WriteString("👉 Select Target Lifecycle Step to Fire:\n\n")
		var optsRow []string
		for i, opt := range m.BuildOptions {
			if i == m.SelectedBuildOpt {
				optsRow = append(optsRow, selectedStyle.Render(strings.ToUpper(opt)))
			} else {
				optsRow = append(optsRow, inactiveStyle.Render(opt))
			}
		}
		body.WriteString("  " + strings.Join(optsRow, "  ") + "\n\n")
		body.WriteString("\x1b[90m [←/→] Navigate Choices  |  [Enter] Initialize Pipeline  |  [Esc] Dashboard\x1b[0m\n")
	} else {
		body.WriteString(fmt.Sprintf("⏳ \x1b[33;1mRunning in background process pool... Assigned Session ID: %d\x1b[0m\n\n", currentSessionID))
	}

	// 2. COMPILATION HISTORY VIEWPORT BOX
	body.WriteString("📋 Complete Build Output History Trail:\n")

	// If we are currently running or have matching session data, fetch from the map registry
	var activeLogsSource []string
	if currentSessionID != 0 && m.Sessions[currentSessionID] != nil {
		activeLogsSource = m.Sessions[currentSessionID].Logs
	} else {
		activeLogsSource = m.BuildLogs
	}
	logLen := len(activeLogsSource)

	logHeight := max(m.TerminalH-18, 5)
	logWidth := max(m.TerminalW-8, 20)

	var historyLines []string
	startIdx := 0
	if logLen > logHeight {
		startIdx = logLen - logHeight
	}

	for i := startIdx; i < logLen; i++ {
		line := activeLogsSource[i]
		if len(line) > logWidth {
			line = line[:logWidth-3] + "..."
		}
		historyLines = append(historyLines, line)
	}

	for len(historyLines) < logHeight {
		historyLines = append(historyLines, strings.Repeat(" ", logWidth))
	}

	consoleBox := logWindowStyle.
		Width(logWidth).
		Height(logHeight).
		MaxWidth(logWidth).
		MaxHeight(logHeight).
		Render(strings.Join(historyLines, "\n"))

	body.WriteString(consoleBox)

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ModalBorderColor).
		Background(ModalBackground).
		Padding(1, 2, 1, 2).
		Width(m.TerminalW - 4).
		Render(body.String())

	return lipgloss.Place(m.TerminalW, m.TerminalH, lipgloss.Center, lipgloss.Center, modalBox)
}
