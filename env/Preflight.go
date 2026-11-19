package env

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// CheckState is the outcome of one preflight check.
type CheckState int

const (
	CheckOK CheckState = iota
	CheckWarn
	CheckFail
)

// Check is one preflight result. go-dark never installs software: a failed
// check comes with the command to run for this OS.
type Check struct {
	Name   string
	State  CheckState
	Detail string // what was found
	Fix    string // how to fix it (install command, what to stop...), empty when OK
}

// Ports the environment publishes on the host.
var Ports = []int{80, 443, APIPort}

// Minimum docker resources: k3s, Rancher, the ingress, Oracle and a few
// Spring services.
const (
	MinCPUs     = 4
	MinMemBytes = 7 * 1024 * 1024 * 1024 // "8 GB" as Docker Desktop reports it
)

// Hooks replaced in tests.
var (
	lookPath = exec.LookPath
	goos     = runtime.GOOS
	dial     = func(addr string) error {
		c, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			_ = c.Close()
		}
		return err
	}
	output = func(name string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, name, args...).Output()
	}
)

// Preflight checks what the installer needs: docker running with enough
// resources, docker compose, kubectl, helm and mkcert on PATH, and the
// published ports free (or held by the environment's own container, when
// repairing). A port held by another container names it and how to stop it.
func Preflight(e Env) []Check {
	var checks []Check

	checks = append(checks, dockerCheck())
	if _, err := output("docker", "compose", "version"); err != nil {
		checks = append(checks, Check{Name: "docker compose", State: CheckFail, Detail: "docker compose doesn't answer", Fix: installHint("docker compose")})
	} else {
		checks = append(checks, Check{Name: "docker compose", Detail: "available"})
	}
	for _, tool := range []string{"kubectl", "helm", "mkcert"} {
		if path, err := lookPath(tool); err != nil {
			checks = append(checks, Check{Name: tool, State: CheckFail, Detail: "not found on PATH", Fix: installHint(tool)})
		} else {
			checks = append(checks, Check{Name: tool, Detail: path})
		}
	}
	holders := map[int]Conflict{}
	for _, c := range Conflicts(e) {
		for _, p := range c.Ports {
			holders[p] = c
		}
	}
	for _, port := range Ports {
		name := "port " + strconv.Itoa(port)
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		busy := dial(addr) == nil
		holder, held := holders[port]
		switch {
		case held:
			checks = append(checks, Check{Name: name, State: CheckFail, Detail: "published by container " + holder.Container + conflictProject(holder),
				Fix: strings.Join(StopArgv(holder), " ")})
		case busy && ownPort(e, port):
			checks = append(checks, Check{Name: name, Detail: "published by " + e.Container()})
		case busy:
			checks = append(checks, Check{Name: name, State: CheckFail, Detail: "something already listens on " + addr,
				Fix: "stop whatever listens on port " + strconv.Itoa(port)})
		default:
			checks = append(checks, Check{Name: name, Detail: "free"})
		}
	}
	return checks
}

func conflictProject(c Conflict) string {
	if c.Project == "" {
		return ""
	}
	return " (compose project " + c.Project + ")"
}

// ownPort reports whether the environment's own container publishes port
// on the host. `docker port <c> <port>` would look up the container side
// (host 80 is container 30080), so every mapping is listed and the host
// side compared: "30080/tcp -> 0.0.0.0:80".
func ownPort(e Env, port int) bool {
	out, err := output("docker", "port", e.Container())
	if err != nil {
		return false
	}
	return hostPorts(string(out))[port]
}

// hostPorts parses `docker port` output into the published host ports.
func hostPorts(out string) map[int]bool {
	ports := map[int]bool{}
	for _, line := range strings.Split(out, "\n") {
		_, host, ok := strings.Cut(line, "->")
		if !ok {
			continue
		}
		host = strings.TrimSpace(host)
		if p, err := strconv.Atoi(host[strings.LastIndex(host, ":")+1:]); err == nil {
			ports[p] = true
		}
	}
	return ports
}

// dockerCheck wants a running daemon with MinCPUs and MinMemBytes; less is a
// warning (it runs, slowly), a missing or stopped docker a failure.
func dockerCheck() Check {
	if _, err := lookPath("docker"); err != nil {
		return Check{Name: "docker", State: CheckFail, Detail: "not found on PATH", Fix: installHint("docker")}
	}
	out, err := output("docker", "info", "--format", "{{.NCPU}} {{.MemTotal}}")
	if err != nil {
		return Check{Name: "docker", State: CheckFail, Detail: "the docker daemon doesn't answer", Fix: startHint()}
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		return Check{Name: "docker", Detail: "running"}
	}
	cpus, _ := strconv.Atoi(fields[0])
	mem, _ := strconv.ParseInt(fields[1], 10, 64)
	detail := fmt.Sprintf("running, %d CPUs, %.1f GB", cpus, float64(mem)/(1024*1024*1024))
	if cpus < MinCPUs || mem < MinMemBytes {
		return Check{Name: "docker", State: CheckWarn, Detail: detail,
			Fix: fmt.Sprintf("give docker at least %d CPUs and 8 GB of memory (Docker Desktop: Settings > Resources)", MinCPUs)}
	}
	return Check{Name: "docker", Detail: detail}
}

// Blocking reports whether any check failed (warnings don't block).
func Blocking(checks []Check) bool {
	for _, c := range checks {
		if c.State == CheckFail {
			return true
		}
	}
	return false
}

// installHints are the install commands per tool and OS.
var installHints = map[string]map[string]string{
	"docker": {
		"darwin":  "brew install --cask docker   # Docker Desktop, then start it",
		"windows": "winget install Docker.DockerDesktop   # then start Docker Desktop",
		"linux":   "sudo apt install docker.io docker-compose-v2 && sudo usermod -aG docker $USER",
	},
	"docker compose": {
		"darwin":  "update Docker Desktop (it bundles docker compose)",
		"windows": "update Docker Desktop (it bundles docker compose)",
		"linux":   "sudo apt install docker-compose-v2",
	},
	"kubectl": {
		"darwin":  "brew install kubectl",
		"windows": "winget install Kubernetes.kubectl",
		"linux":   "sudo snap install kubectl --classic",
	},
	"helm": {
		"darwin":  "brew install helm",
		"windows": "winget install Helm.Helm",
		"linux":   "sudo snap install helm --classic",
	},
	"mkcert": {
		"darwin":  "brew install mkcert nss && mkcert -install",
		"windows": "winget install FiloSottile.mkcert && mkcert -install",
		"linux":   "sudo apt install mkcert libnss3-tools && mkcert -install",
	},
}

func installHint(tool string) string {
	if hint := installHints[tool][goos]; hint != "" {
		return hint
	}
	return "install " + tool
}

func startHint() string {
	switch goos {
	case "linux":
		return "sudo systemctl start docker"
	}
	return "start Docker Desktop"
}
