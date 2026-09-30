package terminal

type ExecMsg struct {
	Line  LogLine
	Done  bool
	Error error
}
