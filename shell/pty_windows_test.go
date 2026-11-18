//go:build windows

package shell

import (
	"errors"
	"testing"
)

func TestStart_UnsupportedOnWindows(t *testing.T) {
	_, err := NewManager().Start(StartOptions{Argv: []string{"cmd.exe"}})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("expected ErrUnsupported, got %v", err)
	}
}
