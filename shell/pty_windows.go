//go:build windows

package shell

import "os"

// Interactive shells are deliberately not offered on Windows (yet). One-off
// commands such as builds still run, through plain pipes instead of a PTY.
func startProcess(argv []string, dir string, env []string, _, _ int, interactive bool) (process, error) {
	if interactive {
		return nil, ErrUnsupported
	}
	return startPipeProcess(argv, dir, env)
}

func loginShell() []string { return []string{"cmd.exe"} }

func environ() []string { return os.Environ() }
