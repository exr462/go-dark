package task

// TaskState tracks the current progress of a specific project build
type TaskState int

const (
	StatePending TaskState = iota
	StateBuilding
	StateSuccess
	StateFailed
)

type BuildTask struct {
	ProjectName  string
	Path         string
	BuildArgs    string   // e.g., "install -DskipTests"
	Dependencies []string // Names of projects that MUST build successfully first
	State        TaskState
	Error        error
}

// Bubble Tea messages to update your UI during the pipeline run
type PipelineTaskStartedMsg string
type PipelineTaskFinishedMsg struct {
	ProjectName string
	Err         error
}
type PipelineCompleteMsg struct {
	Success bool
}
