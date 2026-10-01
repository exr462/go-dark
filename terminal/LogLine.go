package terminal

type LogLine struct {
	Text  string
	IsErr bool

	// IsPrompt marks a synthetic "prompt + command" echo line (not real
	// process output) so the renderer can style it like a shell prompt.
	IsPrompt bool
}
