package session

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/terminal"
)

// AbortTerminalSession triggers an immediate context teardown across background execution subroutines.
func AbortTerminalSession(ui *model.UI, sessionID int) tea.Cmd {
	return func() tea.Msg {
		terminal.ProcMutex.Lock()
		cancel, exists := terminal.ProcPool[sessionID]
		if exists && cancel != nil {
			cancel() // Triggers cascading context cancellation down the OS process thread tree
			delete(terminal.ProcPool, sessionID)
		}

		if sess, ok := ui.Sessions[sessionID]; ok {
			sess.IsRunning = false
			ui.Sessions[sessionID] = sess
		}
		terminal.ProcMutex.Unlock()

		ui.IsBuilding = false
		ui.TerminalLogs = append(ui.TerminalLogs, terminal.LogLine{
			Text:  "🛑 Process manually aborted by terminal administrator.",
			IsErr: true,
		})

		return terminal.TerminalLogMsg{SessionID: sessionID, Done: true}
	}
}
