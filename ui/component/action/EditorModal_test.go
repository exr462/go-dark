package componentaction

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

// fakeContentLoader is an in-memory storage.ContentLoader test double so
// Save behavior can be verified without touching the real filesystem.
type fakeContentLoader struct {
	written  map[string]string
	writeErr error
}

func newFakeContentLoader() *fakeContentLoader {
	return &fakeContentLoader{written: make(map[string]string)}
}

func (f *fakeContentLoader) WriteContent(content string, path string) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.written[path] = content
	return nil
}

func (f *fakeContentLoader) GetContent(path string) (string, error) {
	return f.written[path], nil
}

func newEditorTestUI(initialContent string) *model.UI {
	ta := textarea.New()
	ta.SetValue(initialContent)
	ta.Focus()

	return &model.UI{
		Config:                config.Config{ShortCuts: action.DefaultShortcuts},
		Editor:                ta,
		ActiveFilePath:        "/tmp/file.txt",
		EditorOriginalContent: initialContent,
	}
}

func TestSaveEdit_WritesBufferAndReturnsToDashboard(t *testing.T) {
	ui := newEditorTestUI("hello")
	ui.Editor.SetValue("hello world")
	ui.ViewState = model.StateEditorModal
	loader := newFakeContentLoader()

	saveEdit(ui, loader)

	if got := loader.written["/tmp/file.txt"]; got != "hello world" {
		t.Fatalf("expected file to be written with new content, got %q", got)
	}
	if ui.EditorDirty {
		t.Fatal("expected EditorDirty to be false after a successful save")
	}
	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected ViewState to return to Dashboard, got %v", ui.ViewState)
	}
}

func TestSaveEdit_KeepsEditorOpenOnWriteError(t *testing.T) {
	ui := newEditorTestUI("hello")
	ui.ViewState = model.StateEditorModal
	loader := newFakeContentLoader()
	loader.writeErr = errors.New("disk full")

	saveEdit(ui, loader)

	if ui.ViewState != model.StateEditorModal {
		t.Fatalf("expected editor to stay open on save failure, got %v", ui.ViewState)
	}
	if ui.LastError == nil {
		t.Fatal("expected LastError to be set on a failed save")
	}
}

func TestSaveEdit_NoActiveFilePathIsNoOp(t *testing.T) {
	ui := newEditorTestUI("hello")
	ui.ActiveFilePath = ""
	loader := newFakeContentLoader()

	saveEdit(ui, loader)

	if len(loader.written) != 0 {
		t.Fatalf("expected nothing to be written without an active file path, got %v", loader.written)
	}
}

func TestCancelEdit_RevertsDirtyBufferAndReturnsToDashboard(t *testing.T) {
	ui := newEditorTestUI("original")
	ui.Editor.SetValue("original edited")
	ui.EditorDirty = true
	ui.ViewState = model.StateEditorModal

	cancelEdit(ui)

	if ui.Editor.Value() != "original" {
		t.Fatalf("expected buffer to be reverted to original content, got %q", ui.Editor.Value())
	}
	if ui.EditorDirty {
		t.Fatal("expected EditorDirty to be false after cancel")
	}
	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected ViewState to return to Dashboard, got %v", ui.ViewState)
	}
}

func TestCancelEdit_NoOpWhenNotDirty(t *testing.T) {
	ui := newEditorTestUI("original")
	ui.ViewState = model.StateEditorModal

	cancelEdit(ui)

	if ui.Editor.Value() != "original" {
		t.Fatalf("expected buffer to remain unchanged, got %q", ui.Editor.Value())
	}
	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected ViewState to return to Dashboard, got %v", ui.ViewState)
	}
}

func TestEditorModal_TypingMarksBufferDirty(t *testing.T) {
	ui := newEditorTestUI("abc")
	ui.ViewState = model.StateEditorModal
	ui.EditorMode = model.EditorModeInsert
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("x"))

	if !ui.EditorDirty {
		t.Fatal("expected typing to mark the buffer dirty")
	}
}
