package model

import (
	"fmt"
	"sort"

	"github.com/exr462/go-dark/config"
)

// eligibleNodes narrows the full project registry down to only the entries
// that are (a) actually cloned/fetched locally and (b) selected by the given
// predicate (e.g. Buildable or Deployable), keyed by project name for O(1)
// graph lookups.
func eligibleNodes(activeProjects []config.Project, registry []config.AvailableProject, include func(config.AvailableProject) bool) map[string]config.AvailableProject {
	clonedMap := make(map[string]bool, len(activeProjects))
	for _, p := range activeProjects {
		clonedMap[p.Name] = p.Fetched
	}

	nodes := make(map[string]config.AvailableProject)
	for _, ap := range registry {
		if clonedMap[ap.Name] && include(ap) {
			nodes[ap.Name] = ap
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
// Node names within a wave are sorted for deterministic, reproducible output
// (useful for both rendering and unit testing).
func topoSortWaves(nodes map[string]config.AvailableProject) (flat []config.AvailableProject, waves [][]config.AvailableProject, err error) {
	adj := make(map[string][]string)
	inDegree := make(map[string]int, len(nodes))
	for name := range nodes {
		inDegree[name] = 0
	}

	for name, ap := range nodes {
		for _, dep := range ap.Dependencies {
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
		wave := make([]config.AvailableProject, 0, len(frontier))
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
		return nil, nil, fmt.Errorf("❌ Cyclic dependency or structural mapping issue detected in AvailableProjects metadata")
	}

	return flat, waves, nil
}

// ResolveBuildOrder maps your active project state against AvailableProjects
// metadata and returns a single flat, dependency-respecting build order for
// every cloned, Buildable component.
func ResolveBuildOrder(activeProjects []config.Project, registry []config.AvailableProject) ([]config.AvailableProject, error) {
	nodes := eligibleNodes(activeProjects, registry, func(ap config.AvailableProject) bool { return ap.Buildable })
	flat, _, err := topoSortWaves(nodes)
	return flat, err
}

// ResolveDeployOrder mirrors ResolveBuildOrder but operates on the
// Deployable flag instead of Buildable, and additionally returns the
// dependency "waves" so the deploy engine can roll out independent
// components concurrently - just like independent Kubernetes Deployments in
// the same namespace would.
func ResolveDeployOrder(activeProjects []config.Project, registry []config.AvailableProject) (flat []config.AvailableProject, waves [][]config.AvailableProject, err error) {
	nodes := eligibleNodes(activeProjects, registry, func(ap config.AvailableProject) bool { return ap.Deployable })
	return topoSortWaves(nodes)
}
