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
	ActiveProjectID  string   // ID of the project whose dependencies are being edited
	Cursor           int      // The list cursor for choosing dependencies
	AvailableOptions []string // IDs of every other workspace project
	Original         []string // DependsOn before editing, restored on cancel
}

// NewDependencyScreen prepares the dependency editor for the given project.
func NewDependencyScreen(active config.Project, allProjects []config.Project) DependencyScreenState {
	var options []string
	for _, p := range allProjects {
		// Prevent a project from assigning itself as a dependency
		if p.ID != active.ID {
			options = append(options, p.ID)
		}
	}
	return DependencyScreenState{
		ActiveProjectID:  active.ID,
		AvailableOptions: options,
		Original:         append([]string(nil), active.DependsOn...),
	}
}

func AutoDiscoverPomDependencies(projectPath string, projects []config.Project) []string {
	var discovered []string
	content, err := os.ReadFile(filepath.Join(projectPath, "pom.xml"))
	if err != nil {
		return discovered
	}

	pomStr := string(content)
	for _, p := range projects {
		// Scans the raw text for artifactId references pointing to local components
		if strings.Contains(pomStr, fmt.Sprintf("<artifactId>%s</artifactId>", p.Name)) {
			discovered = append(discovered, p.ID)
		}
	}
	return discovered
}
