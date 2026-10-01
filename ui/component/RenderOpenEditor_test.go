package component

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
)

func newEditorRenderTestUI(content string) *model.UI {
	ta := textarea.New()
	ta.SetValue(content)
	for ta.Line() > 0 {
		ta.CursorUp()
	}
	ta.CursorStart()
	ta.Focus()

	return &model.UI{
		WindowWidth:            100,
		WindowHeight:           30,
		Editor:                 ta,
		ActiveFilePath:         "/tmp/demo.go",
		ActiveLanguageProvider: lsp.DefaultProvider{},
	}
}

func TestRenderOpenEditor_ShowsNormalModeIndicatorByDefault(t *testing.T) {
	ui := newEditorRenderTestUI("package main\n")

	out := RenderOpenEditor(ui)

	if !strings.Contains(out, "NORMAL") {
		t.Fatalf("expected the Normal-mode indicator to be rendered, got:\n%s", out)
	}
}

func TestRenderOpenEditor_ShowsInsertModeIndicator(t *testing.T) {
	ui := newEditorRenderTestUI("package main\n")
	ui.EditorMode = model.EditorModeInsert

	out := RenderOpenEditor(ui)

	if !strings.Contains(out, "INSERT") {
		t.Fatalf("expected the Insert-mode indicator to be rendered, got:\n%s", out)
	}
}

func TestRenderOpenEditor_ShowsLiveCommandBuffer(t *testing.T) {
	ui := newEditorRenderTestUI("package main\n")
	ui.EditorMode = model.EditorModeCommand
	ui.EditorCommandBuffer = "w"

	out := RenderOpenEditor(ui)

	if !strings.Contains(out, ":w") {
		t.Fatalf("expected the live ':w' command buffer to be rendered, got:\n%s", out)
	}
}

func TestRenderOpenEditor_ScrollsToKeepCursorLineVisible(t *testing.T) {
	ui := newEditorRenderTestUI(strings.Repeat("line\n", 200))
	ui.WindowHeight = 20 // force a small viewport so scrolling is required

	for i := 0; i < 150; i++ {
		ui.Editor.CursorDown()
	}

	out := RenderOpenEditor(ui)

	if strings.Contains(out, "  1 │") {
		t.Fatalf("expected the viewport to have scrolled past line 1, but it's still visible:\n%s", out)
	}
}

// TestRenderOpenEditor_FocusesTheTextareaEvenIfNeverFocusedBefore is a
// regression test for a real bug: nothing in the app ever called
// ui.Editor.Focus() (textarea.New() defaults to unfocused), so every
// keystroke forwarded to it - typing, hjkl movement, scrolling - silently
// no-opped because textarea.Model.Update() returns immediately while
// unfocused. RenderOpenEditor must focus it unconditionally, every render,
// the same way the terminal cockpit focuses its input field.
func TestRenderOpenEditor_FocusesTheTextareaEvenIfNeverFocusedBefore(t *testing.T) {
	ui := newEditorRenderTestUI("package main\n")
	ui.Editor.Blur() // simulate the real app's never-focused starting state

	if ui.Editor.Focused() {
		t.Fatal("test setup invalid: editor should start unfocused")
	}

	RenderOpenEditor(ui)

	if !ui.Editor.Focused() {
		t.Fatal("expected RenderOpenEditor to focus the textarea so keystrokes aren't silently dropped")
	}
}

// TestRenderOpenEditor_UsesNearFullWindowSize guards the "full screen" sizing
// requirement: the editor should scale almost all the way to the terminal
// viewport instead of staying a small, fixed-size popup.
func TestRenderOpenEditor_UsesNearFullWindowSize(t *testing.T) {
	ui := newEditorRenderTestUI(strings.Repeat("line\n", 100))
	ui.WindowWidth = 160
	ui.WindowHeight = 50

	out := RenderOpenEditor(ui)

	lines := strings.Split(out, "\n")
	if len(lines) < 45 {
		t.Fatalf("expected the editor to use almost the entire 50-row window, only rendered %d lines", len(lines))
	}
}

// TestRenderOpenEditor_StatusBarStaysPinnedToTheBottom guards against the
// status/shortcut bar floating immediately under whatever little text is in
// a short file. A short file and a long file should render the exact same
// total number of lines, with the status bar ("-- NORMAL --") only ever
// appearing on the very last line - i.e. the content area is always padded
// out to fill the editor, so the footer never moves.
func TestRenderOpenEditor_StatusBarStaysPinnedToTheBottom(t *testing.T) {
	shortUI := newEditorRenderTestUI("one line\n")
	shortUI.WindowWidth = 120
	shortUI.WindowHeight = 40

	longUI := newEditorRenderTestUI(strings.Repeat("another line of code\n", 200))
	longUI.WindowWidth = 120
	longUI.WindowHeight = 40

	shortLines := strings.Split(RenderOpenEditor(shortUI), "\n")
	longLines := strings.Split(RenderOpenEditor(longUI), "\n")

	if len(shortLines) != len(longLines) {
		t.Fatalf("expected a short file and a long file to render the same total height (footer pinned to bottom), got %d vs %d", len(shortLines), len(longLines))
	}

	findStatusLineIdx := func(lines []string) int {
		for i, l := range lines {
			if strings.Contains(l, "NORMAL") {
				return i
			}
		}
		return -1
	}

	shortIdx := findStatusLineIdx(shortLines)
	longIdx := findStatusLineIdx(longLines)
	if shortIdx == -1 || longIdx == -1 {
		t.Fatalf("expected to find the -- NORMAL -- status line in both renders, short=%d long=%d", shortIdx, longIdx)
	}
	if shortIdx != longIdx {
		t.Fatalf("expected the status bar to be pinned at the same bottom row regardless of file length, got short=%d long=%d", shortIdx, longIdx)
	}
}
