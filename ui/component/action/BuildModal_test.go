//go:build !windows

package componentaction

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/session"
)

func newBuildTestUI(t *testing.T, buildCmd string) *model.UI {
	t.Helper()
	ui, m := newTerminalTestUI(t)
	ui.Shells = m
	ui.Config.Projects[0].Commands = map[string]string{"build": buildCmd}
	ui.Config.Projects[0].Type = "go"
	ui.ViewState = model.StateBuildModal
	ui.BuildOptions = config.CommandKeys(ui.Config.Projects[0])
	return ui
}

func TestBuildModal_RunStaysInModalAndStreamsOutput(t *testing.T) {
	ui := newBuildTestUI(t, "echo compiling; sleep 0.3; echo done-building")

	BuildModal(ui, testKeyMsg("ctrl+s"))

	if ui.ViewState != model.StateBuildModal {
		t.Fatal("starting a build must keep the modal open to watch it")
	}
	b := session.LatestBuild(ui.Shells, "rail/api", session.TagBuild)
	if b == nil || !b.Running() {
		t.Fatal("expected a running build session")
	}
	BuildModal(ui, testKeyMsg("ctrl+s"))
	if len(ui.Shells.List()) != 1 {
		t.Fatal("a second run must not start while the build is running")
	}
	eventually(t, "the build to finish", func() bool { return !b.Running() })
	if !strings.Contains(b.Text(), "done-building") || b.ExitCode() != 0 {
		t.Fatalf("unexpected build result %d:\n%s", b.ExitCode(), b.Text())
	}
}

func TestBuildModal_EscapeLeavesBuildRunningAndTerminalKeyAttaches(t *testing.T) {
	ui := newBuildTestUI(t, "echo started; sleep 30")
	BuildModal(ui, testKeyMsg("ctrl+s"))
	b := session.LatestBuild(ui.Shells, "rail/api", session.TagBuild)
	// A ctrl+c sent before the child owns the PTY would have no process to
	// signal; wait until it runs, as a person would.
	eventually(t, "the build to start", func() bool { return strings.Contains(b.Text(), "started") })

	BuildModal(ui, testKeyMsg("esc"))
	if ui.ViewState != model.StateDashboard || !b.Running() {
		t.Fatal("leaving the modal must not stop the build")
	}

	ui.ViewState = model.StateBuildModal
	BuildModal(ui, tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if ui.ViewState != model.StateTerminalCockpit || ui.ActiveTerminalSession() != b.ID {
		t.Fatalf("the terminal key must attach to the build, tabs=%v", ui.TerminalTabs)
	}

	// ctrl+c in the attached pane cancels the build.
	TerminalExecutionModal(ui, ui.Shells, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	eventually(t, "ctrl+c to cancel the build", func() bool { return !b.Running() })
	if b.ExitCode() == 0 {
		t.Fatal("a cancelled build must not report success")
	}
}

func TestBuildModal_UnknownCommandShowsError(t *testing.T) {
	ui := newBuildTestUI(t, "true")
	ui.BuildOptions = []string{"deploy"}

	BuildModal(ui, testKeyMsg("ctrl+s"))

	if !strings.Contains(ui.StatusMsg, "deploy") || len(ui.Shells.List()) != 0 {
		t.Fatalf("expected an error and no session, got %q", ui.StatusMsg)
	}
}

func TestSessionInspector_SelectAttachAndClose(t *testing.T) {
	ui := newBuildTestUI(t, "echo hi; sleep 30")
	BuildModal(ui, testKeyMsg("ctrl+s"))
	OpenTerminal(ui, ui.Shells) // a shell too
	ui.ViewState = model.StateSessionLogsModal
	build := ui.Shells.List()[0]

	SessionLogsModal(ui, testKeyMsg("down"))
	SessionLogsModal(ui, testKeyMsg("up"))
	SessionLogsModal(ui, testKeyMsg("enter"))
	if ui.ViewState != model.StateTerminalCockpit || ui.ActiveTerminalSession() != build.ID {
		t.Fatal("enter must open the selected session in the terminal")
	}

	ui.ViewState = model.StateSessionLogsModal
	SessionLogsModal(ui, testKeyMsg("d"))
	time.Sleep(50 * time.Millisecond)
	if ui.Shells.Get(build.ID) != nil || build.Running() {
		t.Fatal("d must kill and remove the session")
	}
	for _, id := range ui.TerminalTabs {
		if id == build.ID {
			t.Fatal("a closed session must also leave the terminal tabs")
		}
	}

	SessionLogsModal(ui, testKeyMsg("esc"))
	if ui.ViewState != model.StateDashboard {
		t.Fatal("esc must return to the dashboard")
	}
}
