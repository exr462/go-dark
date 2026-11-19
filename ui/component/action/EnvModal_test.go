//go:build !windows

package componentaction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/ui/component"
)

// envTestUI subscribes a team whose cached manifest declares an
// environment, fakes docker/kubectl/helm (the old sandbox holds the ports)
// and makes the lifecycle sessions run a fake go-dark that logs its
// arguments.
func envTestUI(t *testing.T) (*model.UI, string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	team := config.Team{Name: "rail-manifests", GitURL: "git@host:rail/rail-manifests.git", Manifest: "rail"}
	dir, _ := config.TeamDir(team)
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, config.ManifestFileName), []byte(`{"team":"rail","projects":[{"name":"til-order","deployable":true}],
		"environment":{"application":"til","domain":"lcl.local"}}`), 0o644)

	bin := t.TempDir()
	log := filepath.Join(bin, "calls.log")
	scripts := map[string]string{
		"docker": `echo "docker $*" >> ` + log + `
case "$*" in
  "info "*) echo "8 8345035776";;
  "compose version") echo v2;;
  "ps "*) printf 'rancher|local-cicd-sandbox|127.0.0.1:6550->6443/tcp\ntraefik|local-cicd-sandbox|0.0.0.0:80->80/tcp, 0.0.0.0:443->443/tcp\n';;
  "inspect "*) exit 1;;
esac`,
		"kubectl": `echo "kubectl $*" >> ` + log,
		"helm":    `echo "helm $*" >> ` + log,
		"mkcert":  `exit 0`,
		"go-dark": `echo "go-dark $*" >> ` + log + `; case "$*" in *fail*) exit 3;; esac`,
	}
	for name, body := range scripts {
		_ = os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	orig := Executable
	Executable = func() (string, error) { return filepath.Join(bin, "go-dark"), nil }
	t.Cleanup(func() { Executable = orig })

	m := shell.NewManager()
	t.Cleanup(m.Shutdown)
	go func() {
		for {
			m.Listen()()
		}
	}()
	ui := &model.UI{Config: config.Config{ShortCuts: action.DefaultShortcuts, Teams: []config.Team{team}, BasePath: t.TempDir()},
		Shells: m, WindowWidth: 160, WindowHeight: 50}
	ui.Config.RefreshProjects()
	ui.Config.Projects[0].Fetched = true
	return ui, log
}

func waitAction(t *testing.T, ui *model.UI) *shell.Session {
	t.Helper()
	s := ui.Shells.Get(ui.EnvActionID)
	if s == nil {
		t.Fatal("expected a lifecycle session")
	}
	<-s.Done()
	return s
}

func TestEnv_ChecksShowTheOldSandboxAndStopIt(t *testing.T) {
	ui, log := envTestUI(t)

	cmd := OpenEnv(ui)
	if ui.ViewState != model.StateEnvModal || cmd == nil {
		t.Fatal("opening the screen starts the checks")
	}
	EnvChecked(ui, cmd().(model.EnvCheckedMsg))
	out := ansi.Strip(component.RenderEnvModal(ui))
	for _, want := range []string{`"lcl" of team rail`, "/.local/share/go-dark/env/rail", "published by container traefik (compose project local-cicd-sandbox)",
		"docker compose -p local-cicd-sandbox stop", "Another compose project holds the ports", "not installed (i installs it)", "missing api.til.lcl.local, rancher.lcl.local",
		"1 of 1 deployable services cloned", "[o] Stop the old sandbox"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen must show %q:\n%s", want, out)
		}
	}

	EnvModal(ui, testKeyMsg("o"))
	if ui.EnvConfirm != "stop-sandbox" {
		t.Fatal("o asks for confirmation")
	}
	EnvModal(ui, testKeyMsg("n"))
	if ui.EnvConfirm != "" || ui.EnvActionID != 0 {
		t.Fatal("any other key cancels")
	}
	EnvModal(ui, testKeyMsg("o"))
	EnvModal(ui, testKeyMsg("y"))
	s := waitAction(t, ui)
	if s.Tag != TagEnv || s.Command != "docker compose -p local-cicd-sandbox stop" {
		t.Fatalf("stopping the sandbox must stop its compose project only, got %q", s.Command)
	}
	calls, _ := os.ReadFile(log)
	for _, line := range strings.Split(string(calls), "\n") {
		if strings.HasPrefix(line, "docker compose -p") && !strings.HasSuffix(line, " stop") || strings.HasPrefix(line, "docker rm") || strings.HasPrefix(line, "docker volume") {
			t.Fatalf("never remove the sandbox's volumes or files: %s", line)
		}
	}
	if EnvActionExited(ui, s.ID) == nil || !strings.Contains(ui.StatusMsg, "stop-sandbox finished") {
		t.Fatalf("a finished action re-checks, got %q", ui.StatusMsg)
	}
}

func TestEnv_LifecycleActionsRunGoDarkEnvAsSessions(t *testing.T) {
	ui, _ := envTestUI(t)
	OpenEnv(ui)
	dir, _ := config.TeamDir(ui.Config.Teams[0])
	manifest := filepath.Join(dir, config.ManifestFileName)
	install := filepath.Join(os.Getenv("HOME"), ".local", "share", "go-dark", "env", "rail")

	EnvModal(ui, testKeyMsg("i"))
	s := waitAction(t, ui)
	want := Executable
	exe, _ := want()
	if s.Command != exe+" env install --manifest "+manifest+" --dir "+install+" --project-dir "+ui.Config.Projects[0].Path {
		t.Fatalf("install command %q", s.Command)
	}
	if _, err := os.Stat(install); err != nil {
		t.Fatal("the install dir is created for the session")
	}
	if EnvModal(ui, testKeyMsg("s")) != nil {
		t.Fatal("nothing to return")
	}
	if s := waitAction(t, ui); !strings.HasSuffix(s.Command, " env start --manifest "+manifest+" --dir "+install+" --project-dir "+ui.Config.Projects[0].Path) {
		t.Fatalf("start command %q", s.Command)
	}
	EnvModal(ui, testKeyMsg("t"))
	if s := waitAction(t, ui); !strings.Contains(s.Command, " env stop ") {
		t.Fatalf("stop command %q", s.Command)
	}

	EnvModal(ui, testKeyMsg("u"))
	EnvModal(ui, testKeyMsg("y"))
	if s := waitAction(t, ui); !strings.Contains(s.Command, " env uninstall ") || strings.Contains(s.Command, "--volumes") {
		t.Fatalf("y uninstalls keeping the volumes, got %q", s.Command)
	}
	EnvModal(ui, testKeyMsg("u"))
	EnvModal(ui, testKeyMsg("V"))
	if s := waitAction(t, ui); !strings.HasSuffix(s.Command, " --volumes") {
		t.Fatalf("V removes the volumes too, got %q", s.Command)
	}

	out := ansi.Strip(component.RenderEnvModal(ui))
	if !strings.Contains(out, "Last action  uninstall ✅ finished") || !strings.Contains(out, "[l] Log") {
		t.Fatalf("the last action and its log must be shown:\n%s", out)
	}
	EnvModal(ui, testKeyMsg("l"))
	if ui.ViewState != model.StateTerminalCockpit || ui.ActiveTerminalSession() != ui.EnvActionID {
		t.Fatal("l opens the action's log in the terminal pane")
	}
}

func TestEnv_FailedActionIsReported(t *testing.T) {
	ui, _ := envTestUI(t)
	OpenEnv(ui)
	ui.Config.LocalCluster.Database.Password = "fail-me" // makes the fake go-dark exit 3
	EnvModal(ui, testKeyMsg("i"))
	s := waitAction(t, ui)
	if !strings.Contains(s.Command, "--db-password fail-me") {
		t.Fatalf("the configured database password is passed on, got %q", s.Command)
	}
	EnvActionExited(ui, s.ID)
	if !strings.Contains(ui.StatusMsg, "install failed (exit 3)") {
		t.Fatalf("status %q", ui.StatusMsg)
	}
	if out := ansi.Strip(component.RenderEnvModal(ui)); !strings.Contains(out, "install ❌ failed (exit 3)") {
		t.Fatalf("screen:\n%s", out)
	}
}

func TestEnv_HostsFileGoesThroughSudoInTheTerminal(t *testing.T) {
	ui, _ := envTestUI(t)
	OpenEnv(ui)
	EnvModal(ui, testKeyMsg("h"))
	id := ui.ActiveTerminalSession()
	s := ui.Shells.Get(id)
	if ui.ViewState != model.StateTerminalCockpit || s == nil || !s.Interactive || s.Title != "hosts file" {
		t.Fatalf("h must open an interactive terminal tab, got %v %+v", ui.ViewState, s)
	}
	if !strings.Contains(s.Command, "sudo tee -a /etc/hosts") || !strings.HasSuffix(s.Command, "127.0.0.1 api.til.lcl.local rancher.lcl.local") {
		t.Fatalf("the hosts line is added through sudo, got %q", s.Command)
	}
	_ = ui.Shells.Kill(id)
}

func TestEnv_NoEnvironmentDeclared(t *testing.T) {
	ui := &model.UI{Config: config.Config{ShortCuts: action.DefaultShortcuts}, WindowWidth: 120, WindowHeight: 40}
	if OpenEnv(ui) != nil {
		t.Fatal("nothing to check")
	}
	if out := ansi.Strip(component.RenderEnvModal(ui)); !strings.Contains(out, "No subscribed team manifest declares") {
		t.Fatalf("screen:\n%s", out)
	}
	if EnvModal(ui, testKeyMsg("i")) != nil || ui.EnvActionID != 0 {
		t.Fatal("no environment, no action")
	}
}
