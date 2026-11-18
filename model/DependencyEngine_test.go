package model

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

func TestResolveBuildOrder_RespectsDependencies(t *testing.T) {
	registry := []config.Project{
		{Name: "parent", Buildable: true},
		{Name: "api", Buildable: true, DependsOn: []string{"parent"}},
		{Name: "service-a", Buildable: true, DependsOn: []string{"api"}},
		{Name: "service-b", Buildable: true, DependsOn: []string{"api"}},
	}

	order, err := ResolveBuildOrder(workspace(registry, "parent", "api", "service-a", "service-b"))
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
	registry := []config.Project{
		{Name: "parent", Buildable: true},
		{Name: "api", Buildable: true, DependsOn: []string{"parent"}},
	}
	// "api" was never fetched/cloned locally, so it must be excluded.
	order, err := ResolveBuildOrder(workspace(registry, "parent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 1 || order[0].Name != "parent" {
		t.Fatalf("expected only [parent], got %v", order)
	}
}

func TestResolveBuildOrder_SkipsNotBuildable(t *testing.T) {
	registry := []config.Project{
		{Name: "docs", Buildable: false},
	}
	order, err := ResolveBuildOrder(workspace(registry, "docs"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 0 {
		t.Fatalf("expected no buildable projects, got %v", order)
	}
}

func TestResolveBuildOrder_DetectsCycle(t *testing.T) {
	registry := []config.Project{
		{Name: "a", Buildable: true, DependsOn: []string{"b"}},
		{Name: "b", Buildable: true, DependsOn: []string{"a"}},
	}
	_, err := ResolveBuildOrder(workspace(registry, "a", "b"))
	if err == nil {
		t.Fatal("expected a cyclic dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "Cyclic") {
		t.Errorf("expected cyclic dependency message, got: %v", err)
	}
}

func TestResolveDeployOrder_GroupsIndependentServicesIntoWaves(t *testing.T) {
	// Mirrors the real-world belgiantrain shape: several
	// independent deployable services sharing one non-deployable parent.
	registry := []config.Project{
		{Name: "contract-parent", Buildable: true, Deployable: false},
		{Name: "til-purchase", Buildable: true, Deployable: true, DependsOn: []string{"contract-parent"}},
		{Name: "til-product", Buildable: true, Deployable: true, DependsOn: []string{"contract-parent"}},
		{Name: "til-order", Buildable: true, Deployable: true, DependsOn: []string{"til-purchase"}},
	}
	active := workspace(registry, "contract-parent", "til-purchase", "til-product", "til-order")

	flat, waves, err := ResolveDeployOrder(active)
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
	registry := []config.Project{
		{Name: "lib", Buildable: true, Deployable: false},
	}
	flat, waves, err := ResolveDeployOrder(workspace(registry, "lib"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flat) != 0 || len(waves) != 0 {
		t.Fatalf("expected no deployable output, got flat=%v waves=%v", flat, waves)
	}
}

func TestResolveBuildOrder_CrossTeamDependencies(t *testing.T) {
	projects := []config.Project{
		{ID: "rail/api", Name: "api", Buildable: true, Fetched: true, DependsOn: []string{"infra/base"}},
		{ID: "infra/base", Name: "base", Buildable: true, Fetched: true},
		{ID: "local/tool", Name: "tool", Buildable: true, Fetched: true, DependsOn: []string{"rail/api", "gone/missing"}},
	}

	order, err := ResolveBuildOrder(projects)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var ids []string
	for _, p := range order {
		ids = append(ids, p.ID)
	}
	if strings.Join(ids, ",") != "infra/base,rail/api,local/tool" {
		t.Fatalf("expected cross-team order base -> api -> tool (unknown deps ignored), got %v", ids)
	}
}

func TestResolveBuildOrder_SameNameInTwoTeamsAreDistinct(t *testing.T) {
	projects := []config.Project{
		{ID: "rail/common", Name: "common", Buildable: true, Fetched: true},
		{ID: "infra/common", Name: "common", Buildable: true, Fetched: true, DependsOn: []string{"rail/common"}},
	}
	order, err := ResolveBuildOrder(projects)
	if err != nil || len(order) != 2 || order[0].ID != "rail/common" {
		t.Fatalf("projects must be keyed by ID, got %v, %v", order, err)
	}
}
