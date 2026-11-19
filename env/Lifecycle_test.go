//go:build !windows

package env

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/exr462/go-dark/config"
)

var rail = New(config.TeamEnvironment{Team: "rail", Environment: config.Environment{Application: "til", Domain: "lcl.local",
	Ingress: config.EnvironmentIngress{StripPrefix: true}}}, "")

// fakeTools puts docker/kubectl/helm/mkcert scripts on PATH that log their
// calls and answer like a healthy machine. stdin of kubectl exec/apply is
// logged too.
func fakeTools(t *testing.T) (bin, log string) {
	t.Helper()
	bin = t.TempDir()
	log = filepath.Join(t.TempDir(), "calls.log")
	b64 := base64.StdEncoding.EncodeToString([]byte("pem"))
	kubeconfig := "apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: " + b64 + "\n    server: https://127.0.0.1:6443\n  name: default\nusers:\n- name: default\n  user:\n    client-certificate-data: " + b64 + "\n    client-key-data: " + b64 + "\n"
	scripts := map[string]string{
		"docker": `echo "docker $*" >> ` + log + `
case "$*" in
  "info "*) echo "8 8345035776";;
  "compose version") echo v2;;
  "ps "*) echo "other|otherproj|0.0.0.0:8080->80/tcp";;
  "inspect "*) echo true;;
  "port "*) echo "0.0.0.0:80";;
  *"cat /etc/rancher/k3s/k3s.yaml") printf '%s' "$(cat <<'EOF'
` + kubeconfig + `EOF
)";;
  *"cat /etc/resolv.conf") printf 'nameserver 127.0.0.11\n# ExtServers: [host(192.168.65.7)]\n';;
  *) ;;
esac`,
		"kubectl": `case "$*" in
  *"apply -f -"*|*"exec -i"*) in=$(cat); echo "kubectl $* <<< $(printf '%s' "$in" | head -c 400 | tr '\n' ' ')" >> ` + log + `;;
  *) echo "kubectl $*" >> ` + log + `;;
esac
case "$*" in
  *"get --raw /readyz"*) echo ok;;
  *"get --raw /metrics"*) echo 'apiserver_storage_size_bytes{cluster="etcd-0"} 2.3e+07';;
  *"get nodes"*) echo 172.18.0.2;;
  *"get configmap coredns"*) printf '.:53 {\n    forward . /etc/resolv.conf\n}\n';;
  *"create configmap coredns"*) echo "apiVersion: v1\nkind: ConfigMap\ndata:\n  Corefile: patched";;
  *"current-context") echo azr-prd-idas-aks01;;
  *"exec -i oracle-0"*) case "$in" in *"select count(*)"*) echo 2;; esac;;
  *"get deployment"*|*"get statefulset"*) echo "1/1";;
esac`,
		"helm":   `echo "helm $*" >> ` + log + `; echo '[{"name":"til-order","status":"deployed"}]'`,
		"mkcert": `echo "mkcert $*" >> ` + log + `; echo CERT > "$2"; echo KEY > "$4"`,
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	orig := pollInterval
	pollInterval = time.Millisecond
	t.Cleanup(func() { pollInterval = orig })
	return bin, log
}

// props writes a service's config/<env> properties and returns the project dir.
func props(t *testing.T, root, project, envName, body string) string {
	t.Helper()
	dir := filepath.Join(root, project)
	f := filepath.Join(dir, "helm", project, "config", envName, "application-k8s.properties")
	_ = os.MkdirAll(filepath.Dir(f), 0o755)
	if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func calls(t *testing.T, log string) string {
	t.Helper()
	data, _ := os.ReadFile(log)
	return string(data)
}

func TestInstall_RunsEveryStepThroughTheExplicitContext(t *testing.T) {
	_, log := fakeTools(t)
	root := t.TempDir()
	order := props(t, root, "til-order", "dev", "spring.datasource.username=TORA\nspring.liquibase.user=order_developer\n")
	dir := filepath.Join(t.TempDir(), "rail")
	var out bytes.Buffer

	err := Install(context.Background(), rail, Inputs{Dir: dir, ProjectDirs: []string{order}}, &out)
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	got := calls(t, log)
	for _, want := range []string{
		"mkcert -cert-file certs/tls.crt -key-file certs/tls.key api.til.lcl.local rancher.lcl.local",
		"docker compose -p go-dark-rail -f " + dir + "/docker-compose.yaml up -d",
		"docker exec -e KUBECONFIG=/etc/rancher/k3s/k3s.yaml go-dark-rail-rancher kubectl get --raw /readyz",
		"docker exec go-dark-rail-rancher cat /etc/rancher/k3s/k3s.yaml",
		"kubectl config set-cluster go-dark-rail --server https://127.0.0.1:6550 --certificate-authority",
		"kubectl config set-credentials go-dark-rail --client-certificate",
		"kubectl config set-context go-dark-rail --cluster go-dark-rail --user go-dark-rail --namespace til-lcl",
		"kubectl --context go-dark-rail get --raw /readyz",
		"kubectl --context go-dark-rail -n kube-system apply -f - <<< apiVersion",
		"kubectl --context go-dark-rail -n kube-system rollout restart deploy/coredns",
		"kubectl --context go-dark-rail apply -f " + dir + "/platform",
		"kubectl --context go-dark-rail -n nginx-ingress rollout status deploy/nginx-ingress-controller",
		"kubectl --context go-dark-rail -n reloader rollout status deploy/reloader-reloader",
		"kubectl --context go-dark-rail -n til-lcl rollout status statefulset/oracle",
		"kubectl --context go-dark-rail -n til-lcl exec -i oracle-0 -c oracle -- sh -c sqlplus -s -L sys/\"$ORACLE_PASSWORD\"@localhost:1521/FREEPDB1 as sysdba <<< WHENEVER SQLERROR EXIT FAILURE DECLARE n NUMBER; BEGIN   SELECT COUNT(*) INTO n FROM dba_users WHERE username = 'ORDER_DEVELOPER'",
		"as sysdba <<< set heading off feedback off pages 0 select count(*) from dba_users where username in ('ORDER_DEVELOPER','TORA');",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing call %q in:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{"use-context", "mkcert -install", "sudo", "/etc/hosts"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("install must never run %q:\n%s", forbidden, got)
		}
	}
	// Every kubectl call against the cluster names the context.
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "kubectl") && !strings.HasPrefix(line, "kubectl config") && !strings.Contains(line, "--context go-dark-rail") {
			t.Errorf("kubectl without --context: %s", line)
		}
	}
	for _, f := range []string{ComposeFile, K3sConfigFile, CertFile, KeyFile, "platform/10-nginx-ingress.yaml", "platform/35-api-tls.yaml", "platform/50-rancher-ui.yaml", ReadmeFile} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s must be written: %v", f, err)
		}
	}
	tls, _ := os.ReadFile(filepath.Join(dir, "platform", "35-api-tls.yaml"))
	if !strings.Contains(string(tls), base64.StdEncoding.EncodeToString([]byte("CERT\n"))) {
		t.Error("the tls secret must carry mkcert's certificate")
	}
	ui, _ := os.ReadFile(filepath.Join(dir, "platform", "50-rancher-ui.yaml"))
	if !strings.Contains(string(ui), "ip: 172.18.0.2") {
		t.Error("the rancher ui endpoint must be the node's address")
	}
	text := out.String()
	for _, want := range []string{"==> Preflight", "ok   docker", "==> Kube context go-dark-rail -> https://127.0.0.1:6550", "current context unchanged (azr-prd-idas-aks01)",
		"forwarding to 192.168.65.7", "TORA             -> ORDER_DEVELOPER", "==> Done", "hosts file  add: 127.0.0.1 api.til.lcl.local rancher.lcl.local"} {
		if !strings.Contains(text, want) {
			t.Errorf("log must say %q:\n%s", want, text)
		}
	}
}

func TestInstall_StopsWhenPreflightFails(t *testing.T) {
	_, log := fakeTools(t)
	t.Setenv("PATH", filepath.Join(t.TempDir(), "empty"))
	var out bytes.Buffer
	err := Install(context.Background(), rail, Inputs{Dir: t.TempDir()}, &out)
	if err == nil || !strings.Contains(err.Error(), "preflight failed") || !strings.Contains(out.String(), "FAIL docker") {
		t.Fatalf("expected a preflight failure, got %v\n%s", err, out.String())
	}
	if strings.Contains(calls(t, log), "compose") {
		t.Fatal("nothing may be started when the preflight fails")
	}
}

func TestInstall_RefusesWhenTheUsersAreNotThere(t *testing.T) {
	bin, _ := fakeTools(t)
	// Make the count query answer 1 instead of 2.
	kubectl := filepath.Join(bin, "kubectl")
	data, _ := os.ReadFile(kubectl)
	_ = os.WriteFile(kubectl, bytes.ReplaceAll(data, []byte("echo 2;;"), []byte("echo 1;;")), 0o755)
	root := t.TempDir()
	order := props(t, root, "til-order", "dev", "spring.datasource.username=TORA\nspring.liquibase.user=order_developer\n")
	var out bytes.Buffer
	err := Install(context.Background(), rail, Inputs{Dir: t.TempDir(), ProjectDirs: []string{order}}, &out)
	if err == nil || !strings.Contains(err.Error(), "expected 2 database users after creation, found \"1\"") {
		t.Fatalf("expected the verification to fail, got %v", err)
	}
}

func TestStartStopUninstall(t *testing.T) {
	_, log := fakeTools(t)
	dir := t.TempDir()
	files, _ := Files(rail)
	_ = Write(dir, files)
	_ = Write(dir, map[string][]byte{CertFile: []byte("c"), KeyFile: []byte("k")})
	var out bytes.Buffer

	if err := Start(context.Background(), rail, Inputs{Dir: dir}, &out); err != nil {
		t.Fatal(err)
	}
	if err := Stop(context.Background(), rail, Inputs{Dir: dir}, &out); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), rail, Inputs{Dir: dir}, false, &out); err != nil {
		t.Fatal(err)
	}
	got := calls(t, log)
	for _, want := range []string{"docker compose -p go-dark-rail -f " + dir + "/docker-compose.yaml start", "docker compose -p go-dark-rail -f " + dir + "/docker-compose.yaml stop\n",
		"docker compose -p go-dark-rail -f " + dir + "/docker-compose.yaml down\n", "kubectl config delete-context go-dark-rail", "kubectl config delete-cluster go-dark-rail"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "down -v") {
		t.Error("volumes must be kept unless asked")
	}
	if _, err := os.Stat(filepath.Join(dir, ComposeFile)); err == nil {
		t.Error("generated files must be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(CertFile))); err != nil {
		t.Error("certificates must be kept")
	}

	_ = Write(dir, files)
	if err := Uninstall(context.Background(), rail, Inputs{Dir: dir}, true, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(calls(t, log), "down -v") {
		t.Error("--volumes removes the volumes")
	}
	if _, err := os.Stat(dir); err == nil {
		t.Error("with volumes the install dir goes too")
	}
}

func TestDescribe_AndConflicts(t *testing.T) {
	fakeTools(t)
	hosts := filepath.Join(t.TempDir(), "hosts")
	_ = os.WriteFile(hosts, []byte("127.0.0.1 localhost\n127.0.0.1 api.til.lcl.local # go-dark\n"), 0o644)
	orig := hostsFile
	hostsFile = func() string { return hosts }
	t.Cleanup(func() { hostsFile = orig })
	dir := t.TempDir()
	_ = Write(dir, map[string][]byte{ComposeFile: []byte("name: x")})

	st := Describe(rail, dir)
	if !st.Installed || st.Container != "running" || !st.K3sReady || st.EtcdBytes != 23000000 || st.Ingress != "ready" || st.Database != "ready" ||
		len(st.Releases) != 1 || st.Releases[0] != "til-order (deployed)" || len(st.Missing) != 1 || st.Missing[0] != "rancher.lcl.local" {
		t.Fatalf("status %+v", st)
	}
	if HumanBytes(st.EtcdBytes) != "22 MiB" {
		t.Fatalf("human bytes %s", HumanBytes(st.EtcdBytes))
	}
	argv, hint := HostsCommand(rail)
	if argv[0] != "sh" || argv[len(argv)-1] != "127.0.0.1 api.til.lcl.local rancher.lcl.local" || !strings.Contains(argv[2], "sudo tee -a "+hosts) || !strings.Contains(hint, "sudo tee") {
		t.Fatalf("hosts command %q %q", argv, hint)
	}
	goos = "windows"
	defer func() { goos = "darwin" }()
	if argv, hint := HostsCommand(rail); argv != nil || !strings.Contains(hint, "Add-Content") || !strings.Contains(hint, "elevated PowerShell") {
		t.Fatalf("windows hosts hint %q %q", argv, hint)
	}
}

func TestCLI(t *testing.T) {
	fakeTools(t)
	manifest := filepath.Join(t.TempDir(), "go-dark.json")
	_ = os.WriteFile(manifest, []byte(`{"team":"rail","projects":[{"name":"til-order","deployable":true}],"environment":{"application":"til","domain":"lcl.local"}}`), 0o644)
	dir := filepath.Join(t.TempDir(), "env")
	var out bytes.Buffer
	if code := CLI([]string{"status", "--manifest", manifest, "--dir", dir}, &out); code != 0 {
		t.Fatalf("status exit %d:\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "container   go-dark-rail-rancher (running)") || !strings.Contains(out.String(), "https://api.til.lcl.local/<service>") {
		t.Fatalf("status output:\n%s", out.String())
	}
	out.Reset()
	if code := CLI([]string{"stop", "--manifest", manifest, "--dir", dir}, &out); code != 0 {
		t.Fatalf("stop exit %d:\n%s", code, out.String())
	}
	out.Reset()
	if code := CLI([]string{"dance", "--manifest", manifest}, &out); code != 2 || !strings.Contains(out.String(), "usage") {
		t.Fatalf("unknown action: %d %s", code, out.String())
	}
	if code := CLI([]string{"install"}, &out); code != 2 || !strings.Contains(out.String(), "--manifest or --team") {
		t.Fatalf("missing source: %d %s", code, out.String())
	}
	argv := Argv("/bin/go-dark", "install", manifest, Inputs{Dir: dir, ProjectDirs: []string{"/ws/til-order"}}, "pw", false)
	if strings.Join(argv, " ") != "/bin/go-dark env install --manifest "+manifest+" --dir "+dir+" --project-dir /ws/til-order --db-password pw" {
		t.Fatalf("argv %q", argv)
	}
}
