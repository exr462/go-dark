package terminal

type TerminalLogMsg struct {
	Text  string
	IsErr bool
	Done  bool
	Error error
}
