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

// ReplicasBeforeAnnotation records a deployment's replica count before
// go-dark scaled it to 0, so RestoreScaled can put it back.
const ReplicasBeforeAnnotation = "go-dark/replicas-before"

type deploymentList struct {
	Items []struct {
		Metadata struct {
			Name        string            `json:"name"`
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			Replicas int `json:"replicas"`
		} `json:"spec"`
	} `json:"items"`
}

func (c Cluster) deployments() (deploymentList, error) {
	var list deploymentList
	out, err := run("kubectl", c.KubectlArgs("get", "deployments", "-o", "json")...)
	if err != nil {
		return list, fmt.Errorf("kubectl get deployments: %v", err)
	}
	return list, json.Unmarshal(out, &list)
}

// ScaleDownOthers frees the local node for a deployment: every deployment
// in the namespace that isn't in keep and has replicas is scaled to 0, its
// previous replica count stored in an annotation. It returns what it scaled.
func ScaleDownOthers(c Cluster, keep []string) ([]string, error) {
	if err := Guard(c); err != nil {
		return nil, err
	}
	list, err := c.deployments()
	if err != nil {
		return nil, err
	}
	var scaled []string
	for _, d := range list.Items {
		name := d.Metadata.Name
		if slices.Contains(keep, name) || d.Spec.Replicas == 0 {
			continue
		}
		if _, err := run("kubectl", c.KubectlArgs("annotate", "deployment", name, "--overwrite",
			fmt.Sprintf("%s=%d", ReplicasBeforeAnnotation, d.Spec.Replicas))...); err != nil {
			return scaled, fmt.Errorf("annotate %s: %v", name, err)
		}
		if _, err := run("kubectl", c.KubectlArgs("scale", "deployment", name, "--replicas=0")...); err != nil {
			return scaled, fmt.Errorf("scale %s: %v", name, err)
		}
		scaled = append(scaled, name)
	}
	return scaled, nil
}

// RestoreScaled scales every deployment go-dark scaled down back to its
// recorded replica count and removes the annotation.
func RestoreScaled(c Cluster) ([]string, error) {
	if err := Guard(c); err != nil {
		return nil, err
	}
	list, err := c.deployments()
	if err != nil {
		return nil, err
	}
	var restored []string
	for _, d := range list.Items {
		before, ok := d.Metadata.Annotations[ReplicasBeforeAnnotation]
		if !ok {
			continue
		}
		name := d.Metadata.Name
		if _, err := run("kubectl", c.KubectlArgs("scale", "deployment", name, "--replicas="+before)...); err != nil {
			return restored, fmt.Errorf("scale %s: %v", name, err)
		}
		if _, err := run("kubectl", c.KubectlArgs("annotate", "deployment", name, ReplicasBeforeAnnotation+"-")...); err != nil {
			return restored, fmt.Errorf("annotate %s: %v", name, err)
		}
		restored = append(restored, name)
	}
	return restored, nil
}

// ScaledDown lists the deployments go-dark has scaled to 0.
func ScaledDown(c Cluster) ([]string, error) {
	list, err := c.deployments()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, d := range list.Items {
		if _, ok := d.Metadata.Annotations[ReplicasBeforeAnnotation]; ok {
			names = append(names, d.Metadata.Name)
		}
	}
	return names, nil
}
