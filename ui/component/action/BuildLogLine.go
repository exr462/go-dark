package componentaction

import (
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
	componentmessage "github.com/exr462/go-dark/ui/component/message"
)

var sessionChannels = make(map[int]chan string)

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
	return func() tea.Msg {
		activeCh, exists := sessionChannels[msg.SessionID]
		if !exists {
			return nil
		}
		line, ok := <-activeCh
		if !ok {
			return componentmessage.BuildCompleteMsg{SessionID: msg.SessionID, Err: nil}
		}
		return componentmessage.BuildLogLineMsg{SessionID: msg.SessionID, Line: line}
	}
}
