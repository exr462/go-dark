//go:build !windows

package deploy_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/env"
	"github.com/exr462/go-dark/kube"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"gopkg.in/yaml.v3"
)

// TestMain lets the image side-load re-run this binary as `go-dark env
// import-image` (kube.ImportImageArgv), exactly like the real program.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "env" {
		os.Exit(env.CLI(os.Args[2:], os.Stdout))
	}
	os.Exit(m.Run())
}

// fakeTools puts docker/helm/kubectl scripts on PATH that append their
// arguments to log. helm fails for any chart named in failCharts.
func fakeTools(t *testing.T, failCharts ...string) (log string) {
	t.Helper()
	bin := t.TempDir()
	log = filepath.Join(t.TempDir(), "calls.log")
	fail := strings.Join(failCharts, " ")
	scripts := map[string]string{
		"docker": `echo "docker $*" >> ` + log + `
case "$1" in
  save) echo "image-tar" ;;
  exec) cat > /dev/null; echo "unpacking $*" ;;
  *) echo "Successfully tagged" ;;
esac`,
		"kubectl": `case "$1" in rollout) echo "kubectl $*" >> ` + log + `;; *) echo "https://127.0.0.1:6550";; esac`,
		"helm": `echo "helm $*" >> ` + log + `
[ "$1" = list ] && { echo '[{"name":"til-order","status":"deployed"},{"name":"til","status":"deployed"},{"name":"other","status":"failed"}]'; exit 0; }
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

var local = kube.Cluster{Context: "rancher-lcl", Namespace: "til-lcl", Container: "rancher"}

var kubeProd = kube.Cluster{Context: "azr-prd-idas-aks01", Namespace: "til-lcl"}

var lcl = deploy.Env{Name: "lcl", ConfigFrom: "dev", DBURL: "jdbc:oracle:thin:@oracle.til-lcl.svc:1521/FREEPDB1", DBPassword: "changeit_local", Arch: "arm64"}

// service writes a project shaped like the TIL services: a ypto chart with
// per-environment values, config and secrets, a pom and a Dockerfile.
// withLocal adds the developer's values.local-rancher.yaml.
func service(t *testing.T, root, name string, withLocal bool, deps ...string) config.Project {
	t.Helper()
	dir := filepath.Join(root, name)
	chart := filepath.Join(dir, "helm", name)
	files := map[string]string{
		filepath.Join(chart, "Chart.yaml"):              "apiVersion: v2\nname: " + name + "\nversion: {JENKINS_VERSION}\n",
		filepath.Join(chart, "values.yaml"):             "ypto:\n  env: dev\nmain:\n  name: " + name + "\n  env: dev\n  image: azpswenmbsacr01.azurecr.io/bict4058/" + name + "\n  version: {JENKINS_VERSION}\n",
		filepath.Join(chart, "values.Development.yaml"): "main:\n  minReplicas: 2\n  dynatrace: true\ndata:\n  applicationDbPwd: \"???????\"\n  jmsOmsQPwd1: \"???????\"\n  keyStorePwd: \"???????\"\n",
		filepath.Join(chart, "values.Production.yaml"):  "main:\n  minReplicas: 4\n",
		filepath.Join(chart, "config", "dev", "application-k8s.properties"): "spring.datasource.url=jdbc:oracle:thin:@db.til.dev-intern-belgiantrain.be:1525/CDBNPD03\n" +
			"spring.datasource.username=TORA\nspring.liquibase.url=${spring.datasource.url}\nbelgiantrain.jms.connectionFactories[0].connection-string=Endpoint=sb://dev\n",
		filepath.Join(chart, "config", "prd", "application-k8s.properties"): "spring.datasource.url=jdbc:oracle:thin:@db.prd\n",
		filepath.Join(chart, "secrets", "dev", "client-certificate.pfx"):    "dev-cert",
		filepath.Join(chart, "templates", "main.yaml"):                      "kind: ConfigMap\n",
		filepath.Join(dir, "pom.xml"):                                       "<project><groupId>be.til</groupId><artifactId>" + name + "</artifactId><version>1.16.2-SNAPSHOT</version></project>",
		filepath.Join(dir, name+"-service", "Dockerfile"):                   "FROM scratch\n",
		filepath.Join(dir, "target", "Dockerfile"):                          "FROM ignored\n",
	}
	if withLocal {
		files[filepath.Join(chart, deploy.LocalValuesFile)] = "ypto:\n  env: dev\nmain:\n  image: tfc/" + name + "\n  version: 1.2.0-SNAPSHOT\ndata:\n  jmsOmsQPwd1: my-local-jms\n"
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
	order := service(t, root, "til-order", true)
	purchase := service(t, root, "til-purchase", false, "til/til-order")
	plain := config.Project{ID: "til/til-global-aks", Name: "til-global-aks", Path: filepath.Join(root, "aks"), Fetched: true, Deployable: true}
	_ = os.MkdirAll(plain.Path, 0o755)

	waves, skipped := planFor(t, order, purchase, plain)
	if len(waves) != 2 || waves[0][0].Project.ID != "til/til-order" || waves[1][0].Project.ID != "til/til-purchase" {
		t.Fatalf("expected order then purchase, got %+v", waves)
	}
	o := waves[0][0]
	if o.Ref() != "tfc/til-order:1.2.0-SNAPSHOT" || o.ChartName != "til-order" || filepath.Base(o.DockerDir) != "til-order-service" {
		t.Fatalf("the developer's local values name the image: %+v", o)
	}
	if p := waves[1][0]; p.Ref() != "local/til-purchase:1.16.2-SNAPSHOT" || p.LocalValues != "" {
		t.Fatalf("without local values the image is local/<chart>:<pom version>, got %+v", p)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0].Skip, "no helm") {
		t.Fatalf("a project without a chart must be skipped, got %+v", skipped)
	}
}

func TestPrepareChart_StagesTheEnvironmentLikeDev(t *testing.T) {
	waves, _ := planFor(t, service(t, t.TempDir(), "til-order", true))
	tgt := waves[0][0]
	st, err := deploy.PrepareChart(tgt, t.TempDir(), lcl)
	if err != nil || st.HasDeps {
		t.Fatalf("prepare: %v deps=%v", err, st.HasDeps)
	}
	read := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	if !strings.Contains(read(filepath.Join(st.Dir, "Chart.yaml")), "version: 1.2.0-SNAPSHOT") ||
		!strings.Contains(read(filepath.Join(tgt.ChartDir, "Chart.yaml")), "{JENKINS_VERSION}") {
		t.Fatal("the copy gets a valid version, the original stays")
	}

	props := read(filepath.Join(st.Dir, "config", "lcl", "application-k8s.properties"))
	for _, want := range []string{"spring.datasource.url=" + lcl.DBURL + "\n", "spring.datasource.username=TORA", "spring.liquibase.url=${spring.datasource.url}", "Endpoint=sb://dev"} {
		if !strings.Contains(props, want) {
			t.Errorf("config/lcl must be config/dev with the local database, missing %q in:\n%s", want, props)
		}
	}
	if read(filepath.Join(st.Dir, "secrets", "lcl", "client-certificate.pfx")) != "dev-cert" {
		t.Error("secrets/lcl must be a copy of secrets/dev")
	}
	if strings.Contains(read(filepath.Join(tgt.ChartDir, "config", "dev", "application-k8s.properties")), "oracle.til-lcl") {
		t.Error("the project's config must stay untouched")
	}

	var names []string
	for _, f := range st.Values {
		names = append(names, filepath.Base(f))
	}
	if strings.Join(names, ",") != "values.Development.yaml,"+deploy.LocalValuesFile+","+deploy.OverlayFile {
		t.Fatalf("values must layer dev, then the developer's, then go-dark's, got %v", names)
	}

	var overlay map[string]any
	if err := yaml.Unmarshal([]byte(read(filepath.Join(st.Dir, deploy.OverlayFile))), &overlay); err != nil {
		t.Fatal(err)
	}
	main := overlay["main"].(map[string]any)
	data := overlay["data"].(map[string]any)
	if overlay["ypto"].(map[string]any)["env"] != "lcl" || main["env"] != "lcl" || main["image"] != "tfc/til-order" ||
		main["version"] != "1.2.0-SNAPSHOT" || main["imagePullPolicy"] != "Never" || main["arch"] != "arm64" ||
		main["dynatrace"] != false || main["minReplicas"] != 1 {
		t.Fatalf("unexpected overlay main: %v", main)
	}
	if data["applicationDbPwd"] != "changeit_local" || data["jmsOmsQPwd1"] != "my-local-jms" || data["keyStorePwd"] != "changeit-local" {
		t.Fatalf("db secrets get the local password, the developer's values win elsewhere, placeholders get a dummy: %v", data)
	}
}

func TestChartVersion_ServiceVersionOrFallback(t *testing.T) {
	for tag, want := range map[string]string{"1.16.2-SNAPSHOT": "1.16.2-SNAPSHOT", "2.0.0": "2.0.0", "latest": deploy.LocalChartVersion, "4.0.0.ALPHA": deploy.LocalChartVersion} {
		if got := deploy.ChartVersion(deploy.Target{Tag: tag}); got != want {
			t.Errorf("ChartVersion(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestRun_BuildsLoadsAndInstallsWithExplicitContext(t *testing.T) {
	log := fakeTools(t)
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order", true), service(t, root, "til-purchase", false, "til/til-order"))
	progress := deploy.NewProgress()

	err := deploy.Run(context.Background(), waves, deploy.Options{
		Sessions: manager(t), Cluster: local, Env: lcl, Parallel: 2, Progress: progress,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	calls := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(calls) != 8 {
		t.Fatalf("expected build, save, import, helm per service, got:\n%s", data)
	}
	for i, svc := range []string{"tfc/til-order:1.2.0-SNAPSHOT", "local/til-purchase:1.16.2-SNAPSHOT"} {
		c := calls[i*4:]
		// save and import run as one pipe, so either may log first.
		load := []string{c[1], c[2]}
		slices.Sort(load)
		if !strings.HasPrefix(c[0], "docker build -t "+svc+" ") || load[0] != "docker exec -i rancher /usr/bin/ctr -a /run/k3s/containerd/containerd.sock -n k8s.io images import -" ||
			load[1] != "docker save "+svc {
			t.Fatalf("%s must be built then loaded into the cluster, got:\n%s", svc, data)
		}
		if !strings.HasPrefix(c[3], "helm upgrade --install til-") || !strings.Contains(c[3], "--kube-context rancher-lcl --namespace til-lcl") ||
			!strings.Contains(c[3], deploy.OverlayFile) || !strings.Contains(c[3], "--wait") {
			t.Fatalf("helm must target the local cluster explicitly, got %q", c[3])
		}
	}
	if s := progress.Get("til/til-purchase"); s.Image != deploy.StepDone || s.Deploy != deploy.StepDone || s.SessionID == 0 {
		t.Fatalf("unexpected progress %+v", s)
	}
}

func TestRun_StopsAtFirstFailure(t *testing.T) {
	log := fakeTools(t, "til-order")
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order", true), service(t, root, "til-purchase", true, "til/til-order"))
	progress := deploy.NewProgress()
	m := manager(t)

	err := deploy.Run(context.Background(), waves, deploy.Options{Sessions: m, Cluster: local, Env: lcl, Parallel: 1, Progress: progress})

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
	waves, _ := planFor(t, service(t, root, "til-order", true))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := deploy.Run(ctx, waves, deploy.Options{Sessions: manager(t), Cluster: kube.Cluster{Context: "rancher-lcl", Namespace: "x", Container: "rancher"}, Progress: deploy.NewProgress()})
	if err == nil {
		t.Fatal("a cancelled run must report it")
	}
}
