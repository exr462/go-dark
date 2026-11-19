package env_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/env"
	"gopkg.in/yaml.v3"
)

func til(t *testing.T) env.Env {
	t.Helper()
	path := filepath.Join("..", "examples", "belgiantrain", config.ManifestFileName)
	m, err := config.LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	return env.New(config.TeamEnvironment{Team: m.Team, Environment: *m.Environment}, "")
}

func TestEnv_NamesCarryTheTeam(t *testing.T) {
	e := til(t)
	if e.Project() != "go-dark-belgiantrain" || e.Container() != "go-dark-belgiantrain-rancher" || e.Context() != "go-dark-belgiantrain" ||
		e.IngressClass() != "nginx-lcl" || e.NodeLabel() != "belgiantrain.be/env-lcl=selected" || e.APIURL() != "https://api.til.lcl-intern-belgiantrain.be" ||
		e.DBURL() != "jdbc:oracle:thin:@oracle.til-lcl.svc:1521/FREEPDB1" || e.DBPassword != env.DefaultDBPass ||
		e.HostsLine() != "127.0.0.1 api.til.lcl-intern-belgiantrain.be rancher.lcl-intern-belgiantrain.be" {
		t.Fatalf("unexpected names: %s %s %s %s %s %s %s", e.Project(), e.Container(), e.Context(), e.IngressClass(), e.NodeLabel(), e.APIURL(), e.DBURL())
	}
	dir, err := e.InstallDir()
	if err != nil || !strings.HasSuffix(filepath.ToSlash(dir), "/.local/share/go-dark/env/belgiantrain") {
		t.Fatalf("install dir %q %v", dir, err)
	}
}

// yamlDocs parses every document of a manifest; the generated files must at
// least be valid YAML.
func yamlDocs(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var docs []map[string]any
	for _, doc := range strings.Split(string(data), "\n---\n") {
		var m map[string]any
		if err := yaml.Unmarshal([]byte(doc), &m); err != nil {
			t.Fatalf("invalid YAML: %v\n%s", err, doc)
		}
		docs = append(docs, m)
	}
	return docs
}

func TestFiles_GeneratesTheInstallDirectory(t *testing.T) {
	e := til(t)
	files, err := env.Files(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{env.ComposeFile, env.K3sConfigFile, env.ReadmeFile, "certs/isrg-root-yr.pem", "certs/05e55783.0", "certs/558fe057.0",
		"platform/00-namespace.yaml", "platform/10-nginx-ingress.yaml", "platform/20-reloader.yaml", "platform/30-api-master.yaml", "platform/40-oracle.yaml"} {
		if _, ok := files[want]; !ok {
			t.Errorf("missing %s", want)
		}
	}

	compose := yamlDocs(t, files[env.ComposeFile])[0]
	svc := compose["services"].(map[string]any)["rancher"].(map[string]any)
	ports := svc["ports"].([]any)
	if compose["name"] != "go-dark-belgiantrain" || svc["container_name"] != "go-dark-belgiantrain-rancher" || svc["privileged"] != true ||
		ports[0] != "80:30080" || ports[1] != "443:30443" || ports[2] != "127.0.0.1:6550:6443" {
		t.Fatalf("unexpected compose service: %v", svc)
	}
	text := string(files[env.ComposeFile])
	for _, want := range []string{"k3s:/var/lib/rancher", "kubelet:/var/lib/kubelet", "cni:/var/lib/cni", "log:/var/log", "./k3s/config.yaml:/etc/rancher/k3s/config.yaml:ro", "certs/05e55783.0:/etc/rancher/ssl/05e55783.0"} {
		if !strings.Contains(text, want) {
			t.Errorf("compose must mount %s", want)
		}
	}
	if strings.Contains(text, "ipv4_address") || strings.Contains(text, "traefik") {
		t.Error("no static IP and no front proxy any more")
	}

	k3s := yamlDocs(t, files[env.K3sConfigFile])[0]
	if labels := k3s["node-label"].([]any); labels[0] != "belgiantrain.be/env-lcl=selected" {
		t.Fatalf("node label %v", labels)
	}
	if etcd := strings.Join(toStrings(k3s["etcd-arg"].([]any)), " "); !strings.Contains(etcd, "quota-backend-bytes=8589934592") || !strings.Contains(etcd, "auto-compaction-retention=30m") {
		t.Fatalf("etcd protection missing: %s", etcd)
	}
	if api := k3s["kube-apiserver-arg"].([]any); api[0] != "event-ttl=30m" {
		t.Fatalf("event ttl %v", api)
	}

	nginx := yamlDocs(t, files["platform/10-nginx-ingress.yaml"])[0]
	spec := nginx["spec"].(map[string]any)
	if nginx["kind"] != "HelmChart" || spec["repo"] != "https://helm.nginx.com/stable" || spec["version"] != "2.8.1" {
		t.Fatalf("nginx HelmChart %v", spec)
	}
	var values map[string]any
	if err := yaml.Unmarshal([]byte(spec["valuesContent"].(string)), &values); err != nil {
		t.Fatalf("valuesContent must be YAML: %v", err)
	}
	ctl := values["controller"].(map[string]any)
	class := ctl["ingressClass"].(map[string]any)
	entries := ctl["config"].(map[string]any)["entries"].(map[string]any)
	if class["name"] != "nginx-lcl" || class["setAsDefaultIngress"] != true || ctl["enableSnippets"] != true {
		t.Fatalf("ingress class %v", class)
	}
	if !strings.Contains(entries["http-snippets"].(string), "map $request_uri $gd_forwarded_prefix") ||
		!strings.Contains(entries["location-snippets"].(string), "rewrite ^/[^/]+(/.*)$ $1 break;") ||
		!strings.Contains(entries["location-snippets"].(string), "proxy_set_header X-Forwarded-Prefix $gd_forwarded_prefix;") {
		t.Fatalf("the NGINX variables must survive templating: %v", entries)
	}
	if nodePort := ctl["service"].(map[string]any)["httpsPort"].(map[string]any)["nodePort"]; nodePort != 30443 {
		t.Fatalf("https node port %v", nodePort)
	}

	master := yamlDocs(t, files["platform/30-api-master.yaml"])[0]
	meta := master["metadata"].(map[string]any)
	if meta["namespace"] != "til-lcl" || meta["annotations"].(map[string]any)["nginx.org/mergeable-ingress-type"] != "master" ||
		master["spec"].(map[string]any)["ingressClassName"] != "nginx-lcl" || !strings.Contains(string(files["platform/30-api-master.yaml"]), "- api.til.lcl-intern-belgiantrain.be") {
		t.Fatalf("master ingress %v", master)
	}

	oracle := yamlDocs(t, files["platform/40-oracle.yaml"])
	if len(oracle) != 3 || oracle[2]["kind"] != "StatefulSet" || !strings.Contains(string(files["platform/40-oracle.yaml"]), "chown -R 54321:54321") ||
		!strings.Contains(string(files["platform/40-oracle.yaml"]), "[ -d /opt/oracle/oradata/dbconfig/FREE ] || rm -rf /opt/oracle/oradata/FREE") ||
		!strings.Contains(string(files["platform/40-oracle.yaml"]), "path: /var/lib/rancher/go-dark/oracle") || !strings.Contains(string(files["platform/40-oracle.yaml"]), "password: changeit_local") {
		t.Fatalf("oracle manifests: %s", files["platform/40-oracle.yaml"])
	}
	if ns := yamlDocs(t, files["platform/00-namespace.yaml"])[0]["metadata"].(map[string]any); ns["name"] != "til-lcl" || ns["labels"].(map[string]any)["belgiantrain.be/environment"] != "lcl" {
		t.Fatalf("namespace %v", ns)
	}
	if reloader := yamlDocs(t, files["platform/20-reloader.yaml"])[0]["spec"].(map[string]any); reloader["version"] != "2.2.19" {
		t.Fatalf("reloader %v", reloader)
	}
	if !strings.Contains(string(files[env.ReadmeFile]), "127.0.0.1 api.til.lcl-intern-belgiantrain.be rancher.lcl-intern-belgiantrain.be") {
		t.Fatalf("readme must give the hosts line:\n%s", files[env.ReadmeFile])
	}
}

func toStrings(in []any) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.(string)
	}
	return out
}

func TestFiles_NoStripPrefixNoDatabase(t *testing.T) {
	e := env.New(config.TeamEnvironment{Team: "acme", Environment: config.Environment{Application: "shop", Domain: "shop.local",
		Database: config.EnvironmentDatabase{Type: config.DatabaseNone}}}, "secret")
	files, err := env.Files(e)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["platform/40-oracle.yaml"]; ok {
		t.Error("no database wanted, no oracle manifest")
	}
	if nginx := string(files["platform/10-nginx-ingress.yaml"]); strings.Contains(nginx, "location-snippets") || !strings.Contains(nginx, "name: nginx-lcl") {
		t.Errorf("without strip_prefix there is no rewrite snippet:\n%s", nginx)
	}
	if ns := string(files["platform/00-namespace.yaml"]); !strings.Contains(ns, "name: shop-lcl") {
		t.Errorf("namespace defaults to <application>-<name>:\n%s", ns)
	}
}

func TestWrite_AndTheRuntimeManifests(t *testing.T) {
	e := til(t)
	dir := t.TempDir()
	files, _ := env.Files(e)
	files[env.KeyFile] = []byte("key")
	if err := env.Write(dir, files); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(filepath.Join(dir, "platform", "10-nginx-ingress.yaml")); err != nil || st.Size() == 0 {
		t.Fatalf("files must land under dir: %v", err)
	}
	if st, _ := os.Stat(filepath.Join(dir, "certs", "tls.key")); st.Mode().Perm() != 0o600 {
		t.Errorf("the key must be private, got %v", st.Mode().Perm())
	}

	tls := yamlDocs(t, env.TLSSecretManifest(e, []byte("CERT"), []byte("KEY")))[0]
	data := tls["data"].(map[string]any)
	if tls["type"] != "kubernetes.io/tls" || data["tls.crt"] != "Q0VSVA==" || data["tls.key"] != "S0VZ" || tls["metadata"].(map[string]any)["name"] != env.TLSSecret {
		t.Fatalf("tls secret %v", tls)
	}

	ui, err := env.RancherUIManifest(e, "172.18.0.2")
	if err != nil {
		t.Fatal(err)
	}
	docs := yamlDocs(t, ui)
	if len(docs) != 3 || docs[1]["kind"] != "Endpoints" || !strings.Contains(string(ui), "ip: 172.18.0.2") ||
		!strings.Contains(string(ui), "nginx.org/ssl-services: \"rancher\"") || !strings.Contains(string(ui), "- host: rancher.lcl-intern-belgiantrain.be") ||
		!strings.Contains(string(ui), "nginx.org/location-snippets") {
		t.Fatalf("rancher ui manifests:\n%s", ui)
	}
}
