package kube

import (
	"context"
	"errors"
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

const viewLocal = "kubectl config view --minify --context rancher-lcl"

func TestGuard_AllowsRancherOnLoopback(t *testing.T) {
	fake(t, map[string]string{viewLocal: "https://127.0.0.1:6550"})
	if err := Guard(Cluster{Context: "rancher-lcl", Namespace: "til-lcl"}); err != nil {
		t.Fatal(err)
	}
}

func TestGuard_RefusesNonLoopbackAddresses(t *testing.T) {
	for _, server := range []string{"https://172.19.0.250:6443", "https://10.82.0.5:443", "https://192.168.49.2:8443"} {
		fake(t, map[string]string{viewLocal: server})
		if err := Guard(Cluster{Context: "rancher-lcl", Namespace: "til-lcl"}); err == nil {
			t.Fatalf("%s is not loopback and must be refused", server)
		}
	}
}

func TestGuard_RefusesProductionAndHostnames(t *testing.T) {
	fake(t, map[string]string{
		"kubectl config view --minify --context azr-prd-idas-aks01": "https://rancher.intern-belgiantrain.be/k8s/clusters/c-dfjlf",
		viewLocal: "https://rancher.intern-belgiantrain.be/k8s/clusters/c-dfjlf",
	})
	if err := Guard(Cluster{Context: "azr-prd-idas-aks01", Namespace: "til-lcl"}); err == nil || !strings.Contains(err.Error(), "only local contexts") {
		t.Fatalf("the production context must be refused by name, got %v", err)
	}
	// Even a context *named* rancher-lcl pointing at a remote hostname is refused.
	if err := Guard(Cluster{Context: "rancher-lcl", Namespace: "til-lcl"}); err == nil || !strings.Contains(err.Error(), "hostname") {
		t.Fatalf("a hostname server must be refused, got %v", err)
	}
	for _, ctx := range []string{"rancher-desktop", "minikube"} {
		if err := Guard(Cluster{Context: ctx, Namespace: "x"}); err == nil {
			t.Fatalf("context %s is outside the allowlist and must be refused", ctx)
		}
	}
}

func TestGuard_RefusesBadNamespaceAndMissingContext(t *testing.T) {
	fake(t, map[string]string{viewLocal: "https://127.0.0.1:1"})
	for _, ns := range []string{"", "a b", "kube/system"} {
		if Guard(Cluster{Context: "rancher-lcl", Namespace: ns}) == nil {
			t.Errorf("namespace %q must be refused", ns)
		}
	}
	fake(t, map[string]string{})
	if Guard(Cluster{Context: "rancher-lcl", Namespace: "x"}) == nil {
		t.Fatal("a missing context must be refused")
	}
}

func TestStatus(t *testing.T) {
	c := Cluster{Context: "rancher-lcl", Namespace: "til-lcl", Container: "rancher"}
	inspect := "docker inspect -f {{.State.Running}} rancher"
	ready := "kubectl get --raw /readyz --context rancher-lcl"

	fake(t, map[string]string{inspect: "true\n", ready: "ok"})
	if running, err := Status(c); !running || err != nil {
		t.Fatalf("running container with a ready API: %v %v", running, err)
	}
	fake(t, map[string]string{inspect: "false\n"})
	if running, err := Status(c); running || err != nil {
		t.Fatalf("a stopped container is not an error: %v %v", running, err)
	}
	fake(t, map[string]string{inspect: "true\n", ready: ""}, ready)
	if running, err := Status(c); running || err != nil {
		t.Fatalf("a booting API is not running yet, not an error: %v %v", running, err)
	}
	fake(t, map[string]string{})
	if _, err := Status(c); err == nil || !strings.Contains(err.Error(), "sandbox") {
		t.Fatalf("a missing container is an error, got %v", err)
	}
}

func TestImportImageArgvPassesValuesAsArguments(t *testing.T) {
	argv := ImportImageArgv(Cluster{Container: "rancher"}, "local/til-order:1.0; rm -rf /")
	if argv[0] != "sh" || strings.Contains(argv[2], "til-order") || argv[4] != "local/til-order:1.0; rm -rf /" || argv[5] != "rancher" || argv[6] != ContainerdSocket {
		t.Fatalf("image and container must be positional args, not script text: %q", argv)
	}
}

func TestClusterArgsAlwaysNameContextAndNamespace(t *testing.T) {
	c := Cluster{Context: "rancher-lcl", Namespace: "til-lcl"}
	if got := strings.Join(c.Args("upgrade", "--install"), " "); got != "upgrade --install --kube-context rancher-lcl --namespace til-lcl" {
		t.Fatalf("helm args %q", got)
	}
	if got := strings.Join(c.KubectlArgs("get", "pods"), " "); got != "get pods --context rancher-lcl --namespace til-lcl" {
		t.Fatalf("kubectl args %q", got)
	}
}

// recorder fakes kubectl with a deployment list and records mutating calls.
func recorder(t *testing.T, deployments string) *[]string {
	t.Helper()
	var calls []string
	orig := runner
	t.Cleanup(func() { runner = orig })
	runner = func(_ context.Context, name string, args ...string) ([]byte, error) {
		key := strings.Join(append([]string{name}, args...), " ")
		switch {
		case strings.HasPrefix(key, "kubectl config view"):
			return []byte("https://127.0.0.1:53846"), nil
		case strings.HasPrefix(key, "kubectl get deployments"):
			return []byte(deployments), nil
		case strings.HasPrefix(key, "kubectl annotate"), strings.HasPrefix(key, "kubectl scale"):
			calls = append(calls, key)
			return nil, nil
		}
		return nil, errors.New("unexpected " + key)
	}
	return &calls
}

const twoDeployments = `{"items":[
 {"metadata":{"name":"til-order"},"spec":{"replicas":1}},
 {"metadata":{"name":"til-dtm"},"spec":{"replicas":2}},
 {"metadata":{"name":"til-idle"},"spec":{"replicas":0}}]}`

func TestScaleDownOthers_KeepsShippedAndRecordsReplicas(t *testing.T) {
	calls := recorder(t, twoDeployments)
	c := Cluster{Context: "rancher-lcl", Namespace: "til-lcl"}

	scaled, err := ScaleDownOthers(c, []string{"til-order"})

	if err != nil || len(scaled) != 1 || scaled[0] != "til-dtm" {
		t.Fatalf("only til-dtm should be scaled, got %v %v", scaled, err)
	}
	want := []string{
		"kubectl annotate deployment til-dtm --overwrite go-dark/replicas-before=2 --context rancher-lcl --namespace til-lcl",
		"kubectl scale deployment til-dtm --replicas=0 --context rancher-lcl --namespace til-lcl",
	}
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected calls:\n%s", strings.Join(*calls, "\n"))
	}
}

func TestRestoreScaled_PutsBackRecordedReplicas(t *testing.T) {
	calls := recorder(t, `{"items":[
	 {"metadata":{"name":"til-dtm","annotations":{"go-dark/replicas-before":"2"}},"spec":{"replicas":0}},
	 {"metadata":{"name":"til-order"},"spec":{"replicas":1}}]}`)
	c := Cluster{Context: "rancher-lcl", Namespace: "til-lcl"}

	restored, err := RestoreScaled(c)

	if err != nil || len(restored) != 1 || restored[0] != "til-dtm" {
		t.Fatalf("expected til-dtm restored, got %v %v", restored, err)
	}
	if (*calls)[0] != "kubectl scale deployment til-dtm --replicas=2 --context rancher-lcl --namespace til-lcl" ||
		(*calls)[1] != "kubectl annotate deployment til-dtm go-dark/replicas-before- --context rancher-lcl --namespace til-lcl" {
		t.Fatalf("unexpected calls %v", *calls)
	}
	if names, _ := ScaledDown(c); len(names) != 1 {
		t.Fatalf("ScaledDown lists annotated deployments, got %v", names)
	}
}

func TestScaleDownOthers_RefusesNonLocal(t *testing.T) {
	recorder(t, twoDeployments)
	if _, err := ScaleDownOthers(Cluster{Context: "azr-prd-idas-aks01", Namespace: "til"}, nil); err == nil {
		t.Fatal("scaling must go through the guard")
	}
	if _, err := RestoreScaled(Cluster{Context: "azr-prd-idas-aks01", Namespace: "til"}); err == nil {
		t.Fatal("restoring must go through the guard")
	}
}
