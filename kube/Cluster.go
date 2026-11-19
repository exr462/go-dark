// Package kube talks to the local Kubernetes cluster go-dark deploys to.
// Safety first: every command names the context explicitly and Guard refuses
// anything that isn't a known local cluster on a local address - the same
// kubeconfig usually also holds real (production) clusters.
package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// AllowedContexts are the only contexts go-dark will deploy to: the local
// Rancher sandbox (k3s inside the `rancher` container, API published on
// 127.0.0.1).
var AllowedContexts = []string{"rancher-lcl"}

// Cluster is the local deployment target.
type Cluster struct {
	Context   string
	Namespace string
	Container string // docker container running the cluster
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
// is a literal loopback address. Hostnames are refused without being
// resolved: internal hostnames resolve to private addresses too.
func Guard(c Cluster) error {
	if !slices.Contains(AllowedContexts, c.Context) {
		return fmt.Errorf("refusing kube context %q: only local contexts %v are allowed", c.Context, AllowedContexts)
	}
	if c.Namespace == "" || strings.ContainsAny(c.Namespace, " /") {
		return fmt.Errorf("invalid namespace %q", c.Namespace)
	}
	out, err := run("kubectl", "config", "view", "--minify", "--context", c.Context, "-o", "jsonpath={.clusters[0].cluster.server}")
	if err != nil {
		return fmt.Errorf("kube context %q not found (create it with the sandbox's deploy/cluster.sh kubeconfig): %v", c.Context, err)
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
	}
	return fmt.Errorf("refusing context %q: server %s is not a loopback address", c.Context, u.Hostname())
}

// Status reports whether the cluster's container runs and its API answers.
// A stopped container is not an error (StartArgv starts it); a missing one
// or a missing docker is.
func Status(c Cluster) (running bool, err error) {
	out, err := run("docker", "inspect", "-f", "{{.State.Running}}", c.Container)
	if err != nil {
		return false, fmt.Errorf("docker container %q not found - is the local Rancher sandbox set up? (%v)", c.Container, err)
	}
	if strings.TrimSpace(string(out)) != "true" {
		return false, nil
	}
	if _, err := run("kubectl", "get", "--raw", "/readyz", "--context", c.Context); err != nil {
		return false, nil
	}
	return true, nil
}

// StartArgv starts the cluster's container.
func StartArgv(c Cluster) []string {
	return []string{"docker", "start", c.Container}
}

// ContainerdSocket is the k3s node's containerd inside the cluster container.
const ContainerdSocket = "/run/k3s/containerd/containerd.sock"

// ImportImageArgv copies a locally built image into the cluster node's
// containerd, so pods find it without a registry (imagePullPolicy: Never).
// Rancher's embedded k3s has no `k3s ctr`; the sandbox's cluster.sh installs
// a standalone ctr into the container.
func ImportImageArgv(c Cluster, image string) []string {
	script := `command -v docker >/dev/null || { echo "docker not found"; exit 1; }
docker exec "$2" sh -c 'command -v ctr' >/dev/null || { echo "no ctr in container $2 - run the sandbox's deploy/cluster.sh up"; exit 1; }
docker save "$1" | docker exec -i "$2" ctr -a "$3" -n k8s.io images import -`
	return []string{"sh", "-c", script, "go-dark", image, c.Container, ContainerdSocket}
}

// NodeArch is the architecture of the cluster's node (charts pin
// kubernetes.io/arch in their nodeSelector).
func NodeArch(c Cluster) (string, error) {
	out, err := run("kubectl", "get", "nodes", "--context", c.Context, "-o", "jsonpath={.items[0].status.nodeInfo.architecture}")
	if err != nil {
		return "", fmt.Errorf("kubectl get nodes: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
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
