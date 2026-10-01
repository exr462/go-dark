package session

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"runtime"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/terminal"
)

var ActiveLogChannel chan tea.Msg

func SpawnTerminalSession(workingDir string, rawCommand string) tea.Cmd {
	return func() tea.Msg {
		ctx, _ := context.WithCancel(context.Background())
		var shellName string
		var shellArgs []string
		if runtime.GOOS == "windows" {
			shellName = "cmd.exe"
			shellArgs = []string{"/C", rawCommand}
		} else {
			shellName = "sh"
			shellArgs = []string{"-c", rawCommand} // 🛠️ Fix: Must be lowercase -c for unix sh
		}

		cmd := exec.CommandContext(ctx, shellName, shellArgs...)
		cmd.Dir = workingDir
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return LogProcessFinishedMsg{Err: err}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return LogProcessFinishedMsg{Err: err}
		}

		if err := cmd.Start(); err != nil {
			return LogProcessFinishedMsg{Err: err}
		}

		// Initialize or reset the thread-safe channel dispatcher
		ActiveLogChannel = make(chan tea.Msg, 500)

		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine 1: Read stdout lines safely
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				ActiveLogChannel <- LogStreamMsg{
					Text:  terminal.ScrubAnsiNoise(scanner.Text()),
					IsErr: false,
				}
			}
		}()

		// Goroutine 2: Read stderr lines safely
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				log.Printf("scanner: %s", terminal.ScrubAnsiNoise(scanner.Text()))
				ActiveLogChannel <- LogStreamMsg{
					Text:  terminal.ScrubAnsiNoise(scanner.Text()),
					IsErr: true,
				}
			}
		}()

		// Goroutine 3: Wait for output to complete, harvest status, and close the stream channel safely
		go func() {
			wg.Wait()
			waitErr := cmd.Wait()
			ActiveLogChannel <- LogProcessFinishedMsg{Err: waitErr}
			close(ActiveLogChannel)
		}()

		// Return a success indicator message. This tells the UI to immediately
		// chain the long-running log consumer loop!
		return LogStreamMsg{IsErr: false}
	}
}

// ListenForLogs keeps the Bubble Tea loop alive and waiting for incoming data lines
func ListenForLogs() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ActiveLogChannel
		if !ok {
			return nil // Stream channel cleanly exhausted and closed
		}
		return msg
	}
}

type LogStreamMsg struct {
	Text  string
	IsErr bool
}

// LogProcessFinishedMsg notifies the UI that execution has completed.
type LogProcessFinishedMsg struct {
	Err error
}
