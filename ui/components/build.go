package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderBuildModal(m *model.UIState) string {
	var body strings.Builder
	targetProj := m.Config.Projects[m.SelectedProject]
	buildLogWindowStyle := lipgloss.NewStyle().Background(lipgloss.Color("233")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))

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
	var selectedBuildOption = m.BuildOptions[m.SelectedBuildOption]
	switch selectedBuildOption {
	case "without tests":
		selectedBuildOption = "install -DskipTests"
	case "full":
		selectedBuildOption = "install"
	}
	var activeCmdString = fmt.Sprintf("%s %s", mvnBin, selectedBuildOption)
	if targetProj.Type != "java" {
		activeCmdString = fmt.Sprintf("docker build -t %s:latest .", strings.ToLower(targetProj.Name))
	}

	body.WriteString(fmt.Sprintf("🚀 Target Executable Execution: \x1b[35;1m%s\x1b[0m\n\n", activeCmdString))

	// DISPLAY ASSIGNED BOUND SESSION ID IN THE COCKPIT TITLE
	sessionTitleStr := "Detached State"
	if currentSessionID != 0 {
		sessionTitleStr = fmt.Sprintf("Active Session #%d", currentSessionID)
	}
	body.WriteString(TitleStyle.Render(fmt.Sprintf("🔨 Build Flight Deck: %s [%s]", targetProj.Name, sessionTitleStr)) + "\n")
	body.WriteString(fmt.Sprintf("☕ Env: %s  |  🛠️ Engine: %s\n\n", targetProj.JDKName, targetProj.MavenName))

	// 1. OPTIONS SELECTION MENU BLOCK (Evaluate true background activity instead of transient state variables)
	if !isRunningBackground {
		body.WriteString("👉 Select Target Lifecycle Step to Fire:\n\n")
		var optsRow []string
		for i, opt := range m.BuildOptions {
			if i == m.SelectedBuildOption {
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

	logHeight := max(m.WindowHeight-18, 5)
	logWidth := max(m.WindowWidth-8, 20)

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
	consoleBox := buildLogWindowStyle.
		Width(logWidth).
		Height(logHeight).
		MaxWidth(logWidth).
		MaxHeight(logHeight).
		Render(strings.Join(historyLines, "\n"))
	body.WriteString(consoleBox)
	return lipgloss.Place(m.WindowWidth, m.WindowHeight, lipgloss.Center, lipgloss.Center, ModalBox.Width(m.WindowWidth-4).Render(body.String()))
}
