package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/config"
)

const StateDependencyConfig = 99

type DependencyScreenState struct {
	ActiveProjectIndex int      // The index of the project we are modifying
	Cursor             int      // The list cursor for choosing dependencies
	AvailableOptions   []string // List of all possible candidate names scanned from your workspace
}

// NewDependencyScreen Helper initialization constructor
func NewDependencyScreen(activeProjectName string, allProjects []config.Project) DependencyScreenState {
	var options []string
	for _, p := range allProjects {
		// Prevent a project from assigning itself as a dependency
		if p.Name != activeProjectName {
			options = append(options, p.Name)
		}
	}
	return DependencyScreenState{
		AvailableOptions: options,
	}
}

func AutoDiscoverPomDependencies(projectPath string, allAvailable []config.AvailableProject) []string {
	var discovered []string
	content, err := os.ReadFile(filepath.Join(projectPath, "pom.xml"))
	if err != nil {
		return discovered
	}

	pomStr := string(content)
	for _, ap := range allAvailable {
		// Scans the raw text for artifactId references pointing to local components
		if strings.Contains(pomStr, fmt.Sprintf("<artifactId>%s</artifactId>", ap.Name)) {
			discovered = append(discovered, ap.Name)
		}
	}
	return discovered
}

// ResolveBuildOrder maps your active project state against AvailableProjects metadata
func ResolveBuildOrder(activeProjects []config.Project, registry []config.AvailableProject) ([]config.AvailableProject, error) {
	// 1. Create a fast lookup map for what is cloned/available locally
	clonedMap := make(map[string]bool)
	for _, p := range activeProjects {
		clonedMap[p.Name] = p.Fetched
	}

	// 2. Build a local map of registry records that are cloned AND buildable
	registryMap := make(map[string]config.AvailableProject)
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, ap := range registry {
		// Only consider it if it's cloned locally and marked as buildable
		if clonedMap[ap.Name] && ap.Buildable {
			registryMap[ap.Name] = ap
			inDegree[ap.Name] = 0
		}
	}

	// 3. Build the Dependency Graph
	for name, ap := range registryMap {
		for _, dep := range ap.Dependencies {
			// Only map dependencies that are actually cloned and part of our execution build map
			if _, ok := registryMap[dep]; ok {
				adj[dep] = append(adj[dep], name)
				inDegree[name]++
			}
		}
	}

	// 4. Find root nodes (in-degree == 0)
	var queue []string
	for name := range registryMap {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}

	var sortedOrder []config.AvailableProject

	// 5. Process nodes sequentially
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		sortedOrder = append(sortedOrder, registryMap[curr])

		for _, neighbor := range adj[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Safety: Detect cyclic loops or orphan nodes
	if len(sortedOrder) != len(registryMap) {
		return nil, fmt.Errorf("❌ Cyclic dependency or structural mapping issue detected in AvailableProjects metadata")
	}

	return sortedOrder, nil
}
