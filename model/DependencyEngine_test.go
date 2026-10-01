package model

import (
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
)

func cloned(names ...string) []config.Project {
	projects := make([]config.Project, len(names))
	for i, n := range names {
		projects[i] = config.Project{Name: n, Fetched: true}
	}
	return projects
}

func TestResolveBuildOrder_RespectsDependencies(t *testing.T) {
	registry := []config.AvailableProject{
		{Name: "parent", Buildable: true},
		{Name: "api", Buildable: true, Dependencies: []string{"parent"}},
		{Name: "service-a", Buildable: true, Dependencies: []string{"api"}},
		{Name: "service-b", Buildable: true, Dependencies: []string{"api"}},
	}

	order, err := ResolveBuildOrder(cloned("parent", "api", "service-a", "service-b"), registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 4 {
		t.Fatalf("expected 4 resolved projects, got %d (%v)", len(order), order)
	}

	pos := make(map[string]int, len(order))
	for i, p := range order {
		pos[p.Name] = i
	}
	if pos["parent"] > pos["api"] {
		t.Errorf("parent must build before api: order=%v", order)
	}
	if pos["api"] > pos["service-a"] || pos["api"] > pos["service-b"] {
		t.Errorf("api must build before its dependents: order=%v", order)
	}
}

func TestResolveBuildOrder_SkipsNotCloned(t *testing.T) {
	registry := []config.AvailableProject{
		{Name: "parent", Buildable: true},
		{Name: "api", Buildable: true, Dependencies: []string{"parent"}},
	}
	// "api" was never fetched/cloned locally, so it must be excluded.
	order, err := ResolveBuildOrder(cloned("parent"), registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 1 || order[0].Name != "parent" {
		t.Fatalf("expected only [parent], got %v", order)
	}
}

func TestResolveBuildOrder_SkipsNotBuildable(t *testing.T) {
	registry := []config.AvailableProject{
		{Name: "docs", Buildable: false},
	}
	order, err := ResolveBuildOrder(cloned("docs"), registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 0 {
		t.Fatalf("expected no buildable projects, got %v", order)
	}
}

func TestResolveBuildOrder_DetectsCycle(t *testing.T) {
	registry := []config.AvailableProject{
		{Name: "a", Buildable: true, Dependencies: []string{"b"}},
		{Name: "b", Buildable: true, Dependencies: []string{"a"}},
	}
	_, err := ResolveBuildOrder(cloned("a", "b"), registry)
	if err == nil {
		t.Fatal("expected a cyclic dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "Cyclic") {
		t.Errorf("expected cyclic dependency message, got: %v", err)
	}
}

func TestResolveDeployOrder_GroupsIndependentServicesIntoWaves(t *testing.T) {
	// Mirrors the real-world shape in config.AvailableProjects: several
	// independent deployable services sharing one non-deployable parent.
	registry := []config.AvailableProject{
		{Name: "contract-parent", Buildable: true, Deployable: false},
		{Name: "til-purchase", Buildable: true, Deployable: true, Dependencies: []string{"contract-parent"}},
		{Name: "til-product", Buildable: true, Deployable: true, Dependencies: []string{"contract-parent"}},
		{Name: "til-order", Buildable: true, Deployable: true, Dependencies: []string{"til-purchase"}},
	}
	active := cloned("contract-parent", "til-purchase", "til-product", "til-order")

	flat, waves, err := ResolveDeployOrder(active, registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// contract-parent is not Deployable, so it must not appear anywhere in
	// the deploy graph even though it's a dependency of other nodes.
	for _, p := range flat {
		if p.Name == "contract-parent" {
			t.Fatalf("non-deployable dependency leaked into deploy order: %v", flat)
		}
	}
	if len(flat) != 3 {
		t.Fatalf("expected 3 deployable projects, got %d (%v)", len(flat), flat)
	}

	if len(waves) != 2 {
		t.Fatalf("expected 2 waves (parallel purchase+product, then order), got %d: %v", len(waves), waves)
	}
	if len(waves[0]) != 2 {
		t.Fatalf("expected first wave to contain the 2 independent services, got %v", waves[0])
	}
	if len(waves[1]) != 1 || waves[1][0].Name != "til-order" {
		t.Fatalf("expected second wave to be [til-order], got %v", waves[1])
	}
}

func TestResolveDeployOrder_NoDeployableProjectsReturnsEmpty(t *testing.T) {
	registry := []config.AvailableProject{
		{Name: "lib", Buildable: true, Deployable: false},
	}
	flat, waves, err := ResolveDeployOrder(cloned("lib"), registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flat) != 0 || len(waves) != 0 {
		t.Fatalf("expected no deployable output, got flat=%v waves=%v", flat, waves)
	}
}
