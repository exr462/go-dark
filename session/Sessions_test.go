package session

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	componentmessage "github.com/exr462/go-dark/ui/component/message"
)

// drain runs the session's listen commands until completion, like the
// Bubble Tea loop would, and returns every streamed line.
func drain(t *testing.T, cmd tea.Cmd) []string {
	t.Helper()
	var lines []string
	for range 1000 {
		switch msg := cmd().(type) {
		case componentmessage.BuildLogLineMsg:
			lines = append(lines, msg.Line)
			cmd = ListenSession(msg.SessionID)
		case componentmessage.BuildCompleteMsg:
			return lines
		default:
			t.Fatalf("unexpected message %#v", msg)
		}
	}
	t.Fatal("session never completed")
	return nil
}

func TestSpawnBackgroundSession_StreamsAllOutputOfResolvedCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	ui := &model.UI{Sessions: map[int]*model.BuildSession{}}
	proj := config.Project{Name: "svc", Path: dir, Type: "go", Commands: map[string]string{"build": "echo one; echo two; pwd"}}

	lines := drain(t, SpawnBackgroundSession(ui, proj, "build"))

	joined := strings.Join(lines, "\n")
	for _, want := range []string{"$ echo one; echo two; pwd", "one", "two", filepath.Base(dir)} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in output:\n%s", want, joined)
		}
	}
	if ui.Sessions[1].Command != "echo one; echo two; pwd" {
		t.Errorf("session must record the resolved command, got %q", ui.Sessions[1].Command)
	}
}

func TestSpawnBackgroundSession_UsesAssignedMavenProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	mvnHome := t.TempDir()
	_ = os.MkdirAll(filepath.Join(mvnHome, "bin"), 0o755)
	_ = os.WriteFile(filepath.Join(mvnHome, "bin", "mvn"), []byte("#!/bin/sh\necho \"profile-mvn $*\"\n"), 0o755)
	ui := &model.UI{
		Sessions: map[int]*model.BuildSession{},
		Config:   config.Config{Mavens: []config.Profile{{Name: "mvn-x", Path: mvnHome}}},
	}
	proj := config.Project{Name: "api", Path: t.TempDir(), Type: "maven", MavenName: "mvn-x"}

	lines := drain(t, SpawnBackgroundSession(ui, proj, "test"))

	if !strings.Contains(strings.Join(lines, "\n"), "profile-mvn clean test") {
		t.Fatalf("expected the profile's mvn to run 'clean test' once, got %v", lines)
	}
}

func TestSpawnBackgroundSession_UnknownCommandReportsError(t *testing.T) {
	ui := &model.UI{Sessions: map[int]*model.BuildSession{}}
	lines := drain(t, SpawnBackgroundSession(ui, config.Project{Name: "x", Type: "go"}, "deploy"))
	if len(lines) != 1 || !strings.Contains(lines[0], "No \"deploy\" command") {
		t.Fatalf("expected a clear error line, got %v", lines)
	}
}
