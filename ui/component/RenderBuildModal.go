package component

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
	buildExecStyle  = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	buildAlertStyle = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	buildHintStyle  = lipgloss.NewStyle().Foreground(color.Overlay0)
	buildKeyHint    = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderBuildModal(ui *model.UI) string {
	var body strings.Builder
	if len(ui.Config.Projects) == 0 || ui.SelectedProject >= len(ui.Config.Projects) {
		return lipgloss.Place(ui.WindowWidth, ui.WindowHeight, lipgloss.Center, lipgloss.Center, decorator.ModalBox.Render("No project selected"))
	}
	targetProj := ui.Config.Projects[ui.SelectedProject]
	buildLogWindowStyle := lipgloss.NewStyle().
		Background(color.Mantle).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color.Surface1)

	var currentSessionID int
	var isRunningBackground bool
	for id, sess := range ui.Sessions {
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
	var selectedBuildOption = "full"
	if ui.SelectedBuildOption < len(ui.BuildOptions) {
		selectedBuildOption = ui.BuildOptions[ui.SelectedBuildOption]
	}
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

	body.WriteString(fmt.Sprintf("🚀 Target Executable Execution: %s\n\n", buildExecStyle.Render(activeCmdString)))

	sessionTitleStr := "Detached State"
	if currentSessionID != 0 {
		sessionTitleStr = fmt.Sprintf("Active Session #%d", currentSessionID)
	}
	body.WriteString(decorator.Title.Render(fmt.Sprintf("🔨 Build Flight Deck: %s [%s]", targetProj.Name, sessionTitleStr)) + "\n")
	body.WriteString(fmt.Sprintf("☕ Env: %s  |  🛠️ Engine: %s\n\n", targetProj.JDKName, targetProj.MavenName))

	if !isRunningBackground {
		body.WriteString(decorator.Section.Render("👉 Select Target Lifecycle Step to Fire:") + "\n\n")
		var optsRow []string
		for i, opt := range ui.BuildOptions {
			if i == ui.SelectedBuildOption {
				optsRow = append(optsRow, decorator.Selected.Render(strings.ToUpper(opt)))
			} else {
				optsRow = append(optsRow, decorator.Inactive.Render(opt))
			}
		}
		body.WriteString("  " + strings.Join(optsRow, "  ") + "\n\n")
		body.WriteString(buildHintStyle.Render(fmt.Sprintf(
			"[%s] Navigate Choices | [%s] Initialize Pipeline | [%s] Dashboard",
			buildKeyHint.Render("←/→"),
			buildKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			buildKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)) + "\n")
	} else {
		body.WriteString(buildAlertStyle.Render(fmt.Sprintf("⏳ Running in background process pool... Assigned Session ID: %d", currentSessionID)) + "\n\n")
	}

	body.WriteString(decorator.Section.Render("📋 Complete Build Output History Trail:") + "\n")

	var activeLogsSource []string
	if currentSessionID != 0 && ui.Sessions[currentSessionID] != nil {
		activeLogsSource = ui.Sessions[currentSessionID].Logs
	} else {
		activeLogsSource = ui.BuildLogs
	}
	logLen := len(activeLogsSource)

	logHeight := max(ui.WindowHeight-18, 5)
	logWidth := max(ui.WindowWidth-8, 20)

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

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
