package session

import (
	"bufio"
	"context"
	"fmt"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/runner"
	componentmessage "github.com/exr462/go-dark/ui/component/message"
)

// sessionChannels is read from Bubble Tea command goroutines and written by
// the update loop, hence the lock.
var (
	sessionChannelsMu sync.Mutex
	sessionChannels   = make(map[int]chan string)
)

// SpawnBackgroundSession runs the project's command for commandKey (see
// config.CommandKeys) in the background, streaming its output into a session.
func SpawnBackgroundSession(ui *model.UI, proj config.Project, commandKey string) tea.Cmd {
	ui.NextSessionID++
	sID := ui.NextSessionID

	cmdStr := config.ResolveCommand(proj, commandKey)
	session := &model.BuildSession{
		ID:          sID,
		ProjectName: proj.Name,
		Command:     cmdStr,
		IsRunning:   true,
		Logs:        []string{fmt.Sprintf("🚀 [Session %d] Initializing background execution...", sID)},
	}
	ui.Sessions[sID] = session
	ui.ActiveSessionID = sID

	localCh := make(chan string, 500)
	cfg := ui.Config

	go func() {
		if cmdStr == "" {
			localCh <- fmt.Sprintf("❌ No %q command configured for %s", commandKey, proj.Name)
			close(localCh)
			return
		}
		cmd := runner.Command(context.Background(), cfg, proj, cmdStr)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			localCh <- fmt.Sprintf("❌ Setup Pipe Exception: %v", err)
			close(localCh)
			return
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			localCh <- fmt.Sprintf("❌ Start Process Exception: %v", err)
			close(localCh)
			return
		}

		localCh <- "$ " + cmdStr
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			localCh <- scanner.Text()
		}

		waitErr := cmd.Wait()
		if waitErr != nil {
			localCh <- fmt.Sprintf("❌ Exec Terminated with fault code: %v", waitErr)
		}
		close(localCh)
	}()

	return listenToSessionChannel(sID, localCh)
}

func listenToSessionChannel(sID int, ch chan string) tea.Cmd {
	if ch != nil {
		sessionChannelsMu.Lock()
		sessionChannels[sID] = ch
		sessionChannelsMu.Unlock()
	}
	return ListenSession(sID)
}

// ListenSession waits for the next output line of a background session, or
// its completion once the output channel is closed.
func ListenSession(sID int) tea.Cmd {
	return func() tea.Msg {
		sessionChannelsMu.Lock()
		activeCh, exists := sessionChannels[sID]
		sessionChannelsMu.Unlock()
		if !exists {
			return componentmessage.BuildCompleteMsg{SessionID: sID, Err: fmt.Errorf("channel untracked")}
		}

		line, ok := <-activeCh
		if !ok {
			sessionChannelsMu.Lock()
			delete(sessionChannels, sID)
			sessionChannelsMu.Unlock()
			return componentmessage.BuildCompleteMsg{SessionID: sID, Err: nil}
		}
		return componentmessage.BuildLogLineMsg{SessionID: sID, Line: line}
	}
}
