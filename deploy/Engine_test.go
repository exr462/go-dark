package deploy

import (
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
)

func clonedProjects(names ...string) []config.Project {
	projects := make([]config.Project, len(names))
	for i, n := range names {
		projects[i] = config.Project{Name: n, Fetched: true}
	}
	return projects
}

func TestPlan_UsesDefaultNamespaceWhenEmpty(t *testing.T) {
	registry := []config.AvailableProject{{Name: "svc", Deployable: true}}
	entries, err := Plan(clonedProjects("svc"), registry, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(entries[0].Text, DefaultNamespace) {
		t.Fatalf("expected first line to mention default namespace %q, got: %q", DefaultNamespace, entries[0].Text)
	}
}

func TestPlan_NoDeployableProjectsReturnsError(t *testing.T) {
	registry := []config.AvailableProject{{Name: "lib", Deployable: false}}
	_, err := Plan(clonedProjects("lib"), registry, "ns")
	if err == nil {
		t.Fatal("expected an error when no deployable projects are available")
	}
}

func TestPlan_PropagatesDependencyCycleError(t *testing.T) {
	registry := []config.AvailableProject{
		{Name: "a", Deployable: true, Dependencies: []string{"b"}},
		{Name: "b", Deployable: true, Dependencies: []string{"a"}},
	}
	_, err := Plan(clonedProjects("a", "b"), registry, "ns")
	if err == nil {
		t.Fatal("expected a cyclic dependency error to propagate from model.ResolveDeployOrder")
	}
}

func TestPlan_EmitsStepsPerWaveInDependencyOrder(t *testing.T) {
	registry := []config.AvailableProject{
		{Name: "parent", Deployable: false, Buildable: true},
		{Name: "til-purchase", Deployable: true, Dependencies: []string{"parent"}},
		{Name: "til-order", Deployable: true, Dependencies: []string{"til-purchase"}},
	}
	entries, err := Plan(clonedProjects("parent", "til-purchase", "til-order"), registry, "test-ns")
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
