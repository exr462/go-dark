package env

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Status is what the installer screen shows about an environment.
type Status struct {
	Installed bool   // the install directory has a compose file
	Container string // "running", "stopped" or "missing"
	K3sReady  bool   // the API answers through the kube context
	EtcdBytes int64  // apiserver storage size (0 when unknown)
	Ingress   string // controller rollout, e.g. "ready", "not ready", "not installed"
	Database  string
	Releases  []string // helm releases in the namespace
	Missing   []string // hosts not yet in the hosts file
}

// Describe gathers the status with quick, read-only docker/kubectl/helm
// calls (every one names the context explicitly).
func Describe(e Env, dir string) Status {
	var st Status
	if _, err := os.Stat(dir + "/" + ComposeFile); err == nil {
		st.Installed = true
	}
	st.Container = "missing"
	if out, err := output("docker", "inspect", "-f", "{{.State.Running}}", e.Container()); err == nil {
		st.Container = "stopped"
		if strings.TrimSpace(string(out)) == "true" {
			st.Container = "running"
		}
	}
	st.Missing = MissingHosts(e)
	if st.Container != "running" {
		return st
	}
	if _, err := output("kubectl", e.kubectl("get", "--raw", "/readyz")...); err != nil {
		return st
	}
	st.K3sReady = true
	if out, err := output("kubectl", e.kubectl("get", "--raw", "/metrics")...); err == nil {
		st.EtcdBytes = storageSize(string(out))
	}
	st.Ingress = rollout(e, "nginx-ingress", "deployment", "nginx-ingress-controller")
	if e.HasDatabase() {
		st.Database = rollout(e, e.Namespace, "statefulset", "oracle")
	} else {
		st.Database = "none"
	}
	if out, err := output("helm", "list", "--kube-context", e.Context(), "--namespace", e.Namespace, "-o", "json"); err == nil {
		var releases []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		if json.Unmarshal(out, &releases) == nil {
			for _, r := range releases {
				st.Releases = append(st.Releases, r.Name+" ("+r.Status+")")
			}
		}
	}
	return st
}

// rollout summarizes a workload: "ready" when every wanted replica is.
func rollout(e Env, ns, kind, name string) string {
	out, err := output("kubectl", e.kubectl("-n", ns, "get", kind, name, "-o", "jsonpath={.status.readyReplicas}/{.spec.replicas}")...)
	if err != nil {
		return "not installed"
	}
	ready, want, _ := strings.Cut(strings.TrimSpace(string(out)), "/")
	if ready == "" {
		ready = "0"
	}
	if ready == want {
		return "ready"
	}
	return fmt.Sprintf("not ready (%s/%s)", ready, want)
}

var storageMetric = regexp.MustCompile(`(?m)^(?:apiserver_storage_size_bytes|etcd_db_total_size_in_bytes)\{[^}]*\}\s+([0-9.e+]+)`)

// storageSize reads the etcd database size from the apiserver metrics.
func storageSize(metrics string) int64 {
	m := storageMetric.FindStringSubmatch(metrics)
	if m == nil {
		return 0
	}
	f, _ := strconv.ParseFloat(m[1], 64)
	return int64(f)
}

// HumanBytes formats a size for the screen.
func HumanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MiB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%d B", n)
}

// hostsFile is the system hosts file; replaced in tests.
var hostsFile = func() string {
	if goos == "windows" {
		return `C:\Windows\System32\drivers\etc\hosts`
	}
	return "/etc/hosts"
}

// MissingHosts lists the environment's hosts that the hosts file doesn't
// point at this machine yet.
func MissingHosts(e Env) []string {
	data, err := os.ReadFile(hostsFile())
	present := map[string]bool{}
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line, _, _ = strings.Cut(line, "#")
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			for _, h := range fields[1:] {
				present[strings.ToLower(h)] = true
			}
		}
	}
	var missing []string
	for _, h := range e.HostNames() {
		if !present[h] {
			missing = append(missing, h)
		}
	}
	return missing
}

// HostsCommand is how the user adds the hosts line: an interactive sudo on
// macOS/Linux (go-dark runs it in its embedded terminal, so the password is
// typed there), an elevated PowerShell command on Windows.
func HostsCommand(e Env) (argv []string, hint string) {
	line := e.HostsLine()
	if goos == "windows" {
		return nil, fmt.Sprintf(`Add-Content -Path "%s" -Value "%s"   # in an elevated PowerShell`, hostsFile(), line)
	}
	return []string{"sh", "-c", `echo "adding to ` + hostsFile() + `: $1"; printf '%s\n' "$1" | sudo tee -a ` + hostsFile() + ` >/dev/null && echo done`, "go-dark", line},
		"sudo tee -a " + hostsFile() + "   # adds: " + line
}

// Conflict is a container that holds one of the environment's ports.
type Conflict struct {
	Container string
	Project   string // its compose project, if any
	Ports     []int
}

// Conflicts lists other containers publishing the environment's ports: the
// old sandbox's rancher/traefik, another team's environment... Our own
// container is not a conflict.
func Conflicts(e Env) []Conflict {
	out, err := output("docker", "ps", "--format", `{{.Names}}|{{.Label "com.docker.compose.project"}}|{{.Ports}}`)
	if err != nil {
		return nil
	}
	var conflicts []Conflict
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 || parts[0] == e.Container() {
			continue
		}
		c := Conflict{Container: parts[0], Project: parts[1]}
		for _, port := range Ports {
			if hostPort.MatchString(parts[2]+",") && strings.Contains(parts[2]+",", ":"+strconv.Itoa(port)+"->") {
				c.Ports = append(c.Ports, port)
			}
		}
		if len(c.Ports) > 0 {
			conflicts = append(conflicts, c)
		}
	}
	return conflicts
}

var hostPort = regexp.MustCompile(`:\d+->`)

// StopArgv stops the compose project of a conflicting container (never its
// volumes or files), or the container itself when it isn't from compose.
func StopArgv(c Conflict) []string {
	if c.Project != "" {
		return []string{"docker", "compose", "-p", c.Project, "stop"}
	}
	return []string{"docker", "stop", c.Container}
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(s))
}

var ipv4 = regexp.MustCompile(`\b(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\b`)

// resolverIn picks the first non-loopback IPv4 address in a resolv.conf,
// comments included (Docker Desktop lists its VM resolver there).
func resolverIn(resolv string) string {
	for _, m := range ipv4.FindAllString(resolv, -1) {
		if !strings.HasPrefix(m, "127.") {
			return m
		}
	}
	return ""
}

var forwardLine = regexp.MustCompile(`(?m)^([ \t]*)forward[ \t]+\.[ \t]+.*$`)

// corefileBaseline is the stock k3s Corefile, used when the live one is
// missing or blank.
const corefileBaseline = `.:53 {
    errors
    health
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
      pods insecure
      fallthrough in-addr.arpa ip6.arpa
    }
    hosts /etc/coredns/NodeHosts {
      ttl 60
      reload 15s
      fallthrough
    }
    prometheus :9153
    cache 30
    loop
    reload
    loadbalance
    import /etc/coredns/custom/*.override
    forward . /etc/resolv.conf
}
import /etc/coredns/custom/*.server
`

func sortStrings(s []string) { sort.Strings(s) }
