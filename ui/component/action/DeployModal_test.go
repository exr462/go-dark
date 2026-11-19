//go:build !windows

package componentaction

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/env"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/ui/component"
)

// TestMain lets the image side-load re-run this binary as `go-dark env
// import-image` (kube.ImportImageArgv), exactly like the real program.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "env" {
		os.Exit(env.CLI(os.Args[2:], os.Stdout))
	}
	os.Exit(m.Run())
}

// fakeCluster puts docker/kubectl/helm scripts on PATH. docker reports the
// rancher container as running or stopped; `docker start` flips it.
func fakeCluster(t *testing.T, running bool, server string) string {
	t.Helper()
	bin := t.TempDir()
	state := filepath.Join(bin, "state")
	log := filepath.Join(bin, "calls.log")
	_ = os.WriteFile(state, []byte(fmt.Sprint(running)), 0o644)
	scripts := map[string]string{
		"docker": `case "$1" in
  inspect) cat ` + state + `;;
  start) echo rancher; echo true > ` + state + `;;
  exec) cat > /dev/null; echo "docker $*" >> ` + log + `;;
  *) echo "docker $*" >> ` + log + `;;
esac`,
		"kubectl": `case "$*" in
  "config view"*) echo "` + server + `";;
  "get --raw"*) [ "$(cat ` + state + `)" = true ] || exit 1; echo ok;;
  "get nodes"*) echo arm64;;
  "get deployments"*) echo '{"items":[]}';;
  rollout*) echo "kubectl $*" >> ` + log + `;;
esac`,
		"helm": `echo "helm $*" >> ` + log + `
case "$1" in
  list) [ -f ` + bin + `/migrated ] && echo '[]' || echo '[{"name":"til-order","status":"deployed"}]';;
  uninstall) touch ` + bin + `/migrated;;
esac`,
	}
	for name, body := range scripts {
		_ = os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func deployTestUI(t *testing.T) *model.UI {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "til-order")
	chart := filepath.Join(dir, "helm", "til-order")
	_ = os.MkdirAll(chart, 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "order-service"), 0o755)
	_ = os.WriteFile(filepath.Join(chart, "Chart.yaml"), []byte("apiVersion: v2\nname: til-order\nversion: {JENKINS_VERSION}\n"), 0o644)
	_ = os.WriteFile(filepath.Join(chart, "values.yaml"), []byte("main:\n  name: til-order\n"), 0o644)
	_ = os.WriteFile(filepath.Join(chart, deploy.LocalValuesFile), []byte("main:\n  image: tfc/til-order\n  version: 1.16.2-SNAPSHOT\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "order-service", "Dockerfile"), []byte("FROM scratch\n"), 0o644)
	m := shell.NewManager()
	t.Cleanup(m.Shutdown)
	go func() {
		for {
			m.Listen()()
		}
	}()
	return &model.UI{
		Config: config.Config{ShortCuts: action.DefaultShortcuts, Projects: []config.Project{
			{ID: "til/til-order", Name: "til-order", Path: dir, Fetched: true, Deployable: true},
			{ID: "til/til-report", Name: "til-report", Path: filepath.Join(root, "til-report"), Fetched: true, Deployable: true},
		}},
		Shells: m, MaxParallelism: 2, WindowWidth: 140, WindowHeight: 40,
	}
}

func TestDeploy_StartsTheClusterThenDeploys(t *testing.T) {
	log := fakeCluster(t, false, "https://127.0.0.1:53846")
	ui := deployTestUI(t)

	OpenDeploy(ui)
	ClusterChecked(ui, CheckCluster(ui)().(model.DeployPreflightMsg))
	out := ansi.Strip(component.RenderDeployModal(ui))
	if ui.ClusterRunning || !strings.Contains(out, "rancher is stopped or still booting - press s") || !strings.Contains(out, "tfc/til-order:1.16.2-SNAPSHOT") ||
		!strings.Contains(out, "Skipped (no deployable chart): til-report") {
		t.Fatalf("expected a stopped cluster, the plan and the skip:\n%s", out)
	}
	if DeployModal(ui, testKeyMsg("enter")) != nil || !strings.Contains(ui.StatusMsg, "isn't ready") {
		t.Fatal("deploying needs a running cluster")
	}

	DeployModal(ui, testKeyMsg("s"))
	start := ui.Shells.Get(ui.ClusterStartID)
	if start == nil || start.Tag != TagCluster {
		t.Fatal("s must start the rancher container as a session")
	}
	<-start.Done()
	ClusterChecked(ui, ClusterStartExited(ui, start.ID)().(model.DeployPreflightMsg))
	if !ui.ClusterRunning || ui.ClusterErr != nil {
		t.Fatalf("after starting the container the cluster must be ready, got %v %v", ui.ClusterRunning, ui.ClusterErr)
	}

	run := DeployModal(ui, testKeyMsg("d"))
	if run == nil || !ui.DeployRunning {
		t.Fatal("d must start the deployment")
	}
	DeployFinished(ui, run().(model.DeployFinishedMsg))
	if ui.DeployErr != nil || !strings.Contains(ui.StatusMsg, "Deployed to rancher-lcl/til-lcl") {
		t.Fatalf("expected success, got %v / %q", ui.DeployErr, ui.StatusMsg)
	}
	calls, _ := os.ReadFile(log)
	if !strings.Contains(string(calls), "docker build -t tfc/til-order:1.16.2-SNAPSHOT") || !strings.Contains(string(calls), "docker save tfc/til-order:1.16.2-SNAPSHOT") ||
		!strings.Contains(string(calls), "docker exec -i rancher /usr/bin/ctr -a /run/k3s/containerd/containerd.sock -n k8s.io images import -") || !strings.Contains(string(calls), "--kube-context rancher-lcl --namespace til-lcl\n") {
		t.Fatalf("unexpected calls:\n%s", calls)
	}
	if out := ansi.Strip(component.RenderDeployModal(ui)); !strings.Contains(out, "image ✅  deploy ✅") {
		t.Fatalf("expected both steps done:\n%s", out)
	}
}

func TestDeploy_RefusesNonLocalServer(t *testing.T) {
	fakeCluster(t, true, "https://rancher.intern-belgiantrain.be/k8s/clusters/c-dfjlf")
	ui := deployTestUI(t)

	OpenDeploy(ui)
	ClusterChecked(ui, CheckCluster(ui)().(model.DeployPreflightMsg))

	if ui.ClusterErr == nil || !strings.Contains(ansi.Strip(component.RenderDeployModal(ui)), "is a hostname, not a local address") {
		t.Fatalf("the guard must refuse a remote server, got %v", ui.ClusterErr)
	}
	if DeployModal(ui, testKeyMsg("enter")) != nil || DeployModal(ui, testKeyMsg("d")) != nil {
		t.Fatal("no train or deployment may start when the guard refuses")
	}
}

func TestDeploy_ConfiguredProductionContextIsRefused(t *testing.T) {
	fakeCluster(t, true, "https://127.0.0.1:1")
	ui := deployTestUI(t)
	ui.Config.LocalCluster = config.LocalCluster{Context: "azr-prd-idas-aks01"}

	OpenDeploy(ui)
	ClusterChecked(ui, CheckCluster(ui)().(model.DeployPreflightMsg))

	if ui.ClusterErr == nil || !strings.Contains(ui.ClusterErr.Error(), "only local contexts") {
		t.Fatalf("a non-allowlisted context must be refused even on a local address, got %v", ui.ClusterErr)
	}
}

// TestDeploy_TargetsTheTeamEnvironment: with a manifest environment the ship
// screen deploys to go-dark-<team> / its namespace, configured from the
// manifest (values file, database URL), and the legacy config is ignored.
func TestDeploy_TargetsTheTeamEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	log := fakeCluster(t, true, "https://127.0.0.1:6550")
	_ = os.WriteFile(filepath.Join(filepath.Dir(log), "migrated"), nil, 0o644) // no old per-service releases
	ui := deployTestUI(t)
	ui.Config.LocalCluster = config.LocalCluster{Context: "rancher-lcl", Namespace: "legacy"}
	ui.Config.Environments = []config.TeamEnvironment{{Team: "rail", Environment: config.Environment{Application: "til", Domain: "lcl.local",
		ValuesFiles: map[string]string{"dev": "values.Dev.yaml"}}.WithDefaults()}}
	chart := filepath.Join(ui.Config.Projects[0].Path, "helm", "til-order")
	_ = os.WriteFile(filepath.Join(chart, "values.Dev.yaml"), []byte("main:\n  elasticSearchHost: dev-es\n"), 0o644)

	OpenDeploy(ui)
	ClusterChecked(ui, CheckCluster(ui)().(model.DeployPreflightMsg))
	if ui.ClusterErr != nil || !ui.ClusterRunning {
		t.Fatalf("the environment's context must pass the guard: %v", ui.ClusterErr)
	}
	out := ansi.Strip(component.RenderDeployModal(ui))
	if !strings.Contains(out, `environment "lcl" of team rail`) || !strings.Contains(out, "go-dark-rail is running and local (namespace til-lcl)") {
		t.Fatalf("screen:\n%s", out)
	}
	run := DeployModal(ui, testKeyMsg("d"))
	DeployFinished(ui, run().(model.DeployFinishedMsg))
	if ui.DeployErr != nil || !strings.Contains(ui.StatusMsg, "Deployed to go-dark-rail/til-lcl") {
		t.Fatalf("expected success, got %v / %q", ui.DeployErr, ui.StatusMsg)
	}
	calls, _ := os.ReadFile(log)
	if !strings.Contains(string(calls), "docker exec -i go-dark-rail-rancher /usr/bin/ctr") || !strings.Contains(string(calls), "--kube-context go-dark-rail --namespace til-lcl\n") ||
		strings.Contains(string(calls), "legacy") {
		t.Fatalf("unexpected calls:\n%s", calls)
	}
	values, _ := os.ReadFile(filepath.Join(ui.Umbrella().ChartDir(), deploy.UmbrellaValuesFile))
	if !strings.Contains(string(values), "elasticSearchHost: dev-es") {
		t.Fatalf("the manifest's values file must be layered in:\n%s", values)
	}
}

// TestDeploy_UmbrellaToggleMigrateIterate: with a team environment the ship
// screen deploys one umbrella release; an old per-service release has to go
// first (m), services can be switched off (space, remembered in config.json)
// and i rebuilds + restarts one service.
func TestDeploy_UmbrellaToggleMigrateIterate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	log := fakeCluster(t, true, "https://127.0.0.1:6550")
	ui := deployTestUI(t)
	ui.Config.Environments = []config.TeamEnvironment{{Team: "rail", Environment: config.Environment{Application: "til", Domain: "lcl.local"}.WithDefaults()}}
	report := ui.Config.Projects[1]
	report.Path = filepath.Join(filepath.Dir(ui.Config.Projects[0].Path), "til-report")
	chart := filepath.Join(report.Path, "helm", "til-report")
	_ = os.MkdirAll(chart, 0o755)
	_ = os.MkdirAll(filepath.Join(report.Path, "svc"), 0o755)
	_ = os.WriteFile(filepath.Join(chart, "Chart.yaml"), []byte("apiVersion: v2\nname: til-report\nversion: 1.0.0\n"), 0o644)
	_ = os.WriteFile(filepath.Join(chart, "values.yaml"), []byte("main:\n  name: til-report\n"), 0o644)
	_ = os.WriteFile(filepath.Join(chart, deploy.LocalValuesFile), []byte("main:\n  image: local/til-report\n  version: 1.0.0\n"), 0o644)
	_ = os.WriteFile(filepath.Join(report.Path, "svc", "Dockerfile"), []byte("FROM scratch\n"), 0o644)
	ui.Config.Projects[1] = report

	OpenDeploy(ui)
	ClusterChecked(ui, CheckCluster(ui)().(model.DeployPreflightMsg))
	if len(ui.DeployMigrate) != 1 || ui.DeployMigrate[0] != "til-order" {
		t.Fatalf("the old til-order release must be detected, got %v", ui.DeployMigrate)
	}
	// The box wraps long lines: compare on collapsed whitespace.
	out := strings.Join(strings.Fields(strings.ReplaceAll(ansi.Strip(component.RenderDeployModal(ui)), "│", " ")), " ")
	if !strings.Contains(out, "per-service release from before the umbrella (til-order)") || !strings.Contains(out, "[m] Uninstall old releases") ||
		!strings.Contains(out, "[x] til-order") || !strings.Contains(out, `one release "til" (umbrella chart til-umbrella`) {
		t.Fatalf("screen:\n%s", out)
	}
	if DeployModal(ui, testKeyMsg("d")) != nil || !strings.Contains(ui.StatusMsg, "m uninstalls them first") {
		t.Fatal("the umbrella can't install over the old releases")
	}

	migrate := DeployModal(ui, testKeyMsg("m"))
	if migrate == nil || !ui.DeployMigrating {
		t.Fatal("m starts the uninstall")
	}
	recheck := DeployMigrated(ui, migrate().(model.DeployMigratedMsg))
	ClusterChecked(ui, recheck().(model.DeployPreflightMsg))
	calls, _ := os.ReadFile(log)
	if !strings.Contains(string(calls), "helm uninstall til-order --wait --kube-context go-dark-rail --namespace til-lcl") || len(ui.DeployMigrate) != 0 {
		t.Fatalf("expected the uninstall and a clean re-check, got %v:\n%s", ui.DeployMigrate, calls)
	}

	// Switch til-report off: remembered in config.json.
	DeployModal(ui, testKeyMsg("down"))
	DeployModal(ui, testKeyMsg("space"))
	if !ui.ShipEnabled(ui.ShipTargets()[0]) || ui.ShipEnabled(ui.ShipTargets()[1]) {
		t.Fatalf("space switches the highlighted service off, disabled=%v", ui.Config.ShipDisabled)
	}
	if saved, _ := config.LoadConfig(); len(saved.ShipDisabled) != 1 || saved.ShipDisabled[0] != "til/til-report" {
		t.Fatalf("the switch must be saved, got %v", saved.ShipDisabled)
	}
	if out := ansi.Strip(component.RenderDeployModal(ui)); !strings.Contains(out, "[ ] til-report") {
		t.Fatalf("screen:\n%s", out)
	}

	run := DeployModal(ui, testKeyMsg("d"))
	DeployFinished(ui, run().(model.DeployFinishedMsg))
	if ui.DeployErr != nil {
		t.Fatalf("deploy: %v", ui.DeployErr)
	}
	calls, _ = os.ReadFile(log)
	umbrellaDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "go-dark", "env", "rail", "charts", "til-umbrella")
	if !strings.Contains(string(calls), "helm upgrade --install til "+umbrellaDir+" -f "+filepath.Join(umbrellaDir, deploy.UmbrellaValuesFile)) ||
		strings.Contains(string(calls), "docker build -t local/til-report") || !strings.Contains(string(calls), "docker build -t tfc/til-order") {
		t.Fatalf("one umbrella release of the switched-on services:\n%s", calls)
	}
	if st := ui.DeployProgress.Get("til/til-report"); st.Deploy != deploy.StepSkipped {
		t.Fatalf("switched-off service %+v", st)
	}

	DeployModal(ui, testKeyMsg("up"))
	iterate := DeployModal(ui, testKeyMsg("i"))
	DeployFinished(ui, iterate().(model.DeployFinishedMsg))
	calls, _ = os.ReadFile(log)
	if ui.DeployErr != nil || !strings.Contains(string(calls), "kubectl rollout restart deployment/til-order --context go-dark-rail --namespace til-lcl") {
		t.Fatalf("i restarts the service after re-importing its image: %v\n%s", ui.DeployErr, calls)
	}
}
