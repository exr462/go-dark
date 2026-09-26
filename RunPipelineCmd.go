package main

import (
	"context"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/task"
)

// RunPipelineCmd handles dependency checking and concurrent orchestration
//
//goland:noinspection GoMixedReceiverTypes
func (m *appModel) RunPipelineCmd(tasks []*task.BuildTask, maxParallelism int) tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Mutex to protect task state writes across multiple goroutines
		var mu sync.Mutex

		// Create a semaphore channel to limit maximum parallel operations
		sem := make(chan struct{}, maxParallelism)

		// Map for lightning-fast state lookups
		taskMap := make(map[string]*task.BuildTask)
		for _, t := range tasks {
			taskMap[t.ProjectName] = t
		}

		// Channel to notify the coordinator whenever any task completes
		taskDoneChan := make(chan string, len(tasks))

		for {
			mu.Lock()
			activeCount := 0
			pendingCount := 0
			failedPipeline := false

			// Step A: Evaluate graph nodes
			for _, t := range tasks {
				if t.State == task.StateBuilding {
					activeCount++
					continue
				}
				if t.State == task.StateFailed {
					failedPipeline = true
					continue
				}
				if t.State != task.StatePending {
					continue
				}

				pendingCount++

				// Check if all prerequisites are fulfilled successfully
				dependenciesMet := true
				for _, depName := range t.Dependencies {
					depTask, exists := taskMap[depName]
					if !exists || depTask.State != task.StateSuccess {
						dependenciesMet = false
						break
					}
				}

				// Step B: Dispatch the build if prerequisites are clear
				if dependenciesMet {
					t.State = task.StateBuilding
					activeCount++
					pendingCount--
					wg.Add(1)

					// Dispatch build worker in a background goroutine
					go func(buildTask *task.BuildTask) {
						defer wg.Done()

						// Block until a concurrency slot opens up
						sem <- struct{}{}
						defer func() { <-sem }()

						// Trigger execution hook
						err := executeBuild(ctx, buildTask.Path, buildTask.BuildArgs)

						mu.Lock()
						if err != nil {
							buildTask.State = task.StateFailed
							buildTask.Error = err
						} else {
							buildTask.State = task.StateSuccess
						}
						mu.Unlock()

						// Wake up main graph loop to recalculate next tasks
						taskDoneChan <- buildTask.ProjectName
					}(t)
				}
			}
			mu.Unlock()

			// Step C: Check termination boundaries
			if failedPipeline {
				cancel() // Instantly kill remaining processes if a hard dependency fails
				wg.Wait()
				return task.PipelineCompleteMsg{Success: false}
			}

			if pendingCount == 0 && activeCount == 0 {
				break // Everything processed successfully
			}

			// Wait until an active build finishes before cycling the loop
			if activeCount > 0 {
				<-taskDoneChan
			} else if pendingCount > 0 && activeCount == 0 {
				// Deadlock safety fallback: dependencies are cyclic or broken
				return task.PipelineCompleteMsg{Success: false}
			}
		}

		wg.Wait()
		return task.PipelineCompleteMsg{Success: true}
	}
}
