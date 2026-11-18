package componentaction

import (
	"regexp"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
	componentmessage "github.com/exr462/go-dark/ui/component/message"
)

func BuildLogLine(ui *model.UI, msg componentmessage.BuildLogLineMsg) tea.Cmd {
	if sess, exists := ui.Sessions[msg.SessionID]; exists {
		if msg.Line != "" {
			line := msg.Line
			if regexp.MustCompile(`(?i)\[error|fail`).MatchString(line) {
				line = "\x1b[31;1m" + line + "\x1b[0m"
			} else if regexp.MustCompile(`(?i)\[warn`).MatchString(line) {
				line = "\x1b[33;1m" + line + "\x1b[0m"
			} else if regexp.MustCompile(`(?i)\[info|success`).MatchString(line) {
				line = "\x1b[32m" + line + "\x1b[0m"
			}

			sess.Logs = append(sess.Logs, line)
			if ui.ViewState == model.StateBuildModal && ui.ActiveSessionID == msg.SessionID {
				ui.BuildLogs = sess.Logs
			}
		}
	}
	// Keep reading the session's output: the channel lives in the session
	// package (a second, always-empty map here used to stop every build
	// after its first line).
	return session.ListenSession(msg.SessionID)
}
