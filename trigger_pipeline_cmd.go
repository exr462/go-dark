package main

import (
	"context"
	"os"
	"os/exec"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

type TaskState int

const (
	StatePending TaskState = iota
	StateBuilding
	StateSuccess
	StateFailed
)

type BuildTask struct {
	Meta  config.AvailableProject
	Path  string
	State TaskState
	Err   error
}

// TriggerPipelineCmd spins up the concurrency workers
//
//goland:noinspection GoMixedReceiverTypes
func (m *appModel) TriggerPipelineCmd(maxParallelism int) tea.Cmd {
	return func() tea.Msg {
		// Resolve the exact build queue order based on your registry
		sortedQueue, err := model.ResolveBuildOrder(m.state.Config.Projects, config.AvailableProjects)
		if err != nil {
			return PipelineCompleteMsg{Success: false, Log: err.Error()}
		}

		// Map to coordinate lookups of current paths inside Config.Projects
		pathMap := make(map[string]string)
		for _, p := range m.state.Config.Projects {
			pathMap[p.Name] = p.Path
		}

		// Initialize structural runtime build trackers
		var tasks []*BuildTask
		taskMap := make(map[string]*BuildTask)
		for _, sq := range sortedQueue {
			t := &BuildTask{
				Meta:  sq,
				Path:  pathMap[sq.Name],
				State: StatePending,
			}
			tasks = append(tasks, t)
			taskMap[sq.Name] = t
		}

		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var mu sync.Mutex
		sem := make(chan struct{}, maxParallelism)
		taskDoneChan := make(chan string, len(tasks))

		for {
			mu.Lock()
			activeCount := 0
			pendingCount := 0
			pipelineFailed := false

			for _, t := range tasks {
				if t.State == StateBuilding {
					activeCount++
					continue
				}
				if t.State == StateFailed {
					pipelineFailed = true
					continue
				}
				if t.State != StatePending {
					continue
				}

				pendingCount++

				// Check prerequisites
				depsMet := true
				for _, depName := range t.Meta.Dependencies {
					depTask, exists := taskMap[depName]
					// If the dependency exists in our build tree but hasn't finished, wait.
					if exists && depTask.State != StateSuccess {
						depsMet = false
						break
					}
				}

				if depsMet {
					t.State = StateBuilding
					activeCount++
					pendingCount--
					wg.Add(1)

					go func(task *BuildTask) {
						defer wg.Done()
						sem <- struct{}{}
						defer func() { <-sem }()

						// Execute compilation step
						buildErr := runMavenBuild(ctx, task.Path)

						mu.Lock()
						if buildErr != nil {
							task.State = StateFailed
							task.Err = buildErr
						} else {
							task.State = StateSuccess
						}
						mu.Unlock()

						taskDoneChan <- task.Meta.Name
					}(t)
				}
			}
			mu.Unlock()

			if pipelineFailed {
				cancel()
				wg.Wait()
				return PipelineCompleteMsg{Success: false, Log: "Pipeline aborted due to module build error."}
			}

			if pendingCount == 0 && activeCount == 0 {
				break
			}

			if activeCount > 0 {
				<-taskDoneChan
			}
		}

		wg.Wait()
		return PipelineCompleteMsg{Success: true, Log: "All buildable modules completed!"}
	}
}

func runMavenBuild(ctx context.Context, directory string) error {
	// Standard safe compilation pass args
	cmd := exec.CommandContext(ctx, "mvn", "clean", "install", "-DskipTests")
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd.Run()
}

type PipelineCompleteMsg struct {
	Success bool
	Log     string
}
