package initializer

import (
	"context"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/runner"
	"github.com/exr462/go-dark/session"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/task"
)

type initializerTriggerPipeline struct {
	UI *model.UI
}

func (m *initializerTriggerPipeline) OnAction() tea.Cmd {
	return func() tea.Msg {
		cfg := m.UI.Config
		shells := m.UI.Shells
		width, height := m.UI.TerminalPaneSize()
		// Resolve the exact build queue order from the workspace dependency graph
		sortedQueue, err := model.ResolveBuildOrder(cfg.Projects)

		if err != nil {
			return task.PipelineCompleteMsg{Success: false, Log: err.Error()}
		}

		// Initialize structural runtime build trackers
		var tasks []*task.BuildTask
		taskMap := make(map[string]*task.BuildTask)
		for _, sq := range sortedQueue {
			t := &task.BuildTask{
				Meta:  sq,
				Path:  sq.Path,
				State: task.StatePending,
			}
			tasks = append(tasks, t)
			taskMap[sq.ID] = t
		}

		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var mu sync.Mutex
		sem := make(chan struct{}, m.UI.MaxParallelism)
		taskDoneChan := make(chan string, len(tasks))

		for {
			mu.Lock()
			activeCount := 0
			pendingCount := 0
			pipelineFailed := false

			for _, t := range tasks {
				if t.State == task.StateBuilding {
					activeCount++
					continue
				}
				if t.State == task.StateFailed {
					pipelineFailed = true
					continue
				}
				if t.State != task.StatePending {
					continue
				}

				pendingCount++

				// Check prerequisites
				depsMet := true
				for _, depID := range t.Meta.DependsOn {
					depTask, exists := taskMap[depID]
					// If the dependency exists in our build tree but hasn't finished, wait.
					if exists && depTask.State != task.StateSuccess {
						depsMet = false
						break
					}
				}

				if depsMet {
					t.State = task.StateBuilding
					activeCount++
					pendingCount--
					wg.Add(1)

					go func(buildTask *task.BuildTask) {
						defer wg.Done()
						sem <- struct{}{}
						defer func() { <-sem }()

						// Execute compilation step
						buildErr := runPipelineBuild(ctx, shells, cfg, buildTask.Meta, width, height)

						mu.Lock()
						if buildErr != nil {
							buildTask.State = task.StateFailed
							buildTask.Error = buildErr
						} else {
							buildTask.State = task.StateSuccess
						}
						mu.Unlock()

						taskDoneChan <- buildTask.Meta.Name
					}(t)
				}
			}
			mu.Unlock()

			if pipelineFailed {
				cancel()
				wg.Wait()
				return task.PipelineCompleteMsg{Success: false, Log: "Pipeline aborted due to module build error."}
			}

			if pendingCount == 0 && activeCount == 0 {
				break
			}

			if activeCount > 0 {
				<-taskDoneChan
			}
		}

		wg.Wait()
		return task.PipelineCompleteMsg{Success: true, Log: "All buildable modules completed!"}
	}
}

// runPipelineBuild runs the project's pipeline command (by default the
// preset's, e.g. "mvn clean install -DskipTests" for maven) as a "pipeline"
// session, so each module's build can be watched in the session inspector.
// When the pipeline is cancelled the build is killed but its output kept.
// Projects with no pipeline command are treated as built. Without a session
// manager (tests) it runs the command directly.
func runPipelineBuild(ctx context.Context, shells *shell.Manager, cfg config.Config, p config.Project, width, height int) error {
	shellCmd := config.ResolveCommand(p, config.PipelineCommand)
	if shellCmd == "" {
		return nil
	}
	if shells == nil {
		return runner.Command(ctx, cfg, p, shellCmd).Run()
	}
	id, err := session.StartBuild(shells, cfg, p, config.PipelineCommand, session.TagPipeline, width, height, nil)
	if err != nil {
		return err
	}
	s := shells.Get(id)
	select {
	case <-s.Done():
		return s.ExitErr()
	case <-ctx.Done():
		_ = shells.Kill(id)
		return ctx.Err()
	}
}

func InitializeTriggerPipeline(ui *model.UI) Initializer {
	return &initializerTriggerPipeline{ui}
}
