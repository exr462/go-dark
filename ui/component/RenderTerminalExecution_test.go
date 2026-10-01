package component

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/terminal"
)

func TestWrapTerminalLine_ShortLinePassesThrough(t *testing.T) {
	got := wrapTerminalLine("echo hi", 80)
	if len(got) != 1 || got[0] != "echo hi" {
		t.Fatalf("expected single unwrapped line, got %#v", got)
	}
}

func TestWrapTerminalLine_EmptyLineKeptAsBlankRow(t *testing.T) {
	got := wrapTerminalLine("", 80)
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("expected single blank line, got %#v", got)
	}
}

func TestWrapTerminalLine_LongLineWrapsAcrossMultipleRows(t *testing.T) {
	text := strings.Repeat("x", 25)
	got := wrapTerminalLine(text, 10)

	if len(got) != 3 {
		t.Fatalf("expected 3 wrapped rows for 25 chars at width 10, got %d: %#v", len(got), got)
	}
	if joined := strings.Join(got, ""); joined != text {
		t.Fatalf("wrapped rows should reconstruct original text, got %q want %q", joined, text)
	}
}

func TestWrapTerminalLine_NonPositiveWidthReturnsOriginal(t *testing.T) {
	got := wrapTerminalLine("anything", 0)
	if len(got) != 1 || got[0] != "anything" {
		t.Fatalf("expected passthrough on non-positive width, got %#v", got)
	}
}

func newTerminalTestUI() *model.UI {
	inputs := make([]textinput.Model, kbd.Ceiling)
	for i := range inputs {
		inputs[i] = textinput.New()
	}

	return &model.UI{
		WindowWidth:  120,
		WindowHeight: 40,
		Inputs:       inputs,
		Config: config.Config{
			Projects: []config.Project{{Name: "demo-project", Path: "/tmp/demo"}},
		},
	}
}

func TestRenderTerminalExecution_ShowsWelcomeMessageWhenHistoryEmpty(t *testing.T) {
	ui := newTerminalTestUI()

	out := RenderTerminalExecution(ui)

	if !strings.Contains(out, "Welcome to the go-dark shell") {
		t.Fatalf("expected welcome message when no terminal history exists, got:\n%s", out)
	}
}

func TestRenderTerminalExecution_RendersPromptAndOutputHistory(t *testing.T) {
	ui := newTerminalTestUI()
	ui.TerminalLogs = []terminal.LogLine{
		{Text: "dev@demo-project:~$ echo hi", IsPrompt: true},
		{Text: "hi", IsErr: false},
		{Text: "boom", IsErr: true},
	}

	out := RenderTerminalExecution(ui)

	for _, want := range []string{"echo hi", "hi", "boom", "demo-project"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected rendered output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRenderTerminalExecution_UsesNearFullWindowSize(t *testing.T) {
	ui := newTerminalTestUI()
	ui.WindowWidth = 200
	ui.WindowHeight = 60

	out := RenderTerminalExecution(ui)

	lines := strings.Split(out, "\n")
	if len(lines) < 50 {
		t.Fatalf("expected the terminal to use most of a tall window (60 rows), only rendered %d lines", len(lines))
	}
}
