package componentaction

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/component"
)

// TestEditorLifecycle_WorksEvenWhenNeverExplicitlyFocused is an end-to-end
// regression test mirroring the real app's boot sequence: Init.go's
// initializeEditor builds the textarea via textarea.New() and never calls
// Focus() on it; the very first render then happens via
// component.RenderOpenEditor, and only after that does a keypress arrive
// through EditorModal. If RenderOpenEditor didn't focus the textarea, this
// entire flow would silently drop every keystroke.
func TestEditorLifecycle_WorksEvenWhenNeverExplicitlyFocused(t *testing.T) {
	ui := &model.UI{
		Config:                 config.Config{ShortCuts: action.DefaultShortcuts},
		Editor:                 textarea.New(), // never focused, exactly like Init.go's initializeEditor
		ActiveFilePath:         "/tmp/file.go",
		ActiveLanguageProvider: lsp.DefaultProvider{},
		ViewState:              model.StateEditorModal,
		WindowWidth:            100,
		WindowHeight:           30,
	}
	ui.Editor.SetValue("package main\n")

	// Simulate bubbletea's first View() pass before any key event arrives.
	component.RenderOpenEditor(ui)

	loader := newFakeContentLoader()

	// "i" to enter Insert mode, then type a character.
	EditorModal(ui, loader, testKeyMsg("i"))
	EditorModal(ui, loader, testKeyMsg("X"))

	if ui.EditorMode != model.EditorModeInsert {
		t.Fatalf("expected Insert mode to engage, got %v", ui.EditorMode)
	}
	if ui.Editor.Value() == "package main\n" {
		t.Fatal("expected typed character to reach the buffer; the textarea was likely left unfocused")
	}
}

// TestEditorLifecycle_LongLineAppendAtEndWorks is a regression test for a
// real bug: Init.go used to build the textarea with SetWidth(10). Any line
// longer than ~10 characters (i.e. almost every real line of code) then got
// internally soft-wrapped by the textarea, so LineInfo().CharOffset (what
// RenderOpenEditor uses to draw the cursor) stopped matching the real column
// in the raw line, and "A"/"$"/End no longer landed on the actual end of the
// line. Typed characters appeared to land in the wrong place or not at all.
// The fix uses a very large textarea width so a line never wraps internally.
func TestEditorLifecycle_LongLineAppendAtEndWorks(t *testing.T) {
	ta := textarea.New()
	ta.SetWidth(4096) // mirrors Init.go's initializeEditor width fix
	ta.SetHeight(4096)

	longLine := "func doSomethingWithAVeryLongNameThatExceedsTenCharacters() error {"
	ui := &model.UI{
		Config:                 config.Config{ShortCuts: action.DefaultShortcuts},
		Editor:                 ta,
		ActiveFilePath:         "/tmp/file.go",
		ActiveLanguageProvider: lsp.DefaultProvider{},
		ViewState:              model.StateEditorModal,
		WindowWidth:            100,
		WindowHeight:           30,
	}
	ui.Editor.SetValue(longLine)
	component.RenderOpenEditor(ui)

	loader := newFakeContentLoader()

	// Vim "A": jump to the true end of the line and enter Insert mode.
	EditorModal(ui, loader, testKeyMsg("A"))
	if ui.EditorMode != model.EditorModeInsert {
		t.Fatalf("expected Insert mode after 'A', got %v", ui.EditorMode)
	}
	if got := ui.Editor.LineInfo().CharOffset; got != len(longLine) {
		t.Fatalf("expected cursor at the real end of the line (%d), got CharOffset=%d - the line is still being soft-wrapped internally", len(longLine), got)
	}

	EditorModal(ui, loader, testKeyMsg("X"))
	want := longLine + "X"
	if ui.Editor.Value() != want {
		t.Fatalf("expected appended character at the true end of the line, got %q, want %q", ui.Editor.Value(), want)
	}
}
