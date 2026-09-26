package main

import (
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) buildLogLine(msg model.BuildLogLineMsg) (tea.Model, tea.Cmd) {
	if sess, exists := m.state.Sessions[msg.SessionID]; exists {
		if msg.Line != "" {
			line := msg.Line
			if regexp.MustCompile(`(?i)\[error\]|fail`).MatchString(line) {
				line = "\x1b[31;1m" + line + "\x1b[0m"
			} else if regexp.MustCompile(`(?i)\[warn`).MatchString(line) {
				line = "\x1b[33;1m" + line + "\x1b[0m"
			} else if regexp.MustCompile(`(?i)\[info\]|success`).MatchString(line) {
				line = "\x1b[32m" + line + "\x1b[0m"
			}

			sess.Logs = append(sess.Logs, line)
			if m.state.ViewState == model.StateBuildModal && m.state.ActiveSessionID == msg.SessionID {
				m.state.BuildLogs = sess.Logs
			}
		}
	}
	return m, func() tea.Msg {
		activeCh, exists := sessionChannels[msg.SessionID]
		if !exists {
			return nil
		}
		line, ok := <-activeCh
		if !ok {
			return model.BuildCompleteMsg{SessionID: msg.SessionID, Err: nil}
		}
		return model.BuildLogLineMsg{SessionID: msg.SessionID, Line: line}
	}
}
