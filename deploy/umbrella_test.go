//go:build !windows

package deploy_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/exr462/go-dark/deploy"
	"gopkg.in/yaml.v3"
)

// withTemplate gives a fixture chart a template that renders the values the
// umbrella must nest correctly.
func withTemplate(t *testing.T, tgt deploy.Target) {
	t.Helper()
	tpl := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Values.main.name }}\ndata:\n  image: {{ .Values.main.image }}:{{ .Values.main.version }}\n  env: {{ .Values.ypto.env }}\n  replicas: {{ .Values.main.minReplicas | quote }}\n" +
		"  jms: {{ .Values.data.jmsOmsQPwd1 }}\n  props: |\n{{ (.Files.Glob (printf \"config/%s/*\" .Values.main.env)).AsConfig | indent 4 }}\n"
	if err := os.WriteFile(filepath.Join(tgt.ChartDir, "templates", "main.yaml"), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareUmbrella_NestsEverySubchart(t *testing.T) {
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order", true), service(t, root, "til-purchase", false, "til/til-order"))
	order, purchase := waves[0][0], waves[1][0]
	withTemplate(t, order)
	withTemplate(t, purchase)
	u := &deploy.Umbrella{Application: "til", Dir: filepath.Join(t.TempDir(), "charts"), Enabled: func(t deploy.Target) bool { return t.ChartName != "til-purchase" }}

	st, err := deploy.PrepareUmbrella(u, []deploy.Target{order, purchase}, lcl)
	if err != nil {
		t.Fatal(err)
	}
	if st.Dir != filepath.Join(u.Dir, "til-umbrella") || len(st.Values) != 1 || filepath.Base(st.Values[0]) != deploy.UmbrellaValuesFile || st.HasDeps {
		t.Fatalf("staged %+v", st)
	}
	read := func(path string) map[string]any {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		return m
	}
	chart := read(filepath.Join(st.Dir, "Chart.yaml"))
	deps := chart["dependencies"].([]any)
	if chart["name"] != "til-umbrella" || len(deps) != 2 {
		t.Fatalf("Chart.yaml %v", chart)
	}
	if d := deps[0].(map[string]any); d["name"] != "til-order" || d["version"] != "1.2.0-SNAPSHOT" || d["condition"] != "til-order.enabled" {
		t.Fatalf("dependency %v", d)
	}
	for _, sub := range []string{"til-order", "til-purchase"} {
		if _, err := os.Stat(filepath.Join(st.Dir, "charts", sub, "config", "lcl", "application-k8s.properties")); err != nil {
			t.Errorf("subchart %s must be staged with config/lcl: %v", sub, err)
		}
	}
	values := read(st.Values[0])
	o := values["til-order"].(map[string]any)
	p := values["til-purchase"].(map[string]any)
	if o["enabled"] != true || p["enabled"] != false {
		t.Fatalf("enabled switches %v %v", o["enabled"], p["enabled"])
	}
	om := o["main"].(map[string]any)
	if om["image"] != "tfc/til-order" || om["version"] != "1.2.0-SNAPSHOT" || om["env"] != "lcl" || om["minReplicas"] != 1 || om["dynatrace"] != false || om["imagePullPolicy"] != "Never" {
		t.Fatalf("til-order main must be dev + local + overlay: %v", om)
	}
	if o["data"].(map[string]any)["jmsOmsQPwd1"] != "my-local-jms" || o["data"].(map[string]any)["applicationDbPwd"] != lcl.DBPassword {
		t.Fatalf("til-order data %v", o["data"])
	}
	if p["main"].(map[string]any)["image"] != "local/til-purchase" || p["ypto"].(map[string]any)["env"] != "lcl" {
		t.Fatalf("til-purchase values %v", p)
	}
	if _, top := values["main"]; top {
		t.Fatal("subchart values must not leak to the top level")
	}

	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm not installed; the generated chart isn't rendered")
	}
	out, err := exec.Command(helm, "template", "til", st.Dir, "-f", st.Values[0]).CombinedOutput()
	if err != nil {
		t.Fatalf("helm template: %v\n%s", err, out)
	}
	rendered := string(out)
	for _, want := range []string{"image: tfc/til-order:1.2.0-SNAPSHOT", "env: lcl", "replicas: \"1\"", "jms: my-local-jms", "spring.datasource.url=" + lcl.DBURL} {
		if !strings.Contains(rendered, want) {
			t.Errorf("helm must render %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "name: til-purchase") || strings.Contains(rendered, "local/til-purchase") {
		t.Errorf("a disabled subchart renders nothing:\n%s", rendered)
	}
}

func TestRun_UmbrellaIsOneRelease(t *testing.T) {
	log := fakeTools(t)
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order", true), service(t, root, "til-purchase", false, "til/til-order"), service(t, root, "til-dtm", false))
	progress := deploy.NewProgress()
	u := &deploy.Umbrella{Application: "til", Dir: filepath.Join(t.TempDir(), "charts"), Enabled: func(t deploy.Target) bool { return t.ChartName != "til-dtm" }}

	err := deploy.Run(context.Background(), waves, deploy.Options{Sessions: manager(t), Cluster: local, Env: lcl, Parallel: 2, Progress: progress, Umbrella: u})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	calls := strings.Split(strings.TrimSpace(string(data)), "\n")
	var helm, builds []string
	for _, c := range calls {
		switch {
		case strings.HasPrefix(c, "helm"):
			helm = append(helm, c)
		case strings.HasPrefix(c, "docker build"):
			builds = append(builds, c)
		}
	}
	if len(helm) != 1 || !strings.HasPrefix(helm[0], "helm upgrade --install til "+u.ChartDir()+" -f "+filepath.Join(u.ChartDir(), deploy.UmbrellaValuesFile)) ||
		!strings.Contains(helm[0], "--kube-context rancher-lcl --namespace til-lcl") || !strings.Contains(helm[0], "--wait") {
		t.Fatalf("expected one umbrella release, got %q", helm)
	}
	if len(builds) != 2 || strings.Contains(string(data), "til-dtm") {
		t.Fatalf("only enabled services are built:\n%s", data)
	}
	if s := progress.Get("til/til-dtm"); s.Image != deploy.StepSkipped || s.Deploy != deploy.StepSkipped {
		t.Fatalf("disabled target %+v", s)
	}
	if s := progress.Get("til/til-purchase"); s.Image != deploy.StepDone || s.Deploy != deploy.StepDone || s.SessionID == 0 {
		t.Fatalf("enabled target %+v", s)
	}
	if _, err := os.Stat(filepath.Join(u.ChartDir(), "charts", "til-dtm", "Chart.yaml")); err != nil {
		t.Fatal("disabled services stay in the chart, switched off")
	}
}

func TestRun_UmbrellaFailureMarksEveryService(t *testing.T) {
	fakeTools(t, "til")
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order", true), service(t, root, "til-purchase", false))
	progress := deploy.NewProgress()
	u := &deploy.Umbrella{Application: "til", Dir: filepath.Join(t.TempDir(), "charts")}
	err := deploy.Run(context.Background(), waves, deploy.Options{Sessions: manager(t), Cluster: local, Env: lcl, Progress: progress, Umbrella: u, SkipImages: true})
	if err == nil || !strings.Contains(err.Error(), "til: helm exited with 1") {
		t.Fatalf("expected the helm failure, got %v", err)
	}
	for _, id := range []string{"til/til-order", "til/til-purchase"} {
		if s := progress.Get(id); s.Deploy != deploy.StepFailed || s.Image != deploy.StepDone {
			t.Fatalf("%s: %+v", id, s)
		}
	}
	if err := deploy.Run(context.Background(), waves, deploy.Options{Cluster: local, Progress: deploy.NewProgress(), Umbrella: &deploy.Umbrella{Application: "til", Dir: t.TempDir(), Enabled: func(deploy.Target) bool { return false }}}); err == nil {
		t.Fatal("everything switched off is an error")
	}
}

func TestPerServiceReleasesAndIterate(t *testing.T) {
	log := fakeTools(t)
	root := t.TempDir()
	waves, _ := planFor(t, service(t, root, "til-order", true), service(t, root, "til-purchase", false))
	targets := waves[0]

	names, err := deploy.PerServiceReleases(local, targets, "til")
	if err != nil || strings.Join(names, ",") != "til-order" {
		t.Fatalf("expected the old til-order release only, got %v %v", names, err)
	}
	if _, err := deploy.PerServiceReleases(kubeProd, targets, "til"); err == nil {
		t.Fatal("listing goes through the guard")
	}
	if argv := strings.Join(deploy.UninstallArgv(local, "til-order"), " "); argv != "helm uninstall til-order --wait --kube-context rancher-lcl --namespace til-lcl" {
		t.Fatalf("uninstall argv %q", argv)
	}

	progress := deploy.NewProgress()
	if err := deploy.Iterate(context.Background(), targets[0], deploy.Options{Sessions: manager(t), Cluster: local, Progress: progress}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	for _, want := range []string{"docker build -t tfc/til-order:1.2.0-SNAPSHOT", "docker save tfc/til-order:1.2.0-SNAPSHOT",
		"kubectl rollout restart deployment/til-order --context rancher-lcl --namespace til-lcl", "kubectl rollout status deployment/til-order --timeout=11m --context rancher-lcl --namespace til-lcl"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("iterate must run %q:\n%s", want, data)
		}
	}
	if strings.Contains(string(data), "helm upgrade") {
		t.Error("iterate doesn't helm upgrade")
	}
	if s := progress.Get("til/til-order"); s.Image != deploy.StepDone || s.Deploy != deploy.StepDone {
		t.Fatalf("progress %+v", s)
	}
}
