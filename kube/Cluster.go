// Package kube talks to the local Kubernetes cluster go-dark deploys to.
// Safety first: every command names the context explicitly and Guard refuses
// anything that isn't a known local cluster on a local address - the same
// kubeconfig usually also holds real (production) clusters.
package kube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// AllowedContexts are the only contexts go-dark will deploy to.
var AllowedContexts = []string{"minikube"}

// Cluster is the local deployment target.
type Cluster struct {
	Context   string
	Namespace string
}

// runner executes a command and returns its stdout; replaced in tests.
var runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

func run(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return runner(ctx, name, args...)
}

// Guard returns an error unless c is an allowlisted context whose API server
// is a literal loopback address or the minikube VM's own address. Hostnames
// are refused without being resolved: internal hostnames resolve to private
// addresses too.
func Guard(c Cluster) error {
	if !slices.Contains(AllowedContexts, c.Context) {
		return fmt.Errorf("refusing kube context %q: only local contexts %v are allowed", c.Context, AllowedContexts)
	}
	if c.Namespace == "" || strings.ContainsAny(c.Namespace, " /") {
		return fmt.Errorf("invalid namespace %q", c.Namespace)
	}
	out, err := run("kubectl", "config", "view", "--minify", "--context", c.Context, "-o", "jsonpath={.clusters[0].cluster.server}")
	if err != nil {
		return fmt.Errorf("kube context %q not found: %v", c.Context, err)
	}
	server := strings.TrimSpace(string(out))
	u, err := url.Parse(server)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("context %q has no usable server address (%q)", c.Context, server)
	}
	ip := net.ParseIP(u.Hostname())
	switch {
	case ip == nil:
		return fmt.Errorf("refusing context %q: server %s is a hostname, not a local address", c.Context, u.Hostname())
	case ip.IsLoopback():
		return nil
	case ip.IsPrivate() && ip.Equal(minikubeIP()):
		return nil
	}
	return fmt.Errorf("refusing context %q: server %s is not a local address", c.Context, u.Hostname())
}

func minikubeIP() net.IP {
	out, err := run("minikube", "ip")
	if err != nil {
		return nil
	}
	return net.ParseIP(strings.TrimSpace(string(out)))
}

// MinikubeStatus reports whether minikube's API server is running. A
// stopped minikube - or one whose profile doesn't exist yet, which
// `minikube start` creates - is not an error; a missing minikube binary is.
func MinikubeStatus() (running bool, err error) {
	out, err := run("minikube", "status", "-o", "json")
	var st struct{ Host, APIServer string }
	if jerr := json.Unmarshal(out, &st); jerr == nil {
		return st.Host == "Running" && st.APIServer == "Running", nil
	}
	var exitErr *exec.ExitError
	if err == nil || errors.As(err, &exitErr) {
		return false, nil
	}
	return false, fmt.Errorf("minikube not available: %v", err)
}

// DockerEnv returns the environment that points docker at minikube's
// daemon, so images built with it are available to the cluster without a
// registry (the local-rancher charts use imagePullPolicy: Never).
func DockerEnv() ([]string, error) {
	out, err := run("minikube", "docker-env", "--shell", "none")
	if err != nil {
		return nil, fmt.Errorf("minikube docker-env: %v", err)
	}
	var env []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if k, _, ok := strings.Cut(line, "="); ok && (strings.HasPrefix(k, "DOCKER_") || strings.HasPrefix(k, "MINIKUBE_")) {
			env = append(env, line)
		}
	}
	if len(env) == 0 {
		return nil, fmt.Errorf("minikube docker-env returned nothing")
	}
	return env, nil
}

// Args appends the explicit context and namespace flags every kubectl and
// helm call must carry.
func (c Cluster) Args(args ...string) []string {
	return append(args, "--kube-context", c.Context, "--namespace", c.Namespace)
}

// KubectlArgs is Args for kubectl (which spells the flag --context).
func (c Cluster) KubectlArgs(args ...string) []string {
	return append(args, "--context", c.Context, "--namespace", c.Namespace)
}
