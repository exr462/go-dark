package env

import (
	"errors"
	"strings"
	"testing"
)

// fakeMachine replaces the command, PATH, port and OS hooks. tools maps a
// tool name to its path ("" = missing); outputs maps "name arg..." prefixes
// to stdout; busy lists the ports something listens on.
func fakeMachine(t *testing.T, os string, tools map[string]string, outputs map[string]string, busy ...string) {
	t.Helper()
	origLook, origOut, origDial, origOS := lookPath, output, dial, goos
	t.Cleanup(func() { lookPath, output, dial, goos = origLook, origOut, origDial, origOS })
	goos = os
	lookPath = func(tool string) (string, error) {
		if p := tools[tool]; p != "" {
			return p, nil
		}
		return "", errors.New("not found")
	}
	output = func(name string, args ...string) ([]byte, error) {
		key := strings.Join(append([]string{name}, args...), " ")
		for prefix, out := range outputs {
			if strings.HasPrefix(key, prefix) {
				return []byte(out), nil
			}
		}
		return nil, errors.New("exit status 1")
	}
	dial = func(addr string) error {
		for _, b := range busy {
			if strings.HasSuffix(addr, ":"+b) {
				return nil
			}
		}
		return errors.New("connection refused")
	}
}

var allTools = map[string]string{"docker": "/usr/local/bin/docker", "kubectl": "/usr/local/bin/kubectl", "helm": "/opt/homebrew/bin/helm", "mkcert": "/opt/homebrew/bin/mkcert"}

func byName(checks []Check) map[string]Check {
	m := map[string]Check{}
	for _, c := range checks {
		m[c.Name] = c
	}
	return m
}

var e = Env{Team: "rail"}

func TestPreflight_AllGood(t *testing.T) {
	fakeMachine(t, "darwin", allTools, map[string]string{"docker info": "8 8345035776\n", "docker compose version": "Docker Compose version v2.29.1"})
	checks := Preflight(e)
	if Blocking(checks) {
		t.Fatalf("nothing should fail: %+v", checks)
	}
	got := byName(checks)
	if got["docker"].Detail != "running, 8 CPUs, 7.8 GB" || got["docker"].State != CheckOK || got["mkcert"].Detail != "/opt/homebrew/bin/mkcert" || got["port 6550"].Detail != "free" {
		t.Fatalf("unexpected checks %+v", checks)
	}
}

func TestPreflight_MissingToolsComeWithInstallCommandsPerOS(t *testing.T) {
	for os, want := range map[string]string{"darwin": "brew install helm", "windows": "winget install Helm.Helm", "linux": "snap install helm"} {
		fakeMachine(t, os, map[string]string{"docker": "/bin/docker", "kubectl": "/bin/kubectl"}, map[string]string{"docker info": "4 8345035776", "docker compose version": "v2"})
		got := byName(Preflight(e))
		if got["helm"].State != CheckFail || !strings.Contains(got["helm"].Fix, want) {
			t.Errorf("%s: helm hint %+v", os, got["helm"])
		}
		if got["mkcert"].State != CheckFail || !strings.Contains(got["mkcert"].Fix, "mkcert -install") {
			t.Errorf("%s: mkcert hint %+v", os, got["mkcert"])
		}
		if !Blocking(Preflight(e)) {
			t.Errorf("%s: missing tools must block", os)
		}
	}
}

func TestPreflight_DockerStatesAndResources(t *testing.T) {
	fakeMachine(t, "linux", map[string]string{}, nil)
	if c := byName(Preflight(e))["docker"]; c.State != CheckFail || !strings.Contains(c.Fix, "apt install docker.io") {
		t.Fatalf("no docker: %+v", c)
	}
	fakeMachine(t, "darwin", allTools, map[string]string{})
	if c := byName(Preflight(e))["docker"]; c.State != CheckFail || c.Fix != "start Docker Desktop" {
		t.Fatalf("stopped docker: %+v", c)
	}
	fakeMachine(t, "linux", allTools, map[string]string{})
	if c := byName(Preflight(e))["docker"]; c.Fix != "sudo systemctl start docker" {
		t.Fatalf("stopped docker on linux: %+v", c)
	}
	fakeMachine(t, "darwin", allTools, map[string]string{"docker info": "2 4294967296", "docker compose version": "v2"})
	checks := Preflight(e)
	if c := byName(checks)["docker"]; c.State != CheckWarn || !strings.Contains(c.Detail, "2 CPUs, 4.0 GB") || !strings.Contains(c.Fix, "8 GB") {
		t.Fatalf("small docker must warn: %+v", c)
	}
	if Blocking(checks) {
		t.Fatal("a warning doesn't block")
	}
	fakeMachine(t, "darwin", allTools, map[string]string{"docker info": "8 8345035776"})
	if c := byName(Preflight(e))["docker compose"]; c.State != CheckFail || !strings.Contains(c.Fix, "Docker Desktop") {
		t.Fatalf("no compose: %+v", c)
	}
}

func TestPreflight_BusyPortsBlock(t *testing.T) {
	fakeMachine(t, "darwin", allTools, map[string]string{"docker info": "8 8345035776", "docker compose version": "v2"}, "80", "6550")
	checks := Preflight(e)
	got := byName(checks)
	if got["port 80"].State != CheckFail || got["port 6550"].State != CheckFail || got["port 443"].State != CheckOK || !Blocking(checks) {
		t.Fatalf("ports 80 and 6550 are busy: %+v", checks)
	}
	if !strings.Contains(got["port 80"].Detail, "127.0.0.1:80") {
		t.Fatalf("detail %+v", got["port 80"])
	}
}

const oldSandbox = "rancher|local-cicd-sandbox|0.0.0.0:3800->80/tcp, 127.0.0.1:6550->6443/tcp\n" +
	"traefik|local-cicd-sandbox|0.0.0.0:80->80/tcp, [::]:80->80/tcp, 0.0.0.0:443->443/tcp\n" +
	"sonarqube|local-cicd-sandbox|9000/tcp\n" +
	"go-dark-rail-rancher|go-dark-rail|0.0.0.0:80->30080/tcp\n"

func TestPreflight_PortsHeldByTheOldSandboxNameItAndHowToStopIt(t *testing.T) {
	fakeMachine(t, "darwin", allTools, map[string]string{"docker info": "8 8345035776", "docker compose version": "v2", "docker ps": oldSandbox}, "80", "443", "6550")
	got := byName(Preflight(e))
	if got["port 80"].Detail != "published by container traefik (compose project local-cicd-sandbox)" || got["port 80"].Fix != "docker compose -p local-cicd-sandbox stop" {
		t.Fatalf("port 80: %+v", got["port 80"])
	}
	if got["port 6550"].Detail != "published by container rancher (compose project local-cicd-sandbox)" || got["port 443"].State != CheckFail {
		t.Fatalf("ports: %+v %+v", got["port 6550"], got["port 443"])
	}
	conflicts := Conflicts(e)
	if len(conflicts) != 2 || conflicts[0].Container != "rancher" || len(conflicts[0].Ports) != 1 || conflicts[0].Ports[0] != 6550 ||
		conflicts[1].Container != "traefik" || len(conflicts[1].Ports) != 2 {
		t.Fatalf("conflicts %+v", conflicts)
	}
	if StopArgv(Conflict{Container: "x"})[1] != "stop" {
		t.Fatal("a container without compose project is stopped by name")
	}
}

func TestPreflight_OwnContainerHoldingThePortsIsFine(t *testing.T) {
	fakeMachine(t, "darwin", allTools, map[string]string{"docker info": "8 8345035776", "docker compose version": "v2",
		"docker ps": "go-dark-rail-rancher|go-dark-rail|0.0.0.0:80->30080/tcp, 0.0.0.0:443->30443/tcp, 127.0.0.1:6550->6443/tcp\n",
		// Real output: container port -> host address; host 80 is container 30080.
		"docker port": "6443/tcp -> 127.0.0.1:6550\n30080/tcp -> 0.0.0.0:80\n30080/tcp -> [::]:80\n30443/tcp -> 0.0.0.0:443\n30443/tcp -> [::]:443\n"}, "80", "443", "6550")
	checks := Preflight(e)
	if Blocking(checks) {
		t.Fatalf("repairing a running environment must pass: %+v", checks)
	}
	for _, port := range []string{"port 80", "port 443", "port 6550"} {
		if byName(checks)[port].Detail != "published by go-dark-rail-rancher" {
			t.Errorf("%s: %+v", port, byName(checks)[port])
		}
	}
}
