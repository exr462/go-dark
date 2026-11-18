//go:build !windows

package componentaction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
)

func newTerminalTestUI(t *testing.T) (*model.UI, *shell.Manager) {
	t.Helper()
	t.Setenv("SHELL", "/bin/sh")
	root := t.TempDir()
	for _, n := range []string{"api", "web"} {
		_ = os.Mkdir(filepath.Join(root, n), 0o755)
	}
	ui := &model.UI{
		Config: config.Config{
			BasePath:  root,
			ShortCuts: append([]action.Shortcut(nil), action.DefaultShortcuts...),
			Projects: []config.Project{
				{ID: "rail/api", Name: "api", Path: filepath.Join(root, "api")},
				{ID: "local/web", Name: "web", Path: filepath.Join(root, "web")},
			},
		},
		ViewState:    model.StateDashboard,
		WindowWidth:  100,
		WindowHeight: 30,
	}
	m := shell.NewManager()
	t.Cleanup(m.Shutdown)
	go func() {
		for {
			m.Listen()()
		}
	}()
	return ui, m
}

func press(ui *model.UI, m *shell.Manager, keys ...tea.KeyPressMsg) {
	for _, k := range keys {
		TerminalExecutionModal(ui, m, k)
	}
}

func typeInto(ui *model.UI, m *shell.Manager, line string) {
	for _, r := range line {
		press(ui, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	press(ui, m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func activeText(ui *model.UI, m *shell.Manager) string {
	if s := m.Get(ui.ActiveTerminalSession()); s != nil {
		return s.Text()
	}
	return ""
}

func TestOpenTerminal_StartsShellInSelectedProjectSizedToPane(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	ui.SelectedProject = 1

	OpenTerminal(ui, m)

	if ui.ViewState != model.StateTerminalCockpit || len(ui.TerminalTabs) != 1 {
		t.Fatalf("expected one tab in the terminal view, got state=%v tabs=%v", ui.ViewState, ui.TerminalTabs)
	}
	s := m.Get(ui.ActiveTerminalSession())
	if s.Title != "web" || !s.Interactive {
		t.Fatalf("unexpected session %+v", s)
	}
	if w, h := s.Size(); [2]int{w, h} != func() [2]int { w, h := ui.TerminalPaneSize(); return [2]int{w, h} }() {
		t.Fatalf("session must match the pane size, got %dx%d", w, h)
	}
	typeInto(ui, m, "basename \"$PWD\"")
	eventually(t, "the shell to run in the project dir", func() bool {
		return strings.Contains(activeText(ui, m), "\nweb")
	})
}

func TestTerminal_EscAndCtrlCGoToTheShell(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	OpenTerminal(ui, m)

	press(ui, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if ui.ViewState != model.StateTerminalCockpit {
		t.Fatal("esc must reach the shell, not leave the terminal")
	}
	typeInto(ui, m, "sleep 30")
	time.Sleep(200 * time.Millisecond)
	press(ui, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	typeInto(ui, m, "echo back-$((40+2))")
	eventually(t, "ctrl+c to interrupt sleep", func() bool { return strings.Contains(activeText(ui, m), "back-42") })
}

func TestTerminal_DetachKeepsShellAndReopenReattaches(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	OpenTerminal(ui, m)
	typeInto(ui, m, "export KEEP=still-here")
	id := ui.ActiveTerminalSession()

	press(ui, m, tea.KeyPressMsg{Code: ']', Mod: tea.ModCtrl}) // default detach
	if ui.ViewState != model.StateDashboard || !m.Get(id).Running() {
		t.Fatal("detach must return to the dashboard and keep the shell running")
	}

	OpenTerminal(ui, m)
	if len(ui.TerminalTabs) != 1 || ui.ActiveTerminalSession() != id {
		t.Fatalf("reopening must reattach to the same shell, got tabs %v", ui.TerminalTabs)
	}
	typeInto(ui, m, "echo $KEEP")
	eventually(t, "shell state to survive detaching", func() bool {
		return strings.Count(activeText(ui, m), "still-here") >= 2
	})
}

func TestTerminal_NewTabPickerCycleAndClose(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	OpenTerminal(ui, m) // tab 1: api

	press(ui, m, tea.KeyPressMsg{Code: 't', Mod: tea.ModAlt})
	if !ui.TerminalPicking {
		t.Fatal("new tab must open the project picker")
	}
	press(ui, m, tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(ui.TerminalTabs) != 2 || ui.ActiveTerminalTab != 1 || m.Get(ui.ActiveTerminalSession()).Title != "web" {
		t.Fatalf("expected a second, active tab for web, got %v active=%d", ui.TerminalTabs, ui.ActiveTerminalTab)
	}

	press(ui, m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModAlt})
	if ui.ActiveTerminalTab != 0 {
		t.Fatal("next tab must wrap around")
	}
	press(ui, m, tea.KeyPressMsg{Code: 'p', Mod: tea.ModAlt})
	if ui.ActiveTerminalTab != 1 {
		t.Fatal("previous tab must wrap around")
	}

	closing := ui.ActiveTerminalSession()
	press(ui, m, tea.KeyPressMsg{Code: 'w', Mod: tea.ModAlt})
	if m.Get(closing) != nil || len(ui.TerminalTabs) != 1 || ui.ViewState != model.StateTerminalCockpit {
		t.Fatalf("close must end that shell only, got tabs %v", ui.TerminalTabs)
	}
	press(ui, m, tea.KeyPressMsg{Code: 'w', Mod: tea.ModAlt})
	if len(ui.TerminalTabs) != 0 || ui.ViewState != model.StateDashboard || len(m.List()) != 0 {
		t.Fatal("closing the last tab must return to the dashboard")
	}
}

func TestTerminal_PickerEscWithoutTabsReturnsToDashboard(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	ui.ViewState = model.StateTerminalCockpit
	ui.TerminalPicking = true

	press(ui, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	if ui.TerminalPicking || ui.ViewState != model.StateDashboard {
		t.Fatal("cancelling the picker with no tabs must leave the terminal")
	}
}

func TestTerminal_ExitedShellClosesOnEnter(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	OpenTerminal(ui, m)
	id := ui.ActiveTerminalSession()

	typeInto(ui, m, "exit")
	eventually(t, "the shell to exit", func() bool { return !m.Get(id).Running() })
	press(ui, m, tea.KeyPressMsg{Code: 'x', Text: "x"})
	if len(ui.TerminalTabs) != 1 {
		t.Fatal("only enter closes an exited tab")
	}
	press(ui, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(ui.TerminalTabs) != 0 || ui.ViewState != model.StateDashboard {
		t.Fatal("enter must close the exited tab")
	}
}

func TestTerminal_ShortcutsAreConfigurable(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	for i := range ui.Config.ShortCuts {
		if ui.Config.ShortCuts[i].Action == action.TerminalDetach {
			ui.Config.ShortCuts[i].KeyBinding = "ctrl+x"
		}
	}
	OpenTerminal(ui, m)

	press(ui, m, tea.KeyPressMsg{Code: ']', Mod: tea.ModCtrl})
	if ui.ViewState != model.StateTerminalCockpit {
		t.Fatal("the old detach key must now go to the shell")
	}
	press(ui, m, tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	if ui.ViewState != model.StateDashboard {
		t.Fatal("the rebound detach key must leave the terminal")
	}
}

func TestTerminal_PasteGoesToActiveShell(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	OpenTerminal(ui, m)

	TerminalPaste(ui, m, "echo pasted-$((1+1))\n")

	eventually(t, "pasted command output", func() bool { return strings.Contains(activeText(ui, m), "pasted-2") })
}

func TestResizeTerminals_FollowsWindow(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	OpenTerminal(ui, m)

	ui.WindowWidth, ui.WindowHeight = 140, 50
	ResizeTerminals(ui, m)

	w, h := m.Get(ui.ActiveTerminalSession()).Size()
	if pw, ph := ui.TerminalPaneSize(); w != pw || h != ph {
		t.Fatalf("session %dx%d, pane %dx%d", w, h, pw, ph)
	}
}

func TestOpenTerminal_FallsBackWhenProjectNotCloned(t *testing.T) {
	ui, m := newTerminalTestUI(t)
	ui.Config.Projects[0].Path = filepath.Join(ui.Config.BasePath, "missing")

	OpenTerminal(ui, m)
	typeInto(ui, m, "pwd")

	base, _ := filepath.EvalSymlinks(ui.Config.BasePath)
	eventually(t, "the shell to start in the workspace", func() bool {
		return strings.Contains(activeText(ui, m), filepath.Base(base))
	})
}
