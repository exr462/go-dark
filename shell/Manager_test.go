//go:build !windows

package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// pump drives the manager's event loop like the Bubble Tea runtime would,
// until cond holds or the timeout expires.
func pump(t *testing.T, m *Manager, timeout time.Duration, cond func(tea.Msg) bool) {
	t.Helper()
	deadline := time.After(timeout)
	msgs := make(chan tea.Msg)
	go func() {
		for {
			msgs <- m.Listen()()
		}
	}()
	for {
		select {
		case msg := <-msgs:
			if cond(msg) {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for session events")
		}
	}
}

// drain consumes session events in the background, like a UI that renders
// on every OutputMsg.
func drain(m *Manager) {
	go func() {
		for {
			m.Listen()()
		}
	}()
}

func waitForText(t *testing.T, m *Manager, s *Session, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(s.Text(), want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("screen never showed %q; got:\n%s", want, s.Text())
}

func TestStart_RunsCommandInDirAndReportsExit(t *testing.T) {
	m := NewManager()
	defer m.Shutdown()
	dir := t.TempDir()

	id, err := m.Start(StartOptions{ProjectName: "svc", Dir: dir, Argv: []string{"sh", "-c", "echo hi; pwd; exit 3"}})
	if err != nil {
		t.Fatal(err)
	}

	var exit ExitedMsg
	pump(t, m, 5*time.Second, func(msg tea.Msg) bool {
		e, ok := msg.(ExitedMsg)
		exit = e
		return ok
	})

	s := m.Get(id)
	if exit.ID != id || exit.Err == nil || s.Running() {
		t.Fatalf("expected a non-zero exit for session %d, got %+v running=%v", id, exit, s.Running())
	}
	text := s.Text()
	real, _ := filepath.EvalSymlinks(dir)
	if !strings.Contains(text, "hi") || !strings.Contains(text, filepath.Base(real)) {
		t.Fatalf("expected output and working dir on screen, got:\n%s", text)
	}
	if s.Title != "svc" {
		t.Errorf("title should default to the project name, got %q", s.Title)
	}
}

func TestInteractiveShell_StatePersistsAcrossCommands(t *testing.T) {
	m := NewManager()
	defer m.Shutdown()
	sub := filepath.Join(t.TempDir(), "sub")
	_ = os.Mkdir(sub, 0o755)

	id, err := m.Start(StartOptions{Dir: filepath.Dir(sub), Argv: []string{"sh"}, Interactive: true})
	if err != nil {
		t.Fatal(err)
	}
	s := m.Get(id)
	drain(m)

	_ = m.Paste(id, "cd sub\n")
	_ = m.Paste(id, "export GODARK_X=42\n")
	_ = m.Paste(id, "echo \"$(basename \"$PWD\")-$GODARK_X\"\n")
	waitForText(t, m, s, "sub-42")
}

func TestSendKey_TypesAndCtrlCInterrupts(t *testing.T) {
	m := NewManager()
	defer m.Shutdown()
	id, err := m.Start(StartOptions{Dir: t.TempDir(), Argv: []string{"sh"}, Interactive: true})
	if err != nil {
		t.Fatal(err)
	}
	s := m.Get(id)
	drain(m)

	typeLine := func(line string) {
		for _, r := range line {
			_ = m.SendKey(id, tea.KeyPressMsg{Code: r, Text: string(r)})
		}
		_ = m.SendKey(id, tea.KeyPressMsg{Code: tea.KeyEnter})
	}
	typeLine("sleep 30")
	time.Sleep(300 * time.Millisecond)
	_ = m.SendKey(id, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	typeLine("echo after-$((6*7))")

	// Commands run in order, so seeing after-42 within the timeout proves
	// ctrl+c interrupted the 30s sleep.
	waitForText(t, m, s, "after-42")
}

func TestResize_UpdatesEmulatorAndPTY(t *testing.T) {
	m := NewManager()
	defer m.Shutdown()
	id, err := m.Start(StartOptions{Dir: t.TempDir(), Argv: []string{"sh"}, Interactive: true, Width: 80, Height: 24})
	if err != nil {
		t.Fatal(err)
	}
	s := m.Get(id)
	drain(m)

	if err := m.Resize(id, 100, 30); err != nil {
		t.Fatal(err)
	}
	if w, h := s.Size(); w != 100 || h != 30 {
		t.Fatalf("emulator size %dx%d, want 100x30", w, h)
	}
	_ = m.Paste(id, "stty size\n")
	waitForText(t, m, s, "30 100")
}

func TestClose_KillsProcessAndForgetsSession(t *testing.T) {
	m := NewManager()
	id, err := m.Start(StartOptions{Dir: t.TempDir(), Argv: []string{"sleep", "60"}})
	if err != nil {
		t.Fatal(err)
	}
	s := m.Get(id)

	start := time.Now()
	if err := m.Close(id); err != nil && !strings.Contains(err.Error(), "closed") {
		t.Fatal(err)
	}
	if time.Since(start) > 4*time.Second {
		t.Fatal("close took too long")
	}
	select {
	case <-s.Done():
	default:
		t.Fatal("the process must be dead after Close")
	}
	if m.Get(id) != nil || len(m.List()) != 0 {
		t.Fatal("closed sessions must be forgotten")
	}
}

func TestOutputMsgsAreCoalesced(t *testing.T) {
	m := NewManager()
	defer m.Shutdown()
	id, err := m.Start(StartOptions{Dir: t.TempDir(), Argv: []string{"sh", "-c", "i=0; while [ $i -lt 2000 ]; do echo line $i; i=$((i+1)); done"}})
	if err != nil {
		t.Fatal(err)
	}

	outputs := 0
	pump(t, m, 10*time.Second, func(msg tea.Msg) bool {
		if _, ok := msg.(OutputMsg); ok {
			outputs++
			time.Sleep(5 * time.Millisecond) // a slow UI
		}
		_, done := msg.(ExitedMsg)
		return done
	})

	if outputs == 0 || outputs > 200 {
		t.Fatalf("expected a handful of coalesced output messages for 2000 lines, got %d", outputs)
	}
	if !strings.Contains(m.Get(id).Text(), "line 1999") {
		t.Fatal("the final output must be on screen when the exit is reported")
	}
}

func TestMultipleSessionsAreIndependent(t *testing.T) {
	m := NewManager()
	defer m.Shutdown()
	a, _ := m.Start(StartOptions{ProjectName: "a", Dir: t.TempDir(), Argv: []string{"sh"}, Interactive: true})
	b, _ := m.Start(StartOptions{ProjectName: "b", Dir: t.TempDir(), Argv: []string{"sh"}, Interactive: true})
	drain(m)

	_ = m.Paste(a, "echo from-a\n")
	_ = m.Paste(b, "echo from-b\n")
	waitForText(t, m, m.Get(a), "from-a")
	waitForText(t, m, m.Get(b), "from-b")
	if strings.Contains(m.Get(a).Text(), "from-b") {
		t.Fatal("sessions must not share screens")
	}
	if list := m.List(); len(list) != 2 || list[0].ID != a || list[1].ID != b {
		t.Fatalf("List must return sessions in start order, got %v", list)
	}
}

func TestUnknownSessionErrors(t *testing.T) {
	m := NewManager()
	if m.SendKey(9, tea.KeyPressMsg{}) == nil || m.Write(9, nil) == nil || m.Resize(9, 1, 1) == nil || m.Close(9) == nil {
		t.Fatal("operations on unknown sessions must fail")
	}
}

func TestWithTermEnvOverridesTerm(t *testing.T) {
	env := withTermEnv([]string{"TERM=dumb", "HOME=/h"})
	joined := strings.Join(env, " ")
	if strings.Contains(joined, "TERM=dumb") || !strings.Contains(joined, "TERM=xterm-256color") || !strings.Contains(joined, "HOME=/h") {
		t.Fatalf("unexpected env %v", env)
	}
}
