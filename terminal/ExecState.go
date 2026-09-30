package terminal

type ExecState int

const (
	ExecIdle ExecState = iota
	ExecRunning
	ExecCompleted
	ExecFailed
)
