package kube

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// fake replaces command execution: key is "name arg1 arg2...".
func fake(t *testing.T, responses map[string]string, failing ...string) {
	t.Helper()
	orig := runner
	t.Cleanup(func() { runner = orig })
	runner = func(_ context.Context, name string, args ...string) ([]byte, error) {
		key := strings.Join(append([]string{name}, args...), " ")
		for prefix, out := range responses {
			if strings.HasPrefix(key, prefix) {
				if slices.Contains(failing, prefix) {
					return []byte(out), errors.New("exit status 7")
				}
				return []byte(out), nil
			}
		}
		return nil, errors.New("unexpected command: " + key)
	}
}

const viewMinikube = "kubectl config view --minify --context minikube"

func TestGuard_AllowsMinikubeOnLoopback(t *testing.T) {
	fake(t, map[string]string{viewMinikube: "https://127.0.0.1:53846"})
	if err := Guard(Cluster{Context: "minikube", Namespace: "til-local"}); err != nil {
		t.Fatal(err)
	}
}

func TestGuard_AllowsMinikubeVMAddressOnly(t *testing.T) {
	fake(t, map[string]string{viewMinikube: "https://192.168.49.2:8443", "minikube ip": "192.168.49.2\n"})
	if err := Guard(Cluster{Context: "minikube", Namespace: "til-local"}); err != nil {
		t.Fatal(err)
	}
	fake(t, map[string]string{viewMinikube: "https://10.82.0.5:443", "minikube ip": "192.168.49.2\n"})
	if err := Guard(Cluster{Context: "minikube", Namespace: "til-local"}); err == nil {
		t.Fatal("a private address that isn't the minikube VM must be refused")
	}
}

func TestGuard_RefusesProductionAndHostnames(t *testing.T) {
	fake(t, map[string]string{
		"kubectl config view --minify --context azr-prd-idas-aks01": "https://rancher.intern-belgiantrain.be/k8s/clusters/c-dfjlf",
		viewMinikube: "https://rancher.intern-belgiantrain.be/k8s/clusters/c-dfjlf",
	})
	if err := Guard(Cluster{Context: "azr-prd-idas-aks01", Namespace: "til-local"}); err == nil || !strings.Contains(err.Error(), "only local contexts") {
		t.Fatalf("the production context must be refused by name, got %v", err)
	}
	// Even a context *named* minikube pointing at a remote hostname is refused.
	if err := Guard(Cluster{Context: "minikube", Namespace: "til-local"}); err == nil || !strings.Contains(err.Error(), "hostname") {
		t.Fatalf("a hostname server must be refused, got %v", err)
	}
	if err := Guard(Cluster{Context: "rancher-desktop", Namespace: "x"}); err == nil {
		t.Fatal("contexts outside the allowlist are refused")
	}
}

func TestGuard_RefusesBadNamespaceAndMissingContext(t *testing.T) {
	fake(t, map[string]string{viewMinikube: "https://127.0.0.1:1"})
	for _, ns := range []string{"", "a b", "kube/system"} {
		if Guard(Cluster{Context: "minikube", Namespace: ns}) == nil {
			t.Errorf("namespace %q must be refused", ns)
		}
	}
	fake(t, map[string]string{})
	if Guard(Cluster{Context: "minikube", Namespace: "x"}) == nil {
		t.Fatal("a missing context must be refused")
	}
}

func TestMinikubeStatus(t *testing.T) {
	fake(t, map[string]string{"minikube status": `{"Name":"minikube","Host":"Stopped","Kubelet":"Stopped","APIServer":"Stopped"}`}, "minikube status")
	if running, err := MinikubeStatus(); running || err != nil {
		t.Fatalf("stopped minikube (exit 7) is not an error: %v %v", running, err)
	}
	fake(t, map[string]string{"minikube status": `{"Name":"minikube","Host":"Running","Kubelet":"Running","APIServer":"Running"}`})
	if running, err := MinikubeStatus(); !running || err != nil {
		t.Fatalf("expected running, got %v %v", running, err)
	}
	// No profile yet: minikube exits non-zero without JSON; start creates it.
	orig := runner
	runner = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("* Profile \"minikube\" not found."), &exec.ExitError{}
	}
	if running, err := MinikubeStatus(); running || err != nil {
		t.Fatalf("a missing profile means stopped, got %v %v", running, err)
	}
	runner = orig

	fake(t, map[string]string{})
	if _, err := MinikubeStatus(); err == nil {
		t.Fatal("a missing minikube binary is an error")
	}
}

func TestDockerEnv(t *testing.T) {
	fake(t, map[string]string{"minikube docker-env": "DOCKER_TLS_VERIFY=1\nDOCKER_HOST=tcp://127.0.0.1:53841\nDOCKER_CERT_PATH=/h/.minikube/certs\nMINIKUBE_ACTIVE_DOCKERD=minikube\n# comment\n"})
	env, err := DockerEnv()
	if err != nil || len(env) != 4 || env[1] != "DOCKER_HOST=tcp://127.0.0.1:53841" {
		t.Fatalf("unexpected env %v, %v", env, err)
	}
}

func TestClusterArgsAlwaysNameContextAndNamespace(t *testing.T) {
	c := Cluster{Context: "minikube", Namespace: "til-local"}
	if got := strings.Join(c.Args("upgrade", "--install"), " "); got != "upgrade --install --kube-context minikube --namespace til-local" {
		t.Fatalf("helm args %q", got)
	}
	if got := strings.Join(c.KubectlArgs("get", "pods"), " "); got != "get pods --context minikube --namespace til-local" {
		t.Fatalf("kubectl args %q", got)
	}
}
