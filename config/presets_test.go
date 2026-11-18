package config

import (
	"slices"
	"strings"
	"testing"
)

func TestResolveCommand_ProjectOverridesPreset(t *testing.T) {
	p := Project{Name: "Api", Type: "maven", Commands: map[string]string{"test": "mvn verify"}}
	if got := ResolveCommand(p, "test"); got != "mvn verify" {
		t.Errorf("project command must override preset, got %q", got)
	}
	if got := ResolveCommand(p, "install"); got != "mvn clean install" {
		t.Errorf("preset command expected, got %q", got)
	}
	if got := ResolveCommand(p, "deploy"); got != "" {
		t.Errorf("unknown command must resolve to empty, got %q", got)
	}
}

func TestResolveCommand_SubstitutesName(t *testing.T) {
	p := Project{Name: "Til-Order", Type: "docker"}
	if got := ResolveCommand(p, "build"); got != "docker build -t til-order:latest ." {
		t.Errorf("unexpected docker command %q", got)
	}
}

func TestMavenPreset_DoesNotDoubleClean(t *testing.T) {
	for key, cmd := range Presets["maven"].Commands {
		if strings.Count(cmd, "clean") > 1 {
			t.Errorf("maven %s runs clean twice: %q", key, cmd)
		}
	}
}

func TestCommandKeys_PresetOrderThenExtrasWithoutPipeline(t *testing.T) {
	p := Project{Type: "go", Commands: map[string]string{"lint": "golangci-lint run", "e2e": "make e2e", PipelineCommand: "make"}}
	want := []string{"build", "test", "vet", "e2e", "lint"}
	if got := CommandKeys(p); !slices.Equal(got, want) {
		t.Errorf("CommandKeys = %v, want %v", got, want)
	}
}

func TestEveryPresetHasAPipelineCommand(t *testing.T) {
	for _, name := range PresetNames {
		if ResolveCommand(Project{Type: name}, PipelineCommand) == "" {
			t.Errorf("preset %s has no pipeline command", name)
		}
	}
}
