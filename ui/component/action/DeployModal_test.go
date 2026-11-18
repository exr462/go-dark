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
	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/ui/component"
)

// fakeCluster puts minikube/kubectl/docker/helm scripts on PATH. minikube
// reports the given host state; `minikube start` flips it to Running.
func fakeCluster(t *testing.T, running bool, server string) string {
	t.Helper()
	bin := t.TempDir()
	state := filepath.Join(bin, "state")
	log := filepath.Join(bin, "calls.log")
	host := "Stopped"
	if running {
		host = "Running"
	}
	_ = os.WriteFile(state, []byte(host), 0o644)
	scripts := map[string]string{
		"minikube": `case "$1" in
  status) s=$(cat ` + state + `); echo "{\"Host\":\"$s\",\"APIServer\":\"$s\"}"; [ "$s" = Running ] || exit 7;;
  start) echo "Done! kubectl is now configured"; echo Running > ` + state + `;;
  docker-env) echo DOCKER_HOST=tcp://127.0.0.1:1;;
  ip) echo 192.168.49.2;;
esac`,
		"kubectl": `echo "` + server + `"`,
		"docker":  `echo "docker $*" >> ` + log,
		"helm":    `echo "helm $*" >> ` + log,
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

func TestDeploy_StartsMinikubeThenDeploys(t *testing.T) {
	log := fakeCluster(t, false, "https://127.0.0.1:53846")
	ui := deployTestUI(t)

	ClusterChecked(ui, OpenDeploy(ui)().(model.DeployPreflightMsg))
	out := ansi.Strip(component.RenderDeployModal(ui))
	if ui.ClusterRunning || !strings.Contains(out, "minikube is stopped - press s") || !strings.Contains(out, "tfc/til-order:1.16.2-SNAPSHOT") ||
		!strings.Contains(out, "Skipped (no local chart values): til-report") {
		t.Fatalf("expected a stopped cluster, the plan and the skip:\n%s", out)
	}
	if DeployModal(ui, testKeyMsg("enter")) != nil || !strings.Contains(ui.StatusMsg, "isn't ready") {
		t.Fatal("deploying needs a running cluster")
	}

	DeployModal(ui, testKeyMsg("s"))
	start := ui.Shells.Get(ui.MinikubeStartID)
	if start == nil || start.Tag != TagMinikube {
		t.Fatal("s must start minikube as a session")
	}
	<-start.Done()
	ClusterChecked(ui, MinikubeExited(ui, start.ID)().(model.DeployPreflightMsg))
	if !ui.ClusterRunning || ui.ClusterErr != nil {
		t.Fatalf("after minikube start the cluster must be ready, got %v %v", ui.ClusterRunning, ui.ClusterErr)
	}

	run := DeployModal(ui, testKeyMsg("enter"))
	if run == nil || !ui.DeployRunning {
		t.Fatal("enter must start the deployment")
	}
	DeployFinished(ui, run().(model.DeployFinishedMsg))
	if ui.DeployErr != nil || !strings.Contains(ui.StatusMsg, "Deployed to minikube/til-local") {
		t.Fatalf("expected success, got %v / %q", ui.DeployErr, ui.StatusMsg)
	}
	calls, _ := os.ReadFile(log)
	if !strings.Contains(string(calls), "docker build -t tfc/til-order:1.16.2-SNAPSHOT") || !strings.Contains(string(calls), "--kube-context minikube --namespace til-local") {
		t.Fatalf("unexpected calls:\n%s", calls)
	}
	if out := ansi.Strip(component.RenderDeployModal(ui)); !strings.Contains(out, "image ✅  deploy ✅") {
		t.Fatalf("expected both steps done:\n%s", out)
	}
}

func TestDeploy_RefusesNonLocalServer(t *testing.T) {
	fakeCluster(t, true, "https://rancher.intern-belgiantrain.be/k8s/clusters/c-dfjlf")
	ui := deployTestUI(t)

	ClusterChecked(ui, OpenDeploy(ui)().(model.DeployPreflightMsg))

	if ui.ClusterErr == nil || !strings.Contains(ansi.Strip(component.RenderDeployModal(ui)), "is a hostname, not a local address") {
		t.Fatalf("the guard must refuse a remote server, got %v", ui.ClusterErr)
	}
	if DeployModal(ui, testKeyMsg("enter")) != nil {
		t.Fatal("no deployment may start when the guard refuses")
	}
}

func TestDeploy_ConfiguredProductionContextIsRefused(t *testing.T) {
	fakeCluster(t, true, "https://127.0.0.1:1")
	ui := deployTestUI(t)
	ui.Config.LocalCluster = config.LocalCluster{Context: "azr-prd-idas-aks01"}

	ClusterChecked(ui, OpenDeploy(ui)().(model.DeployPreflightMsg))

	if ui.ClusterErr == nil || !strings.Contains(ui.ClusterErr.Error(), "only local contexts") {
		t.Fatalf("a non-allowlisted context must be refused even on a local address, got %v", ui.ClusterErr)
	}
}
