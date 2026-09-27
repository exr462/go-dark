package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) buildComplete(msg model.BuildCompleteMsg) (tea.Model, tea.Cmd) {
	if sess, exists := m.ui.Sessions[msg.SessionID]; exists {
		sess.IsRunning = false
		sess.Logs = append(sess.Logs, "────────────────────────────────────────────────────────")
		if msg.Err != nil {
			sess.Logs = append(sess.Logs, fmt.Sprintf("❌ PROCESS TERMINATED WITH ERROR: %v", msg.Err))
		} else {
			sess.Logs = append(sess.Logs, "✅ PROCESS LOOP SUCCESSFULLY TERMINATED IN BACKGROUND.")
		}

		if m.ui.ViewState == model.StateBuildModal && m.ui.ActiveSessionID == msg.SessionID {
			m.ui.BuildLogs = sess.Logs
			m.ui.IsBuilding = false
		}
	}
	delete(sessionChannels, msg.SessionID)
	return m, nil
}
