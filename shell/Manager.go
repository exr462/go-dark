// Package shell runs processes inside pseudo-terminals and keeps a virtual
// terminal screen (charmbracelet/x/vt) per session, so go-dark can embed
// real interactive shells - and long-running builds - in its UI.
//
// Sessions outlive the screen that shows them: the Manager owns every
// process and only tells the UI that something changed (OutputMsg) or ended
// (ExitedMsg). The UI pulls the screen contents with Session.Render.
package shell

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

// ErrUnsupported is returned by Start on platforms without PTY support.
var ErrUnsupported = errors.New("embedded terminals are not supported on this platform")

// scrollbackLines is how much history each session keeps above its screen.
const scrollbackLines = 5000

// OutputMsg tells the UI a session's screen changed. At most one is pending
// per session at a time, so a chatty process (e.g. a Maven build) can't
// flood the update loop: the UI renders the latest state whenever it gets one.
type OutputMsg struct{ ID int }

// ExitedMsg tells the UI a session's process ended.
type ExitedMsg struct {
	ID  int
	Err error
}

// StartOptions describes the process a session runs.
type StartOptions struct {
	ProjectID   string
	ProjectName string
	Dir         string
	// Argv is the command to run. Empty starts the user's login shell.
	Argv []string
	// Env is the full environment; nil means the current process's.
	Env []string
	// Interactive marks shells (as opposed to one-off commands like builds).
	Interactive bool
	// Title is shown in tab bars; it defaults to the project name.
	Title string
	// Tag classifies the session for the UI (e.g. "build", "pipeline");
	// empty for plain shells.
	Tag string
	// Width and Height are the initial terminal size (default 80x24).
	Width, Height int
}

// Session is one process running in a PTY.
type Session struct {
	ID          int
	ProjectID   string
	ProjectName string
	Dir         string
	Title       string
	Command     string
	Interactive bool
	Tag         string

	term     *vt.SafeEmulator
	proc     process
	notified atomic.Bool // an OutputMsg is queued and not yet delivered
	unseen   atomic.Bool // output arrived since the UI last called MarkSeen

	mu         sync.Mutex
	running    bool
	exitErr    error
	exited     chan struct{}
	outputDone chan struct{} // closed when pumpOutput stops
}

// process is the platform-specific PTY + child process.
type process interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Resize(width, height int) error
	// Kill terminates the whole process group.
	Kill() error
	// Wait blocks until the process exits.
	Wait() error
	// Close releases the PTY.
	Close() error
}

// Running reports whether the process is still alive.
func (s *Session) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// ExitCode is the process's exit code: 0 on success, -1 while running or
// when it was killed by a signal.
func (s *Session) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return -1
	}
	var exitErr interface{ ExitCode() int }
	if s.exitErr == nil {
		return 0
	}
	if errors.As(s.exitErr, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// ExitErr is the process's exit error (nil on success or while running).
func (s *Session) ExitErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitErr
}

// Done is closed once the process has exited.
func (s *Session) Done() <-chan struct{} { return s.exited }

// Render returns the visible screen with ANSI styling, one line per row.
func (s *Session) Render() string { return s.term.Render() }

// Size returns the terminal size in cells.
func (s *Session) Size() (width, height int) { return s.term.Width(), s.term.Height() }

// Cursor returns the cursor position and whether it is visible.
func (s *Session) Cursor() (x, y int, visible bool) {
	pos := s.term.CursorPosition()
	return pos.X, pos.Y, !s.term.CursorHidden()
}

// Text returns the plain text of the scrollback followed by the screen, with
// trailing blanks trimmed. Useful for logs and tests. It only uses the
// emulator's locked accessors, so it is safe while output is streaming.
func (s *Session) Text() string {
	var lines []string
	for y := range s.term.ScrollbackLen() {
		var b strings.Builder
		for x := 0; ; x++ {
			c := s.term.ScrollbackCellAt(x, y)
			if c == nil {
				break
			}
			b.WriteString(c.Content)
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	for _, l := range strings.Split(ansi.Strip(s.term.Render()), "\n") {
		lines = append(lines, strings.TrimRight(l, " "))
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// Unseen reports whether output arrived since MarkSeen was last called.
func (s *Session) Unseen() bool { return s.unseen.Load() }

// MarkSeen clears the unseen-output flag (call it while the session is shown).
func (s *Session) MarkSeen() { s.unseen.Store(false) }

// Manager owns all sessions. It is safe for concurrent use.
type Manager struct {
	mu       sync.Mutex
	sessions map[int]*Session
	order    []int
	nextID   int
	events   chan tea.Msg
}

// NewManager creates an empty session manager.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[int]*Session),
		events:   make(chan tea.Msg, 256),
	}
}

// Start launches a process in a new PTY session and returns its ID.
func (m *Manager) Start(opts StartOptions) (int, error) {
	if opts.Width <= 0 || opts.Height <= 0 {
		opts.Width, opts.Height = 80, 24
	}
	if len(opts.Argv) == 0 {
		opts.Argv = loginShell()
	}

	proc, err := startProcess(opts.Argv, opts.Dir, withTermEnv(opts.Env), opts.Width, opts.Height, opts.Interactive)
	if err != nil {
		return 0, err
	}

	return m.adopt(proc, opts).ID, nil
}

// adopt wraps a started process in a session and starts its pumps.
func (m *Manager) adopt(proc process, opts StartOptions) *Session {
	if opts.Width <= 0 || opts.Height <= 0 {
		opts.Width, opts.Height = 80, 24
	}
	if opts.Title == "" {
		opts.Title = opts.ProjectName
	}
	term := vt.NewSafeEmulator(opts.Width, opts.Height)
	term.SetScrollbackSize(scrollbackLines)

	m.mu.Lock()
	m.nextID++
	s := &Session{
		ID:          m.nextID,
		ProjectID:   opts.ProjectID,
		ProjectName: opts.ProjectName,
		Dir:         opts.Dir,
		Title:       opts.Title,
		Command:     strings.Join(opts.Argv, " "),
		Interactive: opts.Interactive,
		Tag:         opts.Tag,
		term:        term,
		proc:        proc,
		running:     true,
		exited:      make(chan struct{}),
		outputDone:  make(chan struct{}),
	}
	m.sessions[s.ID] = s
	m.order = append(m.order, s.ID)
	m.mu.Unlock()

	go m.pumpOutput(s)
	go pumpInput(s)
	go m.wait(s)
	return s
}

// pumpOutput feeds PTY output into the emulator and notifies the UI.
func (m *Manager) pumpOutput(s *Session) {
	defer close(s.outputDone)
	buf := make([]byte, 32*1024)
	for {
		n, err := s.proc.Read(buf)
		if n > 0 {
			_, _ = s.term.Write(buf[:n])
			s.unseen.Store(true)
			if s.notified.CompareAndSwap(false, true) {
				m.events <- OutputMsg{ID: s.ID}
			}
		}
		if err != nil {
			return
		}
	}
}

// pumpInput forwards what the emulator wants to send to the program - keys
// from SendKey/Paste and replies to terminal queries - into the PTY. The
// emulator's input side is an unbuffered pipe, so this must always run or
// the emulator deadlocks on the first query.
func pumpInput(s *Session) {
	buf := make([]byte, 4096)
	for {
		n, err := s.term.Read(buf)
		if n > 0 {
			if _, werr := s.proc.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// drainGrace bounds how long an exited session waits for its last output. A
// background job can keep the PTY open after the main process is gone.
const drainGrace = 500 * time.Millisecond

func (m *Manager) wait(s *Session) {
	err := s.proc.Wait()
	select {
	case <-s.outputDone:
	case <-time.After(drainGrace):
	}
	s.mu.Lock()
	s.running = false
	s.exitErr = err
	s.mu.Unlock()
	close(s.exited)
	m.events <- ExitedMsg{ID: s.ID, Err: err}
}

// Listen returns a command that delivers the next session event. The UI must
// re-issue it after every OutputMsg/ExitedMsg it receives (a single listener
// loop, like the old ListenForLogs).
func (m *Manager) Listen() tea.Cmd {
	return func() tea.Msg {
		msg := <-m.events
		if out, ok := msg.(OutputMsg); ok {
			if s := m.Get(out.ID); s != nil {
				s.notified.Store(false)
			}
		}
		return msg
	}
}

// Get returns a session by ID, or nil.
func (m *Manager) Get(id int) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

// List returns all sessions in start order.
func (m *Manager) List() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Session, 0, len(m.order))
	for _, id := range m.order {
		out = append(out, m.sessions[id])
	}
	return out
}

// SendKey types a key into the session.
func (m *Manager) SendKey(id int, key tea.KeyPressMsg) error {
	s := m.Get(id)
	if s == nil {
		return fmt.Errorf("no session %d", id)
	}
	s.term.SendKey(uv.KeyPressEvent(key))
	return nil
}

// Paste pastes text into the session (bracketed if the program asked for it).
func (m *Manager) Paste(id int, text string) error {
	s := m.Get(id)
	if s == nil {
		return fmt.Errorf("no session %d", id)
	}
	s.term.Paste(text)
	return nil
}

// Write sends raw bytes to the session's program.
func (m *Manager) Write(id int, data []byte) error {
	s := m.Get(id)
	if s == nil {
		return fmt.Errorf("no session %d", id)
	}
	_, err := s.proc.Write(data)
	return err
}

// Resize changes a session's terminal size (emulator and PTY together).
func (m *Manager) Resize(id, width, height int) error {
	s := m.Get(id)
	if s == nil {
		return fmt.Errorf("no session %d", id)
	}
	if width <= 0 || height <= 0 {
		return nil
	}
	if w, h := s.Size(); w == width && h == height {
		return nil
	}
	s.term.Resize(width, height)
	return s.proc.Resize(width, height)
}

// ResizeAll resizes every session.
func (m *Manager) ResizeAll(width, height int) {
	for _, s := range m.List() {
		_ = m.Resize(s.ID, width, height)
	}
}

// Kill terminates a session's process but keeps the session (and its
// output) around, e.g. to inspect a cancelled build.
func (m *Manager) Kill(id int) error {
	s := m.Get(id)
	if s == nil {
		return fmt.Errorf("no session %d", id)
	}
	if s.Running() {
		return s.proc.Kill()
	}
	return nil
}

// Close kills a session's process (if still running) and forgets it.
func (m *Manager) Close(id int) error {
	m.mu.Lock()
	s := m.sessions[id]
	if s == nil {
		m.mu.Unlock()
		return fmt.Errorf("no session %d", id)
	}
	delete(m.sessions, id)
	m.order = slices.DeleteFunc(m.order, func(x int) bool { return x == id })
	m.mu.Unlock()

	if s.Running() {
		_ = s.proc.Kill()
		<-s.exited
	}
	// Closing the PTY stops pumpOutput; only then is it safe to shut the
	// emulator's input side (which stops pumpInput).
	// The wait is bounded because closing a PTY master doesn't interrupt a
	// blocked read on every platform.
	err := s.proc.Close()
	select {
	case <-s.outputDone:
	case <-time.After(drainGrace):
	}
	// Ends pumpInput. vt's Emulator.Close isn't synchronized with Read, so
	// close the input pipe directly instead.
	if c, ok := s.term.InputPipe().(io.Closer); ok {
		_ = c.Close()
	}
	return err
}

// Shutdown closes every session; call it when the application exits.
func (m *Manager) Shutdown() {
	for _, s := range m.List() {
		_ = m.Close(s.ID)
	}
}

func withTermEnv(env []string) []string {
	if env == nil {
		env = environ()
	}
	env = slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		return strings.HasPrefix(kv, "TERM=") || strings.HasPrefix(kv, "COLORTERM=")
	})
	return append(env, "TERM=xterm-256color", "COLORTERM=truecolor")
}
