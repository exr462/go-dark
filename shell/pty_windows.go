//go:build windows

package shell

import "os"

// Embedded terminals are deliberately not offered on Windows (yet): go-dark
// ships a Windows binary, but every other feature works without a PTY.
func startProcess([]string, string, []string, int, int) (process, error) {
	return nil, ErrUnsupported
}

func loginShell() []string { return []string{"cmd.exe"} }

func environ() []string { return os.Environ() }
