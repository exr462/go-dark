//go:build !windows

package deploy_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/kube"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
)

// fakeTools puts docker/helm/kubectl/minikube scripts on PATH that append
// their arguments to log. helm fails for any chart named in failCharts.
func fakeTools(t *testing.T, failCharts ...string) (log string) {
	t.Helper()
	bin := t.TempDir()
	log = filepath.Join(t.TempDir(), "calls.log")
	fail := strings.Join(failCharts, " ")
	scripts := map[string]string{
		"docker":   `echo "docker $*" >> ` + log + `; echo "Successfully tagged"`,
		"kubectl":  `echo "https://127.0.0.1:53846"`,
		"minikube": `echo "192.168.49.2"`,
		"helm": `echo "helm $*" >> ` + log + `
for c in ` + fail + `; do [ "$3" = "$c" ] && { echo "Error: release $c failed"; exit 1; }; done
echo "STATUS: deployed"`,
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func service(t *testing.T, root, name string, deps ...string) config.Project {
	t.Helper()
	dir := filepath.Join(root, name)
	chart := filepath.Join(dir, "helm", name)
	_ = os.MkdirAll(filepath.Join(chart, "templates"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, name+"-service"), 0o755)
	files := map[string]string{
		filepath.Join(chart, "Chart.yaml"):                "apiVersion: v2\nname: " + name + "\nversion: {JENKINS_VERSION}\n",
		filepath.Join(chart, "values.yaml"):               "main:\n  replicas: 2\n",
		filepath.Join(chart, deploy.LocalValuesFile):      "main:\n  image: tfc/" + name + "\n  version: 1.2.0-SNAPSHOT\n  imagePullPolicy: Never\n",
		filepath.Join(chart, "templates", "main.yaml"):    "kind: ConfigMap\n",
		filepath.Join(dir, name+"-service", "Dockerfile"): "FROM scratch\n",
		filepath.Join(dir, "target", "Dockerfile"):        "FROM ignored\n",
	}
	for path, content := range files {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return config.Project{ID: "til/" + name, Name: name, Path: dir, Fetched: true, Deployable: true, DependsOn: deps}
}

func planFor(t *testing.T, projects ...config.Project) ([][]deploy.Target, []deploy.Target) {
	t.Helper()
	_, projectWaves, err := model.ResolveDeployOrder(projects)
	if err != nil {
		t.Fatal(err)
	}
	return deploy.Plan(projectWaves)
}

func manager(t *testing.T) *shell.Manager {
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

func TestPlan_DiscoversTargetsWavesAndSkips(t *testing.T) {
	root := t.TempDir()
	order := service(t, root, "til-order")
	purchase := service(t, root, "til-purchase", "til/til-order")
	plain := config.Project{ID: "til/til-global-aks", Name: "til-global-aks", Path: filepath.Join(root, "aks"), Fetched: true, Deployable: true}
	_ = os.MkdirAll(plain.Path, 0o755)

	waves, skipped := planFor(t, order, purchase, plain)
	if len(waves) != 2 || waves[0][0].Project.ID != "til/til-order" || waves[1][0].Project.ID != "til/til-purchase" {
		t.Fatalf("expected order then purchase, got %+v", waves)
	}
	o := waves[0][0]
	if o.Image != "tfc/til-order" || o.Tag != "1.2.0-SNAPSHOT" || o.ChartName != "til-order" || filepath.Base(o.DockerDir) != "til-order-service" {
		t.Fatalf("unexpected target %+v", o)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0].Skip, deploy.LocalValuesFile) {
		t.Fatalf("a project without local values must be skipped, got %+v", skipped)
	}
}

func TestPrepareChart_ValidVersionInCopyOnly(t *testing.T) {
	waves, _ := planFor(t, service(t, t.TempDir(), "til-order"))
	tgt := waves[0][0]
	dir, hasDeps, err := deploy.PrepareChart(tgt, t.TempDir())
	if err != nil || hasDeps {
		t.Fatalf("prepare: %v deps=%v", err, hasDeps)
	}
	copied, _ := os.ReadFile(filepath.Join(dir, "Chart.yaml"))
	original, _ := os.ReadFile(filepath.Join(tgt.ChartDir, "Chart.yaml"))
	if !strings.Contains(string(copied), "version: "+deploy.LocalChartVersion) || !strings.Contains(string(original), "{JENKINS_VERSION}") {
		t.Fatalf("the copy gets a valid version, the original stays:\n%s\n---\n%s", copied, original)
	}
	if _, err := os.Stat(filepath.Join(dir, "templates", "main.yaml")); err != nil {
		t.Fatal("templates must be copied")
	}
}

func TestRun_BuildsImagesThenInstallsChartsWithExplicitContext(t *testing.T) {
	log := fakeTools(t)
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order"), service(t, root, "til-purchase", "til/til-order"))
	progress := deploy.NewProgress()

	err := deploy.Run(context.Background(), waves, deploy.Options{
		Sessions: manager(t), Cluster: kube.Cluster{Context: "minikube", Namespace: "til-local"},
		DockerEnv: []string{"DOCKER_HOST=tcp://127.0.0.1:1"}, Parallel: 2, Progress: progress,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	calls := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(calls) != 4 {
		t.Fatalf("expected 4 calls, got:\n%s", data)
	}
	if !strings.HasPrefix(calls[0], "docker build -t tfc/til-order:1.2.0-SNAPSHOT ") || !strings.HasPrefix(calls[2], "docker build -t tfc/til-purchase:") {
		t.Fatalf("images must be built per wave, got:\n%s", data)
	}
	for _, c := range []string{calls[1], calls[3]} {
		if !strings.HasPrefix(c, "helm upgrade --install til-") || !strings.Contains(c, "--kube-context minikube --namespace til-local") ||
			!strings.Contains(c, deploy.LocalValuesFile) || !strings.Contains(c, "--wait") {
			t.Fatalf("helm must target the local cluster explicitly, got %q", c)
		}
	}
	if s := progress.Get("til/til-purchase"); s.Image != deploy.StepDone || s.Deploy != deploy.StepDone || s.SessionID == 0 {
		t.Fatalf("unexpected progress %+v", s)
	}
}

func TestRun_StopsAtFirstFailure(t *testing.T) {
	log := fakeTools(t, "til-order")
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order"), service(t, root, "til-purchase", "til/til-order"))
	progress := deploy.NewProgress()
	m := manager(t)

	err := deploy.Run(context.Background(), waves, deploy.Options{Sessions: m, Cluster: kube.Cluster{Context: "minikube", Namespace: "til-local"}, Parallel: 1, Progress: progress})

	if err == nil || !strings.Contains(err.Error(), "til-order: deploy: helm exited with 1") {
		t.Fatalf("expected the helm failure, got %v", err)
	}
	if s := progress.Get("til/til-order"); s.Deploy != deploy.StepFailed {
		t.Fatalf("order must be failed, got %+v", s)
	}
	if s := progress.Get("til/til-purchase"); s.Image != deploy.StepCancelled || s.Deploy != deploy.StepCancelled {
		t.Fatalf("the next wave must not run, got %+v", s)
	}
	data, _ := os.ReadFile(log)
	if strings.Contains(string(data), "til-purchase") {
		t.Fatalf("nothing of the next wave may run:\n%s", data)
	}
	failed := m.Get(progress.Get("til/til-order").SessionID)
	if failed == nil || !strings.Contains(failed.Text(), "release til-order failed") {
		t.Fatal("the failing step's log must stay inspectable")
	}
}

func TestRun_RefusesNonLocalCluster(t *testing.T) {
	fakeTools(t)
	err := deploy.Run(context.Background(), nil, deploy.Options{Cluster: kube.Cluster{Context: "azr-prd-idas-aks01", Namespace: "til-local"}, Progress: deploy.NewProgress()})
	if err == nil || !strings.Contains(err.Error(), "only local contexts") {
		t.Fatalf("expected the guard to refuse, got %v", err)
	}
}

func TestRun_Cancel(t *testing.T) {
	fakeTools(t)
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := deploy.Run(ctx, waves, deploy.Options{Sessions: manager(t), Cluster: kube.Cluster{Context: "minikube", Namespace: "x"}, Progress: deploy.NewProgress()})
	if err == nil {
		t.Fatal("a cancelled run must report it")
	}
}
