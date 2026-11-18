//go:build !windows

package shell

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// killGrace is how long a hung-up process group gets before SIGKILL.
const killGrace = 2 * time.Second

type unixProcess struct {
	cmd  *exec.Cmd
	ptmx *os.File
	done chan struct{}
}

func startProcess(argv []string, dir string, env []string, width, height int, _ bool) (process, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = env
	// pty.Start makes the child a session leader with the PTY as its
	// controlling terminal, so ctrl+c/ctrl+z reach the foreground job.
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(width), Rows: uint16(height)}) //nolint:gosec
	if err != nil {
		return nil, err
	}
	return &unixProcess{cmd: cmd, ptmx: ptmx, done: make(chan struct{})}, nil
}

func (p *unixProcess) Read(b []byte) (int, error) {
	n, err := p.ptmx.Read(b)
	// Linux reports EIO on the master once the child side is gone.
	if errors.Is(err, syscall.EIO) {
		err = os.ErrClosed
	}
	return n, err
}

func (p *unixProcess) Write(b []byte) (int, error) { return p.ptmx.Write(b) }

func (p *unixProcess) Resize(width, height int) error {
	return pty.Setsize(p.ptmx, &pty.Winsize{Cols: uint16(width), Rows: uint16(height)}) //nolint:gosec
}

// Kill hangs up the whole process group like closing a terminal window
// would, then force-kills whatever is left after a grace period.
func (p *unixProcess) Kill() error {
	pgid := -p.cmd.Process.Pid
	_ = syscall.Kill(pgid, syscall.SIGHUP)
	select {
	case <-p.done:
		return nil
	case <-time.After(killGrace):
		return syscall.Kill(pgid, syscall.SIGKILL)
	}
}

func (p *unixProcess) Wait() error {
	err := p.cmd.Wait()
	close(p.done)
	return err
}

func (p *unixProcess) Close() error { return p.ptmx.Close() }

// loginShell is the user's shell as a login shell, falling back to /bin/sh.
func loginShell() []string {
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	return []string{sh, "-l"}
}

func environ() []string { return os.Environ() }
