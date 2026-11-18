package deploy

import (
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
)

// workspace turns registry-style test fixtures into workspace projects: IDs are
// the plain names (so DependsOn can reference them directly) and only the
// listed projects count as cloned.
func workspace(projects []config.Project, cloned ...string) []config.Project {
	isCloned := make(map[string]bool, len(cloned))
	for _, n := range cloned {
		isCloned[n] = true
	}
	out := make([]config.Project, len(projects))
	for i, p := range projects {
		p.ID = p.Name
		p.Fetched = isCloned[p.Name]
		out[i] = p
	}
	return out
}

func TestPlan_UsesDefaultNamespaceWhenEmpty(t *testing.T) {
	registry := []config.Project{{Name: "svc", Deployable: true}}
	entries, err := Plan(workspace(registry, "svc"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(entries[0].Text, DefaultNamespace) {
		t.Fatalf("expected first line to mention default namespace %q, got: %q", DefaultNamespace, entries[0].Text)
	}
}

func TestPlan_NoDeployableProjectsReturnsError(t *testing.T) {
	registry := []config.Project{{Name: "lib", Deployable: false}}
	_, err := Plan(workspace(registry, "lib"), "ns")
	if err == nil {
		t.Fatal("expected an error when no deployable projects are available")
	}
}

func TestPlan_PropagatesDependencyCycleError(t *testing.T) {
	registry := []config.Project{
		{Name: "a", Deployable: true, DependsOn: []string{"b"}},
		{Name: "b", Deployable: true, DependsOn: []string{"a"}},
	}
	_, err := Plan(workspace(registry, "a", "b"), "ns")
	if err == nil {
		t.Fatal("expected a cyclic dependency error to propagate from model.ResolveDeployOrder")
	}
}

func TestPlan_EmitsStepsPerWaveInDependencyOrder(t *testing.T) {
	registry := []config.Project{
		{Name: "parent", Deployable: false, Buildable: true},
		{Name: "til-purchase", Deployable: true, DependsOn: []string{"parent"}},
		{Name: "til-order", Deployable: true, DependsOn: []string{"til-purchase"}},
	}
	entries, err := Plan(workspace(registry, "parent", "til-purchase", "til-order"), "test-ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var purchaseIdx, orderIdx = -1, -1
	for i, e := range entries {
		if strings.Contains(e.Text, "[til-purchase] Deployment healthy") {
			purchaseIdx = i
		}
		if strings.Contains(e.Text, "[til-order] Rendering Deployment/Service manifests") {
			orderIdx = i
		}
	}
	if purchaseIdx == -1 || orderIdx == -1 {
		t.Fatalf("expected both services to appear in the transcript, got: %v", entries)
	}
	if purchaseIdx >= orderIdx {
		t.Errorf("til-purchase must fully roll out before til-order starts: purchaseIdx=%d orderIdx=%d", purchaseIdx, orderIdx)
	}

	last := entries[len(entries)-1]
	if !strings.Contains(last.Text, "complete") {
		t.Errorf("expected transcript to end with a completion banner, got: %q", last.Text)
	}
}

func TestLines_FlattensTranscriptText(t *testing.T) {
	entries := []LogEntry{{Text: "one"}, {Text: "two"}}
	lines := Lines(entries)
	if len(lines) != 2 || lines[0] != "one" || lines[1] != "two" {
		t.Fatalf("unexpected flattened lines: %v", lines)
	}
}

func TestKebabCase(t *testing.T) {
	cases := map[string]string{
		"til-purchase": "til-purchase",
		"TIL_Product":  "til-product",
		"Service Name": "service-name",
	}
	for in, want := range cases {
		if got := kebabCase(in); got != want {
			t.Errorf("kebabCase(%q) = %q, want %q", in, got, want)
		}
	}
}
