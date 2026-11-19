package componentaction

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/env"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
)

// TagEnv marks the sessions of the installer screen's lifecycle actions.
const TagEnv = "env"

// Executable is this program, which the lifecycle actions run as
// `go-dark env ...` sessions; replaced in tests.
var Executable = os.Executable

// OpenEnv shows the installer screen for the team's local environment and
// starts its checks.
func OpenEnv(ui *model.UI) tea.Cmd {
	ui.ViewState = model.StateEnvModal
	ui.EnvConfirm = ""
	if len(ui.Config.Environments) == 0 {
		ui.EnvChecked = false
		return nil
	}
	ui.EnvIndex = min(max(ui.EnvIndex, 0), len(ui.Config.Environments)-1)
	return CheckEnv(ui)
}

// envInputs lists the team's cloned, deployable services (their config
// provides the database users).
func envInputs(ui *model.UI, e env.Env) env.Inputs {
	in := env.Inputs{Dir: model.EnvDir(e)}
	for _, p := range ui.Config.Projects {
		if p.Source == e.Team && p.Deployable && p.Fetched {
			in.ProjectDirs = append(in.ProjectDirs, p.Path)
		}
	}
	return in
}

// CheckEnv runs the preflight, the port conflicts and the status in the
// background.
func CheckEnv(ui *model.UI) tea.Cmd {
	e, ok := ui.CurrentEnv()
	if !ok {
		return nil
	}
	ui.EnvChecking = true
	dir := model.EnvDir(e)
	return func() tea.Msg {
		return model.EnvCheckedMsg{Team: e.Team, Checks: env.Preflight(e), Conflicts: env.Conflicts(e), Status: env.Describe(e, dir)}
	}
}

// EnvChecked stores the checks (of the environment currently shown).
func EnvChecked(ui *model.UI, msg model.EnvCheckedMsg) {
	if e, ok := ui.CurrentEnv(); !ok || e.Team != msg.Team {
		return
	}
	ui.EnvChecking = false
	ui.EnvChecked = true
	ui.EnvChecks, ui.EnvConflicts, ui.EnvStatus = msg.Checks, msg.Conflicts, msg.Status
}

// EnvModal is the installer screen: i installs or repairs, s starts, t
// stops, u uninstalls (then y keeps the data, V removes the volumes too), h
// adds the hosts line (sudo in the embedded terminal), o stops the old
// sandbox holding the ports (then y), l opens the running/last action's
// log, r re-checks, tab shows the next team's environment, esc leaves.
func EnvModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	if ui.EnvConfirm != "" {
		return envConfirm(ui, key)
	}
	switch key {
	case binding(ui, action.Escape):
		ui.ViewState = model.StateDashboard
	case "r":
		return CheckEnv(ui)
	case "tab":
		if n := len(ui.Config.Environments); n > 1 {
			ui.EnvIndex = (ui.EnvIndex + 1) % n
			ui.EnvChecked = false
			return CheckEnv(ui)
		}
	case "i":
		return envAction(ui, "install", false)
	case "s":
		return envAction(ui, "start", false)
	case "t":
		return envAction(ui, "stop", false)
	case "u":
		if _, ok := ui.CurrentEnv(); ok && !envBusy(ui) {
			ui.EnvConfirm = "uninstall"
			ui.StatusMsg = "⚠ Uninstall: y removes the container and keeps the cluster data, V removes the volumes too, any other key cancels"
		}
	case "o":
		if len(ui.EnvConflicts) > 0 && !envBusy(ui) {
			ui.EnvConfirm = "stop-sandbox"
			ui.StatusMsg = fmt.Sprintf("⚠ Stop %s? y stops it (its volumes and files are kept), any other key cancels", conflictNames(ui))
		}
	case "h":
		return envHosts(ui)
	case "l":
		if ui.EnvActionID != 0 && ui.Shells != nil {
			AttachSession(ui, ui.Shells, ui.EnvActionID)
		}
	}
	return nil
}

func envConfirm(ui *model.UI, key string) tea.Cmd {
	what := ui.EnvConfirm
	ui.EnvConfirm = ""
	switch {
	case what == "uninstall" && key == "y":
		return envAction(ui, "uninstall", false)
	case what == "uninstall" && key == "V":
		return envAction(ui, "uninstall", true)
	case what == "stop-sandbox" && key == "y":
		return stopConflicts(ui)
	}
	ui.StatusMsg = "Cancelled"
	return nil
}

func envBusy(ui *model.UI) bool {
	if ui.EnvActionID == 0 || ui.Shells == nil {
		return false
	}
	s := ui.Shells.Get(ui.EnvActionID)
	return s != nil && s.Running()
}

// envAction runs a lifecycle action as a `go-dark env` session.
func envAction(ui *model.UI, name string, volumes bool) tea.Cmd {
	e, ok := ui.CurrentEnv()
	switch {
	case !ok || ui.Shells == nil:
		return nil
	case envBusy(ui):
		ui.StatusMsg = fmt.Sprintf("⏳ %s is still running (l shows its log)", ui.EnvAction)
		return nil
	}
	manifest, err := env.ManifestPath(ui.Config.Teams, e.Team)
	if err != nil {
		ui.StatusMsg = "❌ " + err.Error()
		return nil
	}
	exe, err := Executable()
	if err != nil {
		ui.StatusMsg = "❌ " + err.Error()
		return nil
	}
	in := envInputs(ui, e)
	if err := os.MkdirAll(in.Dir, 0o755); err != nil {
		ui.StatusMsg = "❌ " + err.Error()
		return nil
	}
	w, h := ui.TerminalPaneSize()
	id, err := ui.Shells.Start(shell.StartOptions{
		ProjectName: e.Team, Dir: in.Dir, Title: "env · " + name, Tag: TagEnv, Width: w, Height: h,
		Argv: env.Argv(exe, name, manifest, in, ui.Config.LocalCluster.Database.Password, volumes),
	})
	if err != nil {
		ui.StatusMsg = fmt.Sprintf("❌ %s: %v", name, err)
		return nil
	}
	ui.EnvActionID, ui.EnvAction = id, name
	ui.StatusMsg = fmt.Sprintf("⏳ %s running as session #%d (l shows it live)", name, id)
	return nil
}

// stopConflicts stops the compose projects (or containers) holding the
// environment's ports - the old sandbox - without touching their volumes.
func stopConflicts(ui *model.UI) tea.Cmd {
	if ui.Shells == nil || len(ui.EnvConflicts) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var argv []string
	for _, c := range ui.EnvConflicts {
		key := c.Project
		if key == "" {
			key = "container:" + c.Container
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		if len(argv) > 0 {
			argv = append(argv, "&&")
		}
		argv = append(argv, env.StopArgv(c)...)
	}
	if len(seen) > 1 {
		// Several projects: one shell command so it stays one session.
		argv = []string{"sh", "-c", strings.Join(argv, " ")}
	}
	w, h := ui.TerminalPaneSize()
	id, err := ui.Shells.Start(shell.StartOptions{ProjectName: "sandbox", Title: "env · stop " + conflictNames(ui), Tag: TagEnv, Width: w, Height: h, Argv: argv})
	if err != nil {
		ui.StatusMsg = "❌ " + err.Error()
		return nil
	}
	ui.EnvActionID, ui.EnvAction = id, "stop-sandbox"
	ui.StatusMsg = fmt.Sprintf("⏳ Stopping %s (session #%d)", conflictNames(ui), id)
	return nil
}

func conflictNames(ui *model.UI) string {
	seen := map[string]bool{}
	names := ""
	for _, c := range ui.EnvConflicts {
		name := c.Project
		if name == "" {
			name = c.Container
		}
		if !seen[name] {
			seen[name] = true
			if names != "" {
				names += ", "
			}
			names += name
		}
	}
	return names
}

// envHosts adds the hosts line through sudo in the embedded terminal, so
// the password is typed there and nothing edits the hosts file silently.
// Without a PTY (Windows) the elevated command is shown instead.
func envHosts(ui *model.UI) tea.Cmd {
	e, ok := ui.CurrentEnv()
	if !ok {
		return nil
	}
	if len(ui.EnvStatus.Missing) == 0 && ui.EnvChecked {
		ui.StatusMsg = "✅ The hosts file already has every host"
		return nil
	}
	argv, hint := env.HostsCommand(e)
	ui.EnvHint = hint
	if argv == nil || ui.Shells == nil {
		ui.StatusMsg = "Run in an elevated PowerShell: " + hint
		return nil
	}
	w, h := ui.TerminalPaneSize()
	id, err := ui.Shells.Start(shell.StartOptions{ProjectName: e.Team, Title: "hosts file", Interactive: true, Width: w, Height: h, Argv: argv})
	if errors.Is(err, shell.ErrUnsupported) {
		ui.StatusMsg = "Run with administrator rights: " + hint
		return nil
	}
	if err != nil {
		ui.StatusMsg = "❌ " + err.Error()
		return nil
	}
	ui.EnvHint = ""
	AttachSession(ui, ui.Shells, id)
	ui.StatusMsg = "Type your password for sudo; " + binding(ui, action.TerminalDetach) + " comes back"
	return nil
}

// EnvActionExited reports a finished lifecycle action and re-checks.
func EnvActionExited(ui *model.UI, id int) tea.Cmd {
	if id != ui.EnvActionID || ui.Shells == nil {
		return nil
	}
	s := ui.Shells.Get(id)
	if s == nil {
		return nil
	}
	if s.ExitCode() == 0 {
		ui.StatusMsg = fmt.Sprintf("✅ %s finished (session #%d)", ui.EnvAction, id)
	} else {
		ui.StatusMsg = fmt.Sprintf("❌ %s failed (exit %d) - l shows the log", ui.EnvAction, s.ExitCode())
	}
	return CheckEnv(ui)
}
