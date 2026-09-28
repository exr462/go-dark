package initializer

import (
	"context"
	"os"
	"os/exec"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/task"
)

type initializerTriggerPipeline struct {
	ui *model.UI
}

func (m *initializerTriggerPipeline) OnAction() tea.Cmd {
	return func() tea.Msg {
		// Resolve the exact build queue order based on your registry
		sortedQueue, err := model.ResolveBuildOrder(m.ui.Config.Projects, config.AvailableProjects)

		if err != nil {
			return task.PipelineCompleteMsg{Success: false, Log: err.Error()}
		}
		// Map to coordinate lookups of current paths inside Config.Projects
		pathMap := make(map[string]string)
		for _, p := range m.ui.Config.Projects {
			pathMap[p.Name] = p.Path
		}

		// Initialize structural runtime build trackers
		var tasks []*task.BuildTask
		taskMap := make(map[string]*task.BuildTask)
		for _, sq := range sortedQueue {
			t := &task.BuildTask{
				Meta:  sq,
				Path:  pathMap[sq.Name],
				State: task.StatePending,
			}
			tasks = append(tasks, t)
			taskMap[sq.Name] = t
		}

		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var mu sync.Mutex
		sem := make(chan struct{}, m.ui.MaxParallelism)
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
				for _, depName := range t.Meta.Dependencies {
					depTask, exists := taskMap[depName]
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
						buildErr := runMavenBuild(ctx, buildTask.Path)

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

func runMavenBuild(ctx context.Context, directory string) error {
	// Standard safe compilation pass args
	cmd := exec.CommandContext(ctx, "mvn", "clean", "install", "-DskipTests")
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd.Run()
}

func InitializeTriggerPipeline(ui *model.UI) Initializer {
	return &initializerTriggerPipeline{ui}
}
