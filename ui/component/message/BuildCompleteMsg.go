package componentmessage

type BuildCompleteMsg struct {
	SessionID int
	Err       error
}
