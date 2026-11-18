package config

import (
	"sort"
	"strings"
)

// PipelineCommand is the command key the startup build pipeline runs for
// every buildable project.
const PipelineCommand = "pipeline"

// Preset is the default command set for one kind of project. Order is the
// order in which commands are offered in the build modal.
type Preset struct {
	Order    []string
	Commands map[string]string
}

// Presets maps a project type to its defaults. "{name}" in a command is
// replaced with the lower-cased project name.
var Presets = map[string]Preset{
	"maven": {
		Order: []string{"clean", "test", "compile", "package", "install-skip-tests", "install"},
		Commands: map[string]string{
			"clean":              "mvn clean",
			"test":               "mvn clean test",
			"compile":            "mvn clean compile",
			"package":            "mvn clean package",
			"install-skip-tests": "mvn clean install -DskipTests",
			"install":            "mvn clean install",
			PipelineCommand:      "mvn clean install -DskipTests",
		},
	},
	"gradle": {
		Order: []string{"clean", "test", "build", "build-skip-tests"},
		Commands: map[string]string{
			"clean":            "./gradlew clean",
			"test":             "./gradlew test",
			"build":            "./gradlew build",
			"build-skip-tests": "./gradlew build -x test",
			PipelineCommand:    "./gradlew build -x test",
		},
	},
	"npm": {
		Order: []string{"install", "test", "build"},
		Commands: map[string]string{
			"install":       "npm ci",
			"test":          "npm test",
			"build":         "npm run build",
			PipelineCommand: "npm ci && npm run build",
		},
	},
	"go": {
		Order: []string{"build", "test", "vet"},
		Commands: map[string]string{
			"build":         "go build ./...",
			"test":          "go test ./...",
			"vet":           "go vet ./...",
			PipelineCommand: "go build ./...",
		},
	},
	"docker": {
		Order: []string{"build"},
		Commands: map[string]string{
			"build":         "docker build -t {name}:latest .",
			PipelineCommand: "docker build -t {name}:latest .",
		},
	},
}

// PresetNames lists the known project types in a stable order.
var PresetNames = []string{"maven", "gradle", "npm", "go", "docker"}

// CommandKeys returns the commands a project offers, preset order first,
// followed by any extra keys the project defines itself (sorted). The
// internal pipeline key is never listed.
func CommandKeys(p Project) []string {
	seen := make(map[string]bool)
	var keys []string
	for _, k := range Presets[p.Type].Order {
		seen[k] = true
		keys = append(keys, k)
	}
	var extra []string
	for k := range p.Commands {
		if !seen[k] && k != PipelineCommand {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	return append(keys, extra...)
}

// ResolveCommand returns the shell command for key: the project's own
// command if set, otherwise the preset's. Empty means not available.
func ResolveCommand(p Project, key string) string {
	cmd, ok := p.Commands[key]
	if !ok {
		cmd = Presets[p.Type].Commands[key]
	}
	return strings.ReplaceAll(cmd, "{name}", strings.ToLower(p.Name))
}

// UsesJVM reports whether JDK/Maven profiles apply to this project type.
func UsesJVM(p Project) bool {
	return p.Type == "maven" || p.Type == "gradle"
}
