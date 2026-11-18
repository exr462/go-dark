package componentaction

import (
	"errors"
	"fmt"
	"os"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/runner"
	"github.com/exr462/go-dark/shell"
)

// OpenTerminal shows the terminal pane: the existing tabs if there are any,
// otherwise a new shell in the selected project.
func OpenTerminal(ui *model.UI, sessions *shell.Manager) tea.Cmd {
	pruneTerminalTabs(ui, sessions)
	if len(ui.TerminalTabs) == 0 {
		var proj config.Project
		if ui.SelectedProject >= 0 && ui.SelectedProject < len(ui.Config.Projects) {
			proj = ui.Config.Projects[ui.SelectedProject]
		}
		if err := startTerminalTab(ui, sessions, proj); err != nil {
			ui.StatusMsg = fmt.Sprintf("❌ Terminal: %v", err)
			return nil
		}
	}
	ui.TerminalPicking = false
	ui.ViewState = model.StateTerminalCockpit
	markActiveSeen(ui, sessions)
	return nil
}

// AttachSession shows an existing session (a build, a pipeline step, a
// shell) in the terminal pane, adding it as a tab if needed.
func AttachSession(ui *model.UI, sessions *shell.Manager, id int) {
	s := sessions.Get(id)
	if s == nil {
		return
	}
	pruneTerminalTabs(ui, sessions)
	i := slices.Index(ui.TerminalTabs, id)
	if i < 0 {
		w, h := ui.TerminalPaneSize()
		_ = sessions.Resize(id, w, h)
		ui.TerminalTabs = append(ui.TerminalTabs, id)
		i = len(ui.TerminalTabs) - 1
	}
	ui.ActiveTerminalTab = i
	ui.TerminalPicking = false
	ui.ViewState = model.StateTerminalCockpit
	s.MarkSeen()
}

// TerminalExecutionModal routes keys inside the terminal pane. The terminal
// shortcuts (all configurable) are handled here; every other key goes to the
// active shell, so esc, ctrl+c, arrows etc. behave like in a real terminal.
func TerminalExecutionModal(ui *model.UI, sessions *shell.Manager, msg tea.KeyPressMsg) tea.Cmd {
	if ui.TerminalPicking {
		return terminalProjectPicker(ui, sessions, msg)
	}

	key := msg.String()
	switch key {
	case binding(ui, action.TerminalDetach):
		ui.ViewState = model.StateDashboard
		return nil
	case binding(ui, action.TerminalNewTab):
		ui.TerminalPicking = true
		ui.TerminalPickIndex = max(ui.SelectedProject, 0)
		return nil
	case binding(ui, action.TerminalNextTab):
		cycleTerminalTab(ui, sessions, 1)
		return nil
	case binding(ui, action.TerminalPrevTab):
		cycleTerminalTab(ui, sessions, -1)
		return nil
	case binding(ui, action.TerminalCloseTab):
		closeActiveTerminalTab(ui, sessions)
		return nil
	}

	id := ui.ActiveTerminalSession()
	s := sessions.Get(id)
	if s == nil {
		closeActiveTerminalTab(ui, sessions)
		return nil
	}
	if !s.Running() {
		// The shell exited (e.g. `exit`): enter closes the tab.
		if key == "enter" {
			closeActiveTerminalTab(ui, sessions)
		}
		return nil
	}
	_ = sessions.SendKey(id, msg)
	return nil
}

// TerminalPaste pastes into the active shell.
func TerminalPaste(ui *model.UI, sessions *shell.Manager, text string) {
	if ui.TerminalPicking {
		return
	}
	_ = sessions.Paste(ui.ActiveTerminalSession(), text)
}

// ResizeTerminals keeps every shell the size of the terminal pane.
func ResizeTerminals(ui *model.UI, sessions *shell.Manager) {
	w, h := ui.TerminalPaneSize()
	for _, id := range ui.TerminalTabs {
		_ = sessions.Resize(id, w, h)
	}
}

func binding(ui *model.UI, a action.Action) string {
	return action.GetShortcutKeyBinding(ui.Config.ShortCuts, a)
}

func terminalProjectPicker(ui *model.UI, sessions *shell.Manager, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		ui.TerminalPickIndex = max(ui.TerminalPickIndex-1, 0)
	case "down", "j":
		ui.TerminalPickIndex = min(ui.TerminalPickIndex+1, max(len(ui.Config.Projects)-1, 0))
	case "esc", binding(ui, action.TerminalDetach):
		ui.TerminalPicking = false
		if len(ui.TerminalTabs) == 0 {
			ui.ViewState = model.StateDashboard
		}
	case "enter":
		var proj config.Project
		if ui.TerminalPickIndex < len(ui.Config.Projects) {
			proj = ui.Config.Projects[ui.TerminalPickIndex]
		}
		ui.TerminalPicking = false
		if err := startTerminalTab(ui, sessions, proj); err != nil {
			ui.StatusMsg = fmt.Sprintf("❌ Terminal: %v", err)
		}
		markActiveSeen(ui, sessions)
	}
	return nil
}

// startTerminalTab opens a login shell for proj (with its JDK/Maven
// environment) in a new tab and makes it active.
func startTerminalTab(ui *model.UI, sessions *shell.Manager, proj config.Project) error {
	w, h := ui.TerminalPaneSize()
	id, err := sessions.Start(shell.StartOptions{
		ProjectID:   proj.ID,
		ProjectName: terminalTitle(proj),
		Dir:         terminalDir(ui, proj),
		Env:         runner.Env(ui.Config, proj),
		Interactive: true,
		Width:       w,
		Height:      h,
	})
	if errors.Is(err, shell.ErrUnsupported) {
		return errors.New("the embedded terminal isn't available on Windows yet")
	}
	if err != nil {
		return err
	}
	ui.TerminalTabs = append(ui.TerminalTabs, id)
	ui.ActiveTerminalTab = len(ui.TerminalTabs) - 1
	return nil
}

func terminalTitle(proj config.Project) string {
	if proj.Name == "" {
		return "shell"
	}
	return proj.Name
}

// terminalDir is the project's folder if it exists (it may not be cloned
// yet), otherwise the workspace, otherwise the home directory.
func terminalDir(ui *model.UI, proj config.Project) string {
	for _, dir := range []string{proj.Path, ui.Config.BasePath} {
		if dir == "" {
			continue
		}
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	home, _ := os.UserHomeDir()
	return home
}

func cycleTerminalTab(ui *model.UI, sessions *shell.Manager, step int) {
	if n := len(ui.TerminalTabs); n > 0 {
		ui.ActiveTerminalTab = (ui.ActiveTerminalTab + step + n) % n
		markActiveSeen(ui, sessions)
	}
}

func closeActiveTerminalTab(ui *model.UI, sessions *shell.Manager) {
	id := ui.ActiveTerminalSession()
	if id == 0 {
		ui.ViewState = model.StateDashboard
		return
	}
	_ = sessions.Close(id)
	ui.TerminalTabs = slices.Delete(ui.TerminalTabs, ui.ActiveTerminalTab, ui.ActiveTerminalTab+1)
	if ui.ActiveTerminalTab >= len(ui.TerminalTabs) {
		ui.ActiveTerminalTab = max(len(ui.TerminalTabs)-1, 0)
	}
	if len(ui.TerminalTabs) == 0 {
		ui.ViewState = model.StateDashboard
		return
	}
	markActiveSeen(ui, sessions)
}

// pruneTerminalTabs drops tabs whose session no longer exists.
func pruneTerminalTabs(ui *model.UI, sessions *shell.Manager) {
	ui.TerminalTabs = slices.DeleteFunc(ui.TerminalTabs, func(id int) bool { return sessions.Get(id) == nil })
	if ui.ActiveTerminalTab >= len(ui.TerminalTabs) {
		ui.ActiveTerminalTab = max(len(ui.TerminalTabs)-1, 0)
	}
}

func markActiveSeen(ui *model.UI, sessions *shell.Manager) {
	if s := sessions.Get(ui.ActiveTerminalSession()); s != nil {
		s.MarkSeen()
	}
}
