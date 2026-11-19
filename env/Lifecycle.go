package env

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Inputs is what a lifecycle action needs besides the environment.
type Inputs struct {
	Dir         string   // install directory
	ProjectDirs []string // the team's cloned, deployable services (for the database users)
}

// Step names, as printed in the log ("==> name").
const stepPrefix = "==> "

// pollInterval is how often readiness is re-checked; shortened in tests.
var pollInterval = 3 * time.Second

// runner executes the lifecycle's commands, streaming their output to w so
// the whole action is one inspectable log.
type runner struct {
	ctx context.Context
	w   io.Writer
	dir string
}

func (r *runner) step(format string, args ...any) {
	fmt.Fprintf(r.w, "\n%s%s\n", stepPrefix, fmt.Sprintf(format, args...))
}

func (r *runner) info(format string, args ...any) {
	fmt.Fprintf(r.w, "    %s\n", fmt.Sprintf(format, args...))
}

func (r *runner) cmd(stdin string, name string, args ...string) *exec.Cmd {
	c := exec.CommandContext(r.ctx, name, args...)
	if st, err := os.Stat(r.dir); err == nil && st.IsDir() {
		c.Dir = r.dir
	}
	if stdin != "" {
		c.Stdin = strings.NewReader(stdin)
	}
	return c
}

// run runs a command with its output in the log.
func (r *runner) run(stdin string, name string, args ...string) error {
	fmt.Fprintf(r.w, "  $ %s\n", strings.Join(append([]string{name}, args...), " "))
	c := r.cmd(stdin, name, args...)
	c.Stdout, c.Stderr = r.w, r.w
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// capture runs a command and returns its stdout; stderr goes to the log.
func (r *runner) capture(stdin string, name string, args ...string) (string, error) {
	c := r.cmd(stdin, name, args...)
	var out bytes.Buffer
	c.Stdout, c.Stderr = &out, r.w
	if err := c.Run(); err != nil {
		return out.String(), fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out.String(), nil
}

// quiet runs a command and reports only whether it succeeded.
func (r *runner) quiet(name string, args ...string) bool {
	c := r.cmd("", name, args...)
	c.Stdout, c.Stderr = io.Discard, io.Discard
	return c.Run() == nil
}

// waitFor polls check until it succeeds or the timeout passes.
func (r *runner) waitFor(what string, timeout time.Duration, check func() bool) error {
	deadline := time.Now().Add(timeout)
	for {
		if check() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s: not ready after %s", what, timeout)
		}
		select {
		case <-r.ctx.Done():
			return r.ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// kubectl prefixes the explicit context every call carries.
func (e Env) kubectl(args ...string) []string {
	return append([]string{"--context", e.Context()}, args...)
}

// compose is the compose command for the environment's project and file.
func (e Env) compose(dir string, args ...string) []string {
	return append([]string{"compose", "-p", e.Project(), "-f", filepath.Join(dir, ComposeFile)}, args...)
}

// Install sets the environment up, or repairs it; every step is idempotent.
// Nothing is installed on the machine (the preflight has to pass), the
// hosts file is not touched (the log says what to add).
func Install(ctx context.Context, e Env, in Inputs, w io.Writer) error {
	r := &runner{ctx: ctx, w: w, dir: in.Dir}

	r.step("Preflight")
	checks := Preflight(e)
	for _, c := range checks {
		mark := "ok  "
		switch c.State {
		case CheckWarn:
			mark = "warn"
		case CheckFail:
			mark = "FAIL"
		}
		r.info("%s %-16s %s", mark, c.Name, c.Detail)
		if c.Fix != "" {
			r.info("     %-16s -> %s", "", c.Fix)
		}
	}
	if Blocking(checks) {
		return errors.New("preflight failed: fix the items above first")
	}

	r.step("Writing %s", in.Dir)
	files, err := Files(e)
	if err != nil {
		return err
	}
	if err := Write(in.Dir, files); err != nil {
		return err
	}
	for _, name := range sortedKeys(files) {
		r.info("%s", name)
	}

	r.step("Certificate for %s (mkcert)", strings.Join(e.HostNames(), ", "))
	if err := r.run("", "mkcert", append([]string{"-cert-file", filepath.FromSlash(CertFile), "-key-file", filepath.FromSlash(KeyFile)}, e.HostNames()...)...); err != nil {
		return err
	}
	cert, err := os.ReadFile(filepath.Join(in.Dir, filepath.FromSlash(CertFile)))
	if err != nil {
		return err
	}
	key, err := os.ReadFile(filepath.Join(in.Dir, filepath.FromSlash(KeyFile)))
	if err != nil {
		return err
	}
	if err := Write(in.Dir, map[string][]byte{PlatformDir + "/35-api-tls.yaml": TLSSecretManifest(e, cert, key)}); err != nil {
		return err
	}

	r.step("Starting %s (docker compose)", e.Container())
	if err := r.run("", "docker", e.compose(in.Dir, "up", "-d")...); err != nil {
		return err
	}
	if err := waitK3s(r, e); err != nil {
		return err
	}

	if err := writeContext(r, e); err != nil {
		return err
	}
	if err := fixCoreDNS(r, e); err != nil {
		return err
	}

	r.step("Platform: namespace %s, NGINX Inc ingress, Reloader, master ingress%s", e.Namespace, map[bool]string{true: ", Oracle", false: ""}[e.HasDatabase()])
	nodeIP, err := r.capture("", "kubectl", e.kubectl("get", "nodes", "-o", `jsonpath={.items[0].status.addresses[?(@.type=="InternalIP")].address}`)...)
	if err != nil {
		return err
	}
	nodeIP = strings.TrimSpace(nodeIP)
	if nodeIP == "" {
		return errors.New("the node has no InternalIP yet")
	}
	ui, err := RancherUIManifest(e, nodeIP)
	if err != nil {
		return err
	}
	if err := Write(in.Dir, map[string][]byte{PlatformDir + "/50-rancher-ui.yaml": ui}); err != nil {
		return err
	}
	if err := r.run("", "kubectl", e.kubectl("apply", "-f", filepath.Join(in.Dir, PlatformDir))...); err != nil {
		return err
	}
	r.info("waiting for the ingress controller (the helm-controller fetches the chart first)...")
	if err := r.waitFor("nginx ingress", 7*time.Minute, func() bool {
		return r.quiet("kubectl", e.kubectl("-n", "nginx-ingress", "rollout", "status", "deploy/nginx-ingress-controller", "--timeout=5s")...)
	}); err != nil {
		return fmt.Errorf("%w (kubectl --context %s -n kube-system logs job/helm-install-nginx-ingress)", err, e.Context())
	}
	if err := r.waitFor("reloader", 5*time.Minute, func() bool {
		return r.quiet("kubectl", e.kubectl("-n", "reloader", "rollout", "status", "deploy/reloader-reloader", "--timeout=5s")...)
	}); err != nil {
		return err
	}
	r.info("ingress class %s (default), NodePorts %d/%d published as 80/443", e.IngressClass(), HTTPNodePort, HTTPSNodePort)

	if e.HasDatabase() {
		if err := installDatabase(r, e, in); err != nil {
			return err
		}
	}

	r.step("Done")
	r.info("services    %s/<service path>", e.APIURL())
	r.info("rancher     https://%s (admin / developer)", e.RancherHost())
	r.info("context     kubectl --context %s -n %s get pods", e.Context(), e.Namespace)
	if missing := MissingHosts(e); len(missing) > 0 {
		r.info("hosts file  add: %s", e.HostsLine())
	}
	return nil
}

// waitK3s waits until the k3s API answers inside the container.
func waitK3s(r *runner, e Env) error {
	r.info("waiting for k3s (the first start creates the cluster, a minute or two)...")
	return r.waitFor("k3s", 10*time.Minute, func() bool {
		return r.quiet("docker", "exec", "-e", "KUBECONFIG="+K3sKubeconfig, e.Container(), "kubectl", "get", "--raw", "/readyz")
	})
}

// kubeconfig is the part of k3s' kubeconfig the context needs.
type kubeconfig struct {
	Clusters []struct {
		Cluster struct {
			CA string `yaml:"certificate-authority-data"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		User struct {
			Cert string `yaml:"client-certificate-data"`
			Key  string `yaml:"client-key-data"`
		} `yaml:"user"`
	} `yaml:"users"`
}

// writeContext adds or refreshes the kube context go-dark deploys through,
// pointing at the API published on 127.0.0.1. The current context is never
// changed, so nothing that runs kubectl without --context changes target.
func writeContext(r *runner, e Env) error {
	r.step("Kube context %s -> https://127.0.0.1:%d", e.Context(), APIPort)
	raw, err := r.capture("", "docker", "exec", e.Container(), "cat", K3sKubeconfig)
	if err != nil {
		return err
	}
	var kc kubeconfig
	if err := yaml.Unmarshal([]byte(raw), &kc); err != nil || len(kc.Clusters) == 0 || len(kc.Users) == 0 {
		return fmt.Errorf("unexpected kubeconfig in %s: %v", e.Container(), err)
	}
	tmp, err := os.MkdirTemp("", "go-dark-kube-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	files := map[string]string{"ca": kc.Clusters[0].Cluster.CA, "cert": kc.Users[0].User.Cert, "key": kc.Users[0].User.Key}
	for name, b64 := range files {
		data, err := decodeBase64(b64)
		if err != nil {
			return fmt.Errorf("kubeconfig %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(tmp, name), data, 0o600); err != nil {
			return err
		}
	}
	ctx := e.Context()
	for _, args := range [][]string{
		{"config", "set-cluster", ctx, "--server", fmt.Sprintf("https://127.0.0.1:%d", APIPort), "--certificate-authority", filepath.Join(tmp, "ca"), "--embed-certs"},
		{"config", "set-credentials", ctx, "--client-certificate", filepath.Join(tmp, "cert"), "--client-key", filepath.Join(tmp, "key"), "--embed-certs"},
		{"config", "set-context", ctx, "--cluster", ctx, "--user", ctx, "--namespace", e.Namespace},
	} {
		if err := r.run("", "kubectl", args...); err != nil {
			return err
		}
	}
	if err := r.waitFor("API on 127.0.0.1", time.Minute, func() bool {
		return r.quiet("kubectl", e.kubectl("get", "--raw", "/readyz")...)
	}); err != nil {
		return fmt.Errorf("%w: context written but the API doesn't answer on 127.0.0.1:%d (is the port published?)", err, APIPort)
	}
	// config view prints nothing when no current context is set, where
	// `config current-context` would log an error.
	current, _ := r.capture("", "kubectl", "config", "view", "-o", "jsonpath={.current-context}")
	if current = strings.TrimSpace(current); current == "" {
		current = "none set"
	}
	r.info("kubectl --context %s works; current context unchanged (%s)", ctx, current)
	return nil
}

// fixCoreDNS repoints CoreDNS at Docker's resolver when the container's
// resolv.conf names one. The stock Corefile forwards to /etc/resolv.conf,
// which fails on hosts where public DNS is blocked: pods can then resolve
// neither chart repositories nor registries. Docker Desktop writes its VM
// resolver into the container's resolv.conf (as a comment, next to its
// embedded 127.0.0.11); on hosts without one, CoreDNS is left alone.
func fixCoreDNS(r *runner, e Env) error {
	r.step("CoreDNS upstream")
	upstream := e.DNSUpstream
	if upstream == "" {
		resolv, err := r.capture("", "docker", "exec", e.Container(), "cat", "/etc/resolv.conf")
		if err != nil {
			return err
		}
		upstream = resolverIn(resolv)
	}
	if upstream == "" {
		r.info("no resolver found in the container's resolv.conf; CoreDNS keeps forwarding to /etc/resolv.conf")
		return nil
	}
	corefile, err := r.capture("", "kubectl", e.kubectl("-n", "kube-system", "get", "configmap", "coredns", "-o", "jsonpath={.data.Corefile}")...)
	if err != nil {
		return err
	}
	if strings.TrimSpace(corefile) == "" {
		corefile = corefileBaseline
	}
	patched := forwardLine.ReplaceAllString(corefile, "${1}forward . "+upstream)
	if !forwardLine.MatchString(corefile) {
		patched = strings.Replace(corefile, "\n}", "\n    forward . "+upstream+"\n}", 1)
	}
	if patched == corefile {
		r.info("CoreDNS already forwards to %s", upstream)
		return nil
	}
	r.info("forwarding to %s", upstream)
	// A merge patch carries the Corefile inline: no stdin, no temp file, and
	// it works the same on Windows.
	patch, err := json.Marshal(map[string]any{"data": map[string]string{"Corefile": patched}})
	if err != nil {
		return err
	}
	if err := r.run("", "kubectl", e.kubectl("-n", "kube-system", "patch", "configmap", "coredns", "--type", "merge", "-p", string(patch))...); err != nil {
		return err
	}
	if err := r.run("", "kubectl", e.kubectl("-n", "kube-system", "rollout", "restart", "deploy/coredns")...); err != nil {
		return err
	}
	return r.waitFor("coredns", 3*time.Minute, func() bool {
		return r.quiet("kubectl", e.kubectl("-n", "kube-system", "rollout", "status", "deploy/coredns", "--timeout=5s")...)
	})
}

// installDatabase waits for Oracle and creates the schema users.
func installDatabase(r *runner, e Env, in Inputs) error {
	r.step("Database: Oracle Free in %s", e.Namespace)
	r.info("waiting for the database (the first start creates it, a few minutes)...")
	if err := r.waitFor("oracle", 15*time.Minute, func() bool {
		return r.quiet("kubectl", e.kubectl("-n", e.Namespace, "rollout", "status", "statefulset/oracle", "--timeout=5s")...)
	}); err != nil {
		return fmt.Errorf("%w (kubectl --context %s -n %s logs oracle-0)", err, e.Context(), e.Namespace)
	}
	users := Users(in.ProjectDirs, e.ConfigFrom, e.Database.Users)
	if len(users) == 0 {
		r.info("no spring.datasource.username found under <project>/helm/*/config/%s - no schema users created", e.ConfigFrom)
		return nil
	}
	return createUsers(r, e, users)
}

// createUsers runs the schema SQL as SYSDBA and verifies the users exist
// afterwards (sqlplus exits 0 on empty input).
func createUsers(r *runner, e Env, users []DBUser) error {
	sqlplus := e.kubectl("-n", e.Namespace, "exec", "-i", "oracle-0", "-c", "oracle", "--",
		"sh", "-c", `sqlplus -s -L sys/"$ORACLE_PASSWORD"@localhost:1521/`+OracleService+" as sysdba")
	if _, err := r.capture(SchemaSQL(users, e.DBPassword), "kubectl", sqlplus...); err != nil {
		return fmt.Errorf("creating the database users failed: %w", err)
	}
	out, err := r.capture(CountSQL(users), "kubectl", sqlplus...)
	if err != nil {
		return err
	}
	names := UserNames(users)
	if got := strings.TrimSpace(out); got != fmt.Sprint(len(names)) {
		return fmt.Errorf("expected %d database users after creation, found %q", len(names), got)
	}
	r.info("database users (app -> schema owner), password %s:", e.DBPassword)
	for _, u := range users {
		r.info("  %-16s -> %s", u.App, u.Owner)
	}
	r.info("jdbc url %s", e.DBURL())
	return nil
}

// installed reports whether the install directory has its compose file;
// without it the container is addressed by name.
func installed(in Inputs) bool {
	_, err := os.Stat(filepath.Join(in.Dir, ComposeFile))
	return err == nil
}

// Start starts a stopped environment, waits for its API and re-applies the
// CoreDNS upstream, which k3s resets to its stock Corefile when it boots.
func Start(ctx context.Context, e Env, in Inputs, w io.Writer) error {
	r := &runner{ctx: ctx, w: w, dir: in.Dir}
	r.step("Starting %s", e.Container())
	argv := []string{"start", e.Container()}
	if installed(in) {
		argv = e.compose(in.Dir, "start")
	}
	if err := r.run("", "docker", argv...); err != nil {
		return err
	}
	if err := waitK3s(r, e); err != nil {
		return err
	}
	if err := r.waitFor("API on 127.0.0.1", time.Minute, func() bool {
		return r.quiet("kubectl", e.kubectl("get", "--raw", "/readyz")...)
	}); err != nil {
		return err
	}
	return fixCoreDNS(r, e)
}

// Stop stops the environment; the cluster and its data are kept.
func Stop(ctx context.Context, e Env, in Inputs, w io.Writer) error {
	r := &runner{ctx: ctx, w: w, dir: in.Dir}
	r.step("Stopping %s (data kept)", e.Container())
	if !installed(in) {
		return r.run("", "docker", "stop", e.Container())
	}
	return r.run("", "docker", e.compose(in.Dir, "stop")...)
}

// Uninstall removes the container (and, with volumes, the cluster and
// database data), the kube context and the generated files. Certificates
// are kept unless volumes is set.
func Uninstall(ctx context.Context, e Env, in Inputs, volumes bool, w io.Writer) error {
	r := &runner{ctx: ctx, w: w, dir: in.Dir}
	r.step("Removing %s", e.Container())
	args := e.compose(in.Dir, "down")
	if volumes {
		args = append(args, "-v")
	}
	if installed(in) {
		if err := r.run("", "docker", args...); err != nil {
			return err
		}
	} else {
		r.info("no %s in %s; removing the container by name", ComposeFile, in.Dir)
		_ = r.run("", "docker", "rm", "-f", e.Container())
	}
	r.step("Removing kube context %s", e.Context())
	for _, args := range [][]string{{"config", "delete-context", e.Context()}, {"config", "delete-cluster", e.Context()}, {"config", "delete-user", e.Context()}} {
		if !r.quiet("kubectl", args...) {
			r.info("kubectl %s: nothing to remove", strings.Join(args, " "))
		}
	}
	r.step("Removing generated files")
	for _, name := range []string{ComposeFile, filepath.Dir(K3sConfigFile), PlatformDir, ChartsDir, ReadmeFile, "certs/isrg-root-yr.pem", "certs/05e55783.0", "certs/558fe057.0"} {
		_ = os.RemoveAll(filepath.Join(in.Dir, filepath.FromSlash(name)))
	}
	if volumes {
		_ = os.RemoveAll(filepath.Join(in.Dir, CertsDir))
		_ = os.Remove(in.Dir)
		r.info("%s removed, including the cluster and database volumes", in.Dir)
	} else {
		r.info("volumes kept (docker volume ls | grep %s); certificates kept in %s", e.Project(), filepath.Join(in.Dir, CertsDir))
	}
	return nil
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}
