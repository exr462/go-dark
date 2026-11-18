//go:build !windows

package component

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
)

func startTestShell(t *testing.T, ui *model.UI, m *shell.Manager, title, script string) int {
	t.Helper()
	w, h := ui.TerminalPaneSize()
	id, err := m.Start(shell.StartOptions{ProjectName: title, Dir: t.TempDir(), Argv: []string{"sh", "-c", script}, Width: w, Height: h})
	if err != nil {
		t.Fatal(err)
	}
	ui.TerminalTabs = append(ui.TerminalTabs, id)
	return id
}

func newRenderUI() *model.UI {
	return &model.UI{
		Config:       config.Config{ShortCuts: action.DefaultShortcuts},
		WindowWidth:  90,
		WindowHeight: 28,
		ViewState:    model.StateTerminalCockpit,
	}
}

func TestRenderTerminal_LayoutMatchesPaneConstants(t *testing.T) {
	ui := newRenderUI()
	m := shell.NewManager()
	defer m.Shutdown()
	id := startTestShell(t, ui, m, "api", "printf 'MARK'; sleep 5")
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(m.Get(id).Text(), "MARK") && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	out, cursor := RenderTerminalExecution(ui, m)
	lines := strings.Split(ansi.Strip(out), "\n")

	if len(lines) > ui.WindowHeight {
		t.Fatalf("rendered %d lines into a %d-line window", len(lines), ui.WindowHeight)
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w > ui.WindowWidth {
			t.Fatalf("line %d is %d wide (window %d): %q", i, w, ui.WindowWidth, l)
		}
	}
	// The first screen row starts exactly at (TerminalPaneLeft, TerminalPaneTop).
	row := lines[model.TerminalPaneTop]
	if got := string([]rune(row)[model.TerminalPaneLeft:][:4]); got != "MARK" {
		t.Fatalf("expected the session's first row at the pane origin, row is %q", row)
	}
	if !strings.Contains(lines[model.TerminalPaneTop-1], "1 api") {
		t.Fatalf("expected the tab bar above the screen, got %q", lines[model.TerminalPaneTop-1])
	}
	if cursor == nil || cursor.Y != model.TerminalPaneTop || cursor.X != model.TerminalPaneLeft+4 {
		t.Fatalf("cursor must sit right after MARK, got %+v", cursor)
	}
}

func TestRenderTerminal_TabMarkersAndExitHint(t *testing.T) {
	ui := newRenderUI()
	m := shell.NewManager()
	defer m.Shutdown()
	done := startTestShell(t, ui, m, "done", "exit 0")
	busy := startTestShell(t, ui, m, "busy", "echo out; sleep 5")
	ui.ActiveTerminalTab = 0
	deadline := time.Now().Add(5 * time.Second)
	for (m.Get(done).Running() || !m.Get(busy).Unseen()) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	ui.WindowWidth = 160 // wide enough for the whole hint line
	out, cursor := RenderTerminalExecution(ui, m)
	plain := ansi.Strip(out)

	if !strings.Contains(plain, "1 done ✗") || !strings.Contains(plain, "2 busy*") {
		t.Fatalf("expected exited and unseen-output markers in the tab bar:\n%s", plain)
	}
	if !strings.Contains(plain, "enter closes the tab") {
		t.Fatal("an exited active tab must explain how to close it")
	}
	if cursor != nil {
		t.Fatal("no cursor for an exited shell")
	}
	if !strings.Contains(plain, "[ctrl+]] dashboard") || !strings.Contains(plain, "[alt+t] new tab") {
		t.Fatalf("hint must show the configured terminal shortcuts:\n%s", plain)
	}
}

func TestRenderTerminal_PickerListsProjects(t *testing.T) {
	ui := newRenderUI()
	ui.TerminalPicking = true
	ui.Config.Projects = []config.Project{{ID: "rail/api", Path: "/ws/api"}, {ID: "local/web", Path: "/ws/web"}}
	ui.TerminalPickIndex = 1

	out, cursor := RenderTerminalExecution(ui, shell.NewManager())
	plain := ansi.Strip(out)

	if !strings.Contains(plain, "rail/api") || !strings.Contains(plain, "> local/web") || cursor != nil {
		t.Fatalf("expected the project picker with web highlighted:\n%s", plain)
	}
}

func TestRenderBuildModalAndInspector_ShowLiveSessionOutput(t *testing.T) {
	ui := newRenderUI()
	m := shell.NewManager()
	defer m.Shutdown()
	ui.Shells = m
	ui.Config.Projects = []config.Project{{ID: "t/api", Name: "api", Type: "go"}}
	ui.BuildOptions = config.CommandKeys(ui.Config.Projects[0])
	id, _ := m.Start(shell.StartOptions{ProjectID: "t/api", Title: "api · build", Tag: "build", Dir: t.TempDir(),
		Argv: []string{"sh", "-c", "echo '[ERROR] boom'; exit 2"}})
	<-m.Get(id).Done()
	time.Sleep(50 * time.Millisecond)

	build := ansi.Strip(RenderBuildModal(ui))
	if !strings.Contains(build, "[ERROR] boom") || !strings.Contains(build, "exit code 2") || !strings.Contains(build, "Open in terminal") {
		t.Fatalf("build modal must show the session output and result:\n%s", build)
	}

	inspector := ansi.Strip(RenderSessionLogsModal(ui))
	if !strings.Contains(inspector, "api · build") || !strings.Contains(inspector, "EXIT 2") || !strings.Contains(inspector, "[ERROR] boom") {
		t.Fatalf("inspector must list the session with a preview:\n%s", inspector)
	}
}
