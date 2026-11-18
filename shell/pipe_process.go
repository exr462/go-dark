package shell

import (
	"bytes"
	"io"
	"os/exec"
	"sync"
)

// pipeProcess runs a command with plain pipes instead of a PTY, for
// platforms without one. stdout and stderr are merged, and bare "\n" is
// turned into "\r\n" so the emulator starts each line at column 0 like a
// terminal's line discipline would.
type pipeProcess struct {
	cmd   *exec.Cmd
	out   *io.PipeReader
	stdin io.WriteCloser
	done  chan struct{}
	pend  []byte // translated bytes not yet returned by Read
	mu    sync.Mutex
}

func startPipeProcess(argv []string, dir string, env []string) (process, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = env
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &pipeProcess{cmd: cmd, out: pr, stdin: stdin, done: make(chan struct{})}
	go func() {
		<-p.done
		_ = pw.Close()
	}()
	return p, nil
}

func (p *pipeProcess) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pend) == 0 {
		buf := make([]byte, len(b))
		n, err := p.out.Read(buf)
		if n == 0 {
			return 0, err
		}
		p.pend = toCRLF(buf[:n])
	}
	n := copy(b, p.pend)
	p.pend = p.pend[n:]
	return n, nil
}

func (p *pipeProcess) Write(b []byte) (int, error) { return p.stdin.Write(b) }

func (p *pipeProcess) Resize(int, int) error { return nil }

func (p *pipeProcess) Kill() error { return p.cmd.Process.Kill() }

func (p *pipeProcess) Wait() error {
	err := p.cmd.Wait()
	close(p.done)
	return err
}

func (p *pipeProcess) Close() error {
	_ = p.stdin.Close()
	return p.out.Close()
}

// toCRLF turns every "\n" not already preceded by "\r" into "\r\n". A "\r"
// at the very end of a chunk followed by "\n" at the start of the next is
// rare enough (and harmless: an extra carriage return) to ignore.
func toCRLF(b []byte) []byte {
	if !bytes.Contains(b, []byte("\n")) {
		return append([]byte(nil), b...)
	}
	out := make([]byte, 0, len(b)+bytes.Count(b, []byte("\n")))
	for i, c := range b {
		if c == '\n' && (i == 0 || b[i-1] != '\r') {
			out = append(out, '\r')
		}
		out = append(out, c)
	}
	return out
}
