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
