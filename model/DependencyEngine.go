package model

import (
	"fmt"
	"sort"

	"github.com/exr462/go-dark/config"
)

// eligibleNodes narrows the workspace down to only the projects that are
// (a) actually cloned/fetched locally and (b) selected by the given
// predicate (e.g. Buildable or Deployable), keyed by project ID for O(1)
// graph lookups.
func eligibleNodes(projects []config.Project, include func(config.Project) bool) map[string]config.Project {
	nodes := make(map[string]config.Project)
	for _, p := range projects {
		if p.Fetched && include(p) {
			nodes[p.ID] = p
		}
	}
	return nodes
}

// topoSortWaves performs a breadth-first Kahn's-algorithm topological sort
// over the given eligible node set, grouping every node whose dependencies
// are already satisfied into the same "wave". Waves are what let the build
// pipeline and the deploy engine run independent components in parallel
// while still respecting ordering across dependent components - exactly how
// a real CI pipeline or a Kubernetes rollout of several independent
// Deployments would behave.
//
// Node IDs within a wave are sorted for deterministic, reproducible output
// (useful for both rendering and unit testing).
func topoSortWaves(nodes map[string]config.Project) (flat []config.Project, waves [][]config.Project, err error) {
	adj := make(map[string][]string)
	inDegree := make(map[string]int, len(nodes))
	for name := range nodes {
		inDegree[name] = 0
	}

	for name, p := range nodes {
		for _, dep := range p.DependsOn {
			if _, ok := nodes[dep]; ok {
				adj[dep] = append(adj[dep], name)
				inDegree[name]++
			}
		}
	}

	var frontier []string
	for name, deg := range inDegree {
		if deg == 0 {
			frontier = append(frontier, name)
		}
	}
	sort.Strings(frontier)

	for len(frontier) > 0 {
		wave := make([]config.Project, 0, len(frontier))
		for _, name := range frontier {
			wave = append(wave, nodes[name])
		}

		var next []string
		for _, name := range frontier {
			for _, neighbor := range adj[name] {
				inDegree[neighbor]--
				if inDegree[neighbor] == 0 {
					next = append(next, neighbor)
				}
			}
		}
		sort.Strings(next)

		flat = append(flat, wave...)
		waves = append(waves, wave)
		frontier = next
	}

	if len(flat) != len(nodes) {
		return nil, nil, fmt.Errorf("❌ Cyclic dependency or structural mapping issue detected between workspace projects")
	}

	return flat, waves, nil
}

// ResolveBuildOrder returns a single flat, dependency-respecting build order
// for every cloned, Buildable project. Dependencies on projects that aren't
// part of the build (not cloned, not buildable, unknown) are ignored.
func ResolveBuildOrder(projects []config.Project) ([]config.Project, error) {
	nodes := eligibleNodes(projects, func(p config.Project) bool { return p.Buildable })
	flat, _, err := topoSortWaves(nodes)
	return flat, err
}

// ResolveDeployOrder mirrors ResolveBuildOrder but operates on the
// Deployable flag instead of Buildable, and additionally returns the
// dependency "waves" so the deploy engine can roll out independent
// components concurrently - just like independent Kubernetes Deployments in
// the same namespace would.
func ResolveDeployOrder(projects []config.Project) (flat []config.Project, waves [][]config.Project, err error) {
	nodes := eligibleNodes(projects, func(p config.Project) bool { return p.Deployable })
	return topoSortWaves(nodes)
}
