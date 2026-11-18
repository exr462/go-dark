//go:build !windows

package initializer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/task"
)

func pipelineUI(t *testing.T, projects ...config.Project) *model.UI {
	t.Helper()
	ui := &model.UI{Shells: shell.NewManager(), MaxParallelism: 2, WindowWidth: 100, WindowHeight: 30}
	t.Cleanup(ui.Shells.Shutdown)
	go func() {
		for {
			ui.Shells.Listen()()
		}
	}()
	for i := range projects {
		projects[i].Path = t.TempDir()
		projects[i].Fetched, projects[i].Buildable = true, true
	}
	ui.Config.Projects = projects
	return ui
}

func TestPipeline_BuildsInDependencyOrderAsInspectableSessions(t *testing.T) {
	order := filepath.Join(t.TempDir(), "order")
	cmd := func(name string) map[string]string {
		return map[string]string{config.PipelineCommand: "echo " + name + " >> " + order}
	}
	ui := pipelineUI(t,
		config.Project{ID: "t/app", Name: "app", DependsOn: []string{"t/lib"}, Commands: cmd("app")},
		config.Project{ID: "t/lib", Name: "lib", Commands: cmd("lib")},
	)

	msg := InitializeTriggerPipeline(ui).OnAction()()

	if done, ok := msg.(task.PipelineCompleteMsg); !ok || !done.Success {
		t.Fatalf("expected a successful pipeline, got %#v", msg)
	}
	got, _ := os.ReadFile(order)
	if strings.Fields(string(got))[0] != "lib" {
		t.Fatalf("lib must build before app, got %q", got)
	}
	var tags []string
	for _, s := range ui.Shells.List() {
		tags = append(tags, s.Tag+":"+s.ProjectName)
	}
	if len(tags) != 2 || !strings.Contains(strings.Join(tags, " "), "pipeline:lib") {
		t.Fatalf("each module must run as a pipeline session, got %v", tags)
	}
}

func TestPipeline_FailureAbortsAndKeepsLogs(t *testing.T) {
	ui := pipelineUI(t,
		config.Project{ID: "t/bad", Name: "bad", Commands: map[string]string{config.PipelineCommand: "echo broken-build; exit 1"}},
		config.Project{ID: "t/after", Name: "after", DependsOn: []string{"t/bad"}, Commands: map[string]string{config.PipelineCommand: "true"}},
	)

	msg := InitializeTriggerPipeline(ui).OnAction()()

	if done, ok := msg.(task.PipelineCompleteMsg); !ok || done.Success {
		t.Fatalf("expected a failed pipeline, got %#v", msg)
	}
	list := ui.Shells.List()
	if len(list) != 1 || !strings.Contains(list[0].Text(), "broken-build") {
		t.Fatalf("the failed build's log must stay inspectable and dependents must not run, got %v", list)
	}
}
