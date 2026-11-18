//go:build !windows

package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/shell"
)

func newManager(t *testing.T) *shell.Manager {
	t.Helper()
	m := shell.NewManager()
	t.Cleanup(m.Shutdown)
	go func() {
		for {
			m.Listen()()
		}
	}()
	return m
}

func waitDone(t *testing.T, s *shell.Session) {
	t.Helper()
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session never finished")
	}
}

func TestStartBuild_RunsResolvedCommandInProjectDir(t *testing.T) {
	m := newManager(t)
	dir := t.TempDir()
	proj := config.Project{ID: "local/svc", Name: "svc", Path: dir, Type: "go", Commands: map[string]string{"build": "echo one; pwd"}}

	id, err := StartBuild(m, config.Config{}, proj, "build", TagBuild, 80, 24, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := m.Get(id)
	waitDone(t, s)

	real, _ := filepath.EvalSymlinks(dir)
	if text := s.Text(); !strings.Contains(text, "one") || !strings.Contains(text, filepath.Base(real)) {
		t.Fatalf("unexpected output:\n%s", text)
	}
	if s.Tag != TagBuild || s.ProjectID != "local/svc" || s.Title != "svc · build" || s.Interactive {
		t.Fatalf("unexpected session metadata %+v", s)
	}
}

func TestStartBuild_UsesAssignedMavenProfileAndRunsCleanOnce(t *testing.T) {
	m := newManager(t)
	mvnHome := t.TempDir()
	_ = os.MkdirAll(filepath.Join(mvnHome, "bin"), 0o755)
	_ = os.WriteFile(filepath.Join(mvnHome, "bin", "mvn"), []byte("#!/bin/sh\necho \"profile-mvn $*\"\n"), 0o755)
	cfg := config.Config{Mavens: []config.Profile{{Name: "mvn-x", Path: mvnHome}}}
	proj := config.Project{ID: "local/api", Name: "api", Path: t.TempDir(), Type: "maven", MavenName: "mvn-x"}

	id, err := StartBuild(m, cfg, proj, "test", TagBuild, 80, 24, nil)
	if err != nil {
		t.Fatal(err)
	}
	waitDone(t, m.Get(id))

	if text := m.Get(id).Text(); !strings.Contains(text, "profile-mvn clean test") {
		t.Fatalf("expected the profile's mvn to run 'clean test', got:\n%s", text)
	}
}

func TestStartBuild_UnknownCommandFails(t *testing.T) {
	_, err := StartBuild(newManager(t), config.Config{}, config.Project{Name: "x", Type: "go"}, "deploy", TagBuild, 80, 24, nil)
	if err == nil || !strings.Contains(err.Error(), `"deploy"`) {
		t.Fatalf("expected a clear error, got %v", err)
	}
}

func TestStartBuild_ReplacesFinishedBuildsUnlessKept(t *testing.T) {
	m := newManager(t)
	proj := config.Project{ID: "local/svc", Name: "svc", Path: t.TempDir(), Commands: map[string]string{"build": "true"}}

	first, _ := StartBuild(m, config.Config{}, proj, "build", TagBuild, 80, 24, nil)
	waitDone(t, m.Get(first))
	second, _ := StartBuild(m, config.Config{}, proj, "build", TagBuild, 80, 24, func(id int) bool { return id == first })
	waitDone(t, m.Get(second))
	if m.Get(first) == nil {
		t.Fatal("a build kept open (e.g. as a terminal tab) must not be closed")
	}

	third, _ := StartBuild(m, config.Config{}, proj, "build", TagBuild, 80, 24, nil)
	if m.Get(first) != nil || m.Get(second) != nil || m.Get(third) == nil {
		t.Fatalf("finished builds must be replaced, sessions: %v", m.List())
	}
	if LatestBuild(m, "local/svc", TagBuild).ID != third {
		t.Fatal("LatestBuild must return the newest build")
	}
}

func TestRunningBuilds(t *testing.T) {
	m := newManager(t)
	slow := config.Project{ID: "local/slow", Name: "slow", Path: t.TempDir(), Commands: map[string]string{"build": "sleep 5"}}
	id, _ := StartBuild(m, config.Config{}, slow, "build", TagBuild, 80, 24, nil)
	_, _ = m.Start(shell.StartOptions{Dir: t.TempDir(), Argv: []string{"sh"}, Interactive: true})

	if RunningBuilds(m) != 1 || RunningBuild(m, "local/slow").ID != id {
		t.Fatal("only non-interactive sessions count as builds")
	}
	if RunningBuild(m, "local/other") != nil || RunningBuild(nil, "x") != nil || RunningBuilds(nil) != 0 {
		t.Fatal("unexpected running build")
	}
}
