package main

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
)

// The update loop must keep re-arming the session listener, otherwise only
// the first event of the first session would ever arrive.
func TestUpdate_ReArmsSessionListenerUntilExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no PTY on windows")
	}
	m := &appModel{ui: &model.UI{}, sessions: shell.NewManager()}
	defer m.sessions.Shutdown()
	id, err := m.sessions.Start(shell.StartOptions{Dir: t.TempDir(), Argv: []string{"sh", "-c", "echo wired"}})
	if err != nil {
		t.Fatal(err)
	}

	cmd := m.listenSessions()
	deadline := time.After(5 * time.Second)
	for {
		msgCh := make(chan any, 1)
		go func() { msgCh <- cmd() }()
		var msg any
		select {
		case msg = <-msgCh:
		case <-deadline:
			t.Fatal("never received the exit event")
		}
		_, next := m.Update(msg)
		if next == nil {
			t.Fatalf("Update must re-arm the listener after %T", msg)
		}
		if _, done := msg.(shell.ExitedMsg); done {
			break
		}
		cmd = next
	}
	if !strings.Contains(m.sessions.Get(id).Text(), "wired") {
		t.Fatal("expected the session output")
	}
}

func TestListenSessions_NilManagerIsSafe(t *testing.T) {
	if (&appModel{ui: &model.UI{}}).listenSessions() != nil {
		t.Fatal("no manager means no listener")
	}
}
