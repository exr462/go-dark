package session

import (
	"bufio"
	"context"
	"os/exec"
	"runtime"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/terminal"
)

// SpawnTerminalSession provisions a background shell environment and returns a tracking command.
func SpawnTerminalSession(ui *model.UI, workingDir string, rawCommand string) tea.Cmd {
	return func() tea.Msg {
		terminal.ProcMutex.Lock()
		id := terminal.NextID
		terminal.NextID++
		ui.ActiveTerminalSessionID = id

		// Update the session state inside our global tracker
		ui.TerminalSessions[id] = terminal.SessionState{
			ProjectName: "Terminal Console",
			IsRunning:   true,
		}
		terminal.ProcMutex.Unlock()

		// 1. Establish context with absolute cancellation capabilities
		ctx, cancel := context.WithCancel(context.Background())

		terminal.ProcMutex.Lock()
		terminal.ProcPool[id] = cancel
		terminal.ProcMutex.Unlock()

		// 2. Select host platform wrapper shell flags
		var shellName string
		var shellArgs []string
		if runtime.GOOS == "windows" {
			shellName = "cmd.exe"
			shellArgs = []string{"/C", rawCommand}
		} else {
			shellName = "sh"
			shellArgs = []string{"/C", rawCommand}
		}
		// 3. Configure process execution matrix
		cmd := exec.CommandContext(ctx, shellName, shellArgs...)
		cmd.Dir = workingDir // Anchor command straight to configured working path

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return terminal.TerminalLogMsg{SessionID: id, Done: true, Error: err}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return terminal.TerminalLogMsg{SessionID: id, Done: true, Error: err}
		}

		if err := cmd.Start(); err != nil {
			return terminal.TerminalLogMsg{SessionID: id, Done: true, Error: err}
		}

		// 4. Concurrent pipe aggregation channels
		var wg sync.WaitGroup
		wg.Add(2)

		// Read stdout line-by-line using your internal channel architecture setup
		// Note: Because Bubble Tea commands execute instantly once, we spawn background loops
		// that directly populate your ui log matrix or pass sequential messages.
		// For proper streaming, we proxy logs via routine channels.
		logChan := make(chan terminal.LogLine, 100)

		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				logChan <- terminal.LogLine{Text: terminal.ScrubAnsiNoise(scanner.Text()), IsErr: false}
			}
		}()

		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				logChan <- terminal.LogLine{Text: terminal.ScrubAnsiNoise(scanner.Text()), IsErr: true}
			}
		}()

		// Track process completion state cleanly
		go func() {
			wg.Wait()
			close(logChan)
		}()

		// Drain loop strategy matching Go-Dark's map session registry model
		// This thread processes records smoothly without blocking layout tasks
		go func() {
			for logLine := range logChan {
				// We append directly to UI state or pass message updates
				// depending on your main update dispatcher layout loop
				ui.TerminalLogs = append(ui.TerminalLogs, logLine)
			}

			_ = cmd.Wait()

			terminal.ProcMutex.Lock()
			if sess, ok := ui.Sessions[id]; ok {
				sess.IsRunning = false
				ui.Sessions[id] = sess
			}
			delete(terminal.ProcPool, id)
			terminal.ProcMutex.Unlock()

			ui.IsBuilding = false
		}()

		return terminal.TerminalLogMsg{SessionID: id, Text: "🚀 Process started successfully.", IsErr: false}
	}
}
