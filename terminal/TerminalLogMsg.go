package terminal

type TerminalLogMsg struct {
	SessionID int
	Text      string
	IsErr     bool
	Done      bool
	Error     error
}
