package componentaction

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
	componentmessage "github.com/exr462/go-dark/ui/component/message"
)

func BuildComplete(ui *model.UI, msg componentmessage.BuildCompleteMsg) tea.Cmd {
	if sess, exists := ui.Sessions[msg.SessionID]; exists {
		sess.IsRunning = false
		sess.Logs = append(sess.Logs, "────────────────────────────────────────────────────────")
		if msg.Err != nil {
			sess.Logs = append(sess.Logs, fmt.Sprintf("❌ PROCESS TERMINATED WITH ERROR: %v", msg.Err))
		} else {
			sess.Logs = append(sess.Logs, "✅ PROCESS LOOP SUCCESSFULLY TERMINATED IN BACKGROUND.")
		}

		if ui.ViewState == model.StateBuildModal && ui.ActiveSessionID == msg.SessionID {
			ui.BuildLogs = sess.Logs
			ui.IsBuilding = false
		}
	}
	delete(sessionChannels, msg.SessionID)
	return nil
}
