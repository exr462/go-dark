package component

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
	"github.com/exr462/go-dark/shell"
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

	latest := session.LatestBuild(ui.Shells, targetProj.ID, session.TagBuild)
	running := latest != nil && latest.Running()

	activeCmdString := "(no commands configured for this project type)"
	if ui.SelectedBuildOption < len(ui.BuildOptions) {
		activeCmdString = config.ResolveCommand(targetProj, ui.BuildOptions[ui.SelectedBuildOption])
	}
	if running {
		activeCmdString = latest.Title
	}
	body.WriteString(fmt.Sprintf("🚀 Target Executable Execution: %s\n\n", buildExecStyle.Render(activeCmdString)))

	sessionTitleStr := "Detached State"
	if latest != nil {
		sessionTitleStr = fmt.Sprintf("Session #%d", latest.ID)
	}
	body.WriteString(decorator.Title.Render(fmt.Sprintf("🔨 Build Flight Deck: %s [%s]", targetProj.Name, sessionTitleStr)) + "\n")
	body.WriteString(fmt.Sprintf("☕ Env: %s  |  🛠️ Engine: %s\n\n", targetProj.JDKName, targetProj.MavenName))

	if !running {
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
	}
	body.WriteString(buildStatusLine(latest) + "\n")
	body.WriteString(buildHintStyle.Render(buildHint(ui, latest, running)) + "\n\n")

	body.WriteString(decorator.Section.Render("📋 Build Output:") + "\n")

	logHeight := max(ui.WindowHeight-20, 5)
	logWidth := max(ui.WindowWidth-8, 20)
	var historyLines []string
	if latest != nil {
		historyLines = tailLines(latest.Text(), logHeight)
	} else {
		historyLines = []string{"Console engine ready. Select command step to initialize stream..."}
	}
	for i, line := range historyLines {
		historyLines[i] = colorizeLogLine(ansi.Truncate(line, logWidth, "…"))
	}
	for len(historyLines) < logHeight {
		historyLines = append(historyLines, "")
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
		decorator.ModalBox.Width(ui.WindowWidth-4+decorator.BorderFrame).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}

func buildStatusLine(latest *shell.Session) string {
	switch {
	case latest == nil:
		return ""
	case latest.Running():
		return buildAlertStyle.Render(fmt.Sprintf("⏳ Running in the background as session #%d - it keeps going if you close this screen", latest.ID))
	case latest.ExitCode() == 0:
		return decorator.Success.Render("✅ Last build succeeded")
	default:
		return decorator.Error.Render(fmt.Sprintf("❌ Last build failed (exit code %d)", latest.ExitCode()))
	}
}

func buildHint(ui *model.UI, latest *shell.Session, running bool) string {
	key := func(a action.Action) string {
		return buildKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, a))
	}
	var parts []string
	if !running {
		parts = append(parts, fmt.Sprintf("[%s] Choose", buildKeyHint.Render("←/→")), fmt.Sprintf("[%s] Run", key(action.Save)))
	}
	if latest != nil {
		parts = append(parts, fmt.Sprintf("[%s] Open in terminal (ctrl+c there cancels)", key(action.OpenTerminal)))
	}
	parts = append(parts, fmt.Sprintf("[%s] Dashboard", key(action.Escape)))
	return strings.Join(parts, " | ")
}

// tailLines returns the last n lines of text.
func tailLines(text string, n int) []string {
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

var (
	logErrorPattern = regexp.MustCompile(`(?i)\[error|fail`)
	logWarnPattern  = regexp.MustCompile(`(?i)\[warn`)
	logInfoPattern  = regexp.MustCompile(`(?i)\[info|success`)
)

// colorizeLogLine highlights Maven-style [ERROR]/[WARN]/[INFO] lines.
func colorizeLogLine(line string) string {
	switch {
	case logErrorPattern.MatchString(line):
		return decorator.Error.Render(line)
	case logWarnPattern.MatchString(line):
		return decorator.Warning.Render(line)
	case logInfoPattern.MatchString(line):
		return decorator.Success.Render(line)
	}
	return line
}
