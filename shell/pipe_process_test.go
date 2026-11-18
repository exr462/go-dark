//go:build !windows

package shell

import (
	"strings"
	"testing"
	"time"
)

// The pipe fallback is what Windows uses for builds; exercise it on Unix by
// starting it directly behind a session.
func TestPipeProcess_StreamsLinesFromColumnZero(t *testing.T) {
	proc, err := startPipeProcess([]string{"sh", "-c", "echo one; echo two >&2; printf 'a\\r\\nb\\n'"}, t.TempDir(), withTermEnv(nil))
	if err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	s := m.adopt(proc, StartOptions{Title: "pipe", Width: 40, Height: 10})
	drain(m)

	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("pipe process never exited")
	}
	if got := s.Text(); got != "one\ntwo\na\nb" {
		t.Fatalf("expected each line at column 0, got %q", got)
	}
	if s.ExitCode() != 0 {
		t.Fatalf("exit code %d", s.ExitCode())
	}
}

func TestToCRLF(t *testing.T) {
	cases := map[string]string{"a\nb": "a\r\nb", "a\r\nb": "a\r\nb", "\n": "\r\n", "plain": "plain"}
	for in, want := range cases {
		if got := string(toCRLF([]byte(in))); got != want {
			t.Errorf("toCRLF(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExitCode(t *testing.T) {
	m := NewManager()
	defer m.Shutdown()
	id, _ := m.Start(StartOptions{Dir: t.TempDir(), Argv: []string{"sh", "-c", "exit 7"}})
	drain(m)
	s := m.Get(id)
	<-s.Done()
	if s.ExitCode() != 7 || !strings.Contains(s.ExitErr().Error(), "7") {
		t.Fatalf("expected exit code 7, got %d (%v)", s.ExitCode(), s.ExitErr())
	}
}
