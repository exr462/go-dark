package main

import (
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
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

func TestUpdate_CtrlTOpensTerminalAndPasteReachesShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no PTY on windows")
	}
	t.Setenv("SHELL", "/bin/sh")
	ui := &model.UI{
		Config:       config.Config{ShortCuts: action.DefaultShortcuts, BasePath: t.TempDir()},
		ViewState:    model.StateDashboard,
		WindowWidth:  100,
		WindowHeight: 30,
	}
	m := &appModel{ui: ui, sessions: shell.NewManager()}
	defer m.sessions.Shutdown()
	go func() {
		for {
			m.sessions.Listen()()
		}
	}()

	m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if ui.ViewState != model.StateTerminalCockpit || len(ui.TerminalTabs) != 1 {
		t.Fatalf("ctrl+t must open a terminal tab, state=%v tabs=%v", ui.ViewState, ui.TerminalTabs)
	}
	m.Update(tea.PasteMsg{Content: "echo via-update\n"})
	s := m.sessions.Get(ui.ActiveTerminalSession())
	deadline := time.Now().Add(5 * time.Second)
	for strings.Count(s.Text(), "via-update") < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if strings.Count(s.Text(), "via-update") < 2 {
		t.Fatalf("paste must reach the shell, screen:\n%s", s.Text())
	}

	view := m.View()
	if view.Cursor == nil {
		t.Fatal("the terminal view must place the real cursor")
	}

	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if w, _ := s.Size(); w != 120-2*model.TerminalPaneLeft {
		t.Fatalf("window resize must resize the shell, width %d", w)
	}
}
