package componentaction

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

// newVimEditorTestUI builds an editor UI whose cursor sits at the very start
// of the document (mirroring what happens right after a real file load),
// which makes cursor-movement assertions predictable.
func newVimEditorTestUI(initialContent string) *model.UI {
	ta := textarea.New()
	ta.SetValue(initialContent)
	for ta.Line() > 0 {
		ta.CursorUp()
	}
	ta.CursorStart()
	ta.Focus()

	return &model.UI{
		Config:                config.Config{ShortCuts: action.DefaultShortcuts},
		Editor:                ta,
		ActiveFilePath:        "/tmp/file.txt",
		EditorOriginalContent: initialContent,
		ViewState:             model.StateEditorModal,
	}
}

func col(ui *model.UI) int { return ui.Editor.LineInfo().CharOffset }

func TestEditorModal_OpensInNormalModeByDefault(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	if ui.EditorMode != model.EditorModeNormal {
		t.Fatalf("expected zero-value EditorMode to be Normal, got %v", ui.EditorMode)
	}
}

func TestEditorModal_iEntersInsertMode(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("i"))

	if ui.EditorMode != model.EditorModeInsert {
		t.Fatalf("expected 'i' to enter Insert mode, got %v", ui.EditorMode)
	}
}

func TestEditorModal_EscapeInInsertModeReturnsToNormalWithoutLeavingEditor(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	ui.EditorMode = model.EditorModeInsert
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("esc"))

	if ui.EditorMode != model.EditorModeNormal {
		t.Fatalf("expected esc to return to Normal mode, got %v", ui.EditorMode)
	}
	if ui.ViewState != model.StateEditorModal {
		t.Fatalf("expected esc in Insert mode to keep the editor open, got %v", ui.ViewState)
	}
}

func TestEditorModal_EscapeInNormalModeCancelsAndLeavesEditor(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("esc"))

	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected esc in Normal mode to leave the editor, got %v", ui.ViewState)
	}
}

func TestEditorModal_InsertModeTypingGoesIntoBuffer(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	ui.EditorMode = model.EditorModeInsert
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("Z"))

	if !strings.Contains(ui.Editor.Value(), "Z") {
		t.Fatalf("expected typed rune to be inserted into the buffer, got %q", ui.Editor.Value())
	}
}

func TestEditorModal_NormalModeLettersDoNotTypeIntoBuffer(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	// "l" is a Normal-mode movement key; it must not be inserted as text.
	EditorModal(ui, loader, testKeyMsg("l"))

	if ui.Editor.Value() != "abc" {
		t.Fatalf("expected Normal-mode 'l' to move the cursor, not type, got %q", ui.Editor.Value())
	}
}

func TestEditorModal_hjklMoveCursor(t *testing.T) {
	ui := newVimEditorTestUI("abc\ndef\nghi")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("l"))
	if got := col(ui); got != 1 {
		t.Fatalf("expected 'l' to move cursor right to col 1, got %d", got)
	}

	EditorModal(ui, loader, testKeyMsg("j"))
	if ui.Editor.Line() != 1 {
		t.Fatalf("expected 'j' to move cursor down to line 1, got %d", ui.Editor.Line())
	}

	EditorModal(ui, loader, testKeyMsg("h"))
	if got := col(ui); got != 0 {
		t.Fatalf("expected 'h' to move cursor left back to col 0, got %d", got)
	}

	EditorModal(ui, loader, testKeyMsg("k"))
	if ui.Editor.Line() != 0 {
		t.Fatalf("expected 'k' to move cursor back up to line 0, got %d", ui.Editor.Line())
	}
}

func TestEditorModal_DollarAndZeroMoveToLineEnds(t *testing.T) {
	ui := newVimEditorTestUI("abcdef")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("$"))
	if got := col(ui); got != len("abcdef") {
		t.Fatalf("expected '$' to move to end of line, got col %d", got)
	}

	EditorModal(ui, loader, testKeyMsg("0"))
	if got := col(ui); got != 0 {
		t.Fatalf("expected '0' to move to start of line, got col %d", got)
	}
}

func TestEditorModal_ggJumpsToTopAndGJumpsToBottom(t *testing.T) {
	ui := newVimEditorTestUI("one\ntwo\nthree")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("G"))
	if ui.Editor.Line() != 2 {
		t.Fatalf("expected 'G' to jump to the last line, got %d", ui.Editor.Line())
	}

	EditorModal(ui, loader, testKeyMsg("g"))
	EditorModal(ui, loader, testKeyMsg("g"))
	if ui.Editor.Line() != 0 {
		t.Fatalf("expected 'gg' to jump back to the first line, got %d", ui.Editor.Line())
	}
}

func TestEditorModal_xDeletesCharacterUnderCursor(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("x"))

	if ui.Editor.Value() != "bc" {
		t.Fatalf("expected 'x' to delete the character under the cursor, got %q", ui.Editor.Value())
	}
	if !ui.EditorDirty {
		t.Fatal("expected 'x' to mark the buffer dirty")
	}
}

func TestEditorModal_ddDeletesCurrentLine(t *testing.T) {
	ui := newVimEditorTestUI("one\ntwo\nthree")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("d"))
	EditorModal(ui, loader, testKeyMsg("d"))

	if ui.Editor.Value() != "two\nthree" {
		t.Fatalf("expected 'dd' to remove the first line, got %q", ui.Editor.Value())
	}
}

func TestEditorModal_oOpensLineBelowAndEntersInsert(t *testing.T) {
	ui := newVimEditorTestUI("one\ntwo")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("o"))

	if ui.EditorMode != model.EditorModeInsert {
		t.Fatalf("expected 'o' to enter Insert mode, got %v", ui.EditorMode)
	}
	if ui.Editor.Value() != "one\n\ntwo" {
		t.Fatalf("expected 'o' to open a new blank line below, got %q", ui.Editor.Value())
	}
	if ui.Editor.Line() != 1 {
		t.Fatalf("expected cursor to land on the new blank line (row 1), got %d", ui.Editor.Line())
	}
}

func TestEditorModal_OOpensLineAboveAndEntersInsert(t *testing.T) {
	ui := newVimEditorTestUI("one\ntwo")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg("O"))

	if ui.EditorMode != model.EditorModeInsert {
		t.Fatalf("expected 'O' to enter Insert mode, got %v", ui.EditorMode)
	}
	if ui.Editor.Value() != "\none\ntwo" {
		t.Fatalf("expected 'O' to open a new blank line above, got %q", ui.Editor.Value())
	}
	if ui.Editor.Line() != 0 {
		t.Fatalf("expected cursor to land on the new blank line (row 0), got %d", ui.Editor.Line())
	}
}

func TestEditorModal_ColonEntersCommandModeAndAccumulatesBuffer(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	if ui.EditorMode != model.EditorModeCommand {
		t.Fatalf("expected ':' to enter Command mode, got %v", ui.EditorMode)
	}

	EditorModal(ui, loader, testKeyMsg("w"))
	if ui.EditorCommandBuffer != "w" {
		t.Fatalf("expected typed 'w' to accumulate in the command buffer, got %q", ui.EditorCommandBuffer)
	}
}

func TestEditorModal_ColonWWritesWithoutLeavingEditor(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	ui.Editor.SetValue("abc edited")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	EditorModal(ui, loader, testKeyMsg("w"))
	EditorModal(ui, loader, tea.KeyMsg{Type: tea.KeyEnter})

	if got := loader.written["/tmp/file.txt"]; got != "abc edited" {
		t.Fatalf("expected :w to persist the buffer, got %q", got)
	}
	if ui.ViewState != model.StateEditorModal {
		t.Fatalf("expected :w to keep the editor open, got %v", ui.ViewState)
	}
	if ui.EditorDirty {
		t.Fatal("expected :w to clear the dirty flag")
	}
	if ui.EditorMode != model.EditorModeNormal {
		t.Fatalf("expected Command mode to return to Normal after running, got %v", ui.EditorMode)
	}
}

func TestEditorModal_ColonQRefusesToQuitWithUnsavedChanges(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	ui.Editor.SetValue("abc edited")
	ui.EditorDirty = true
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	EditorModal(ui, loader, testKeyMsg("q"))
	EditorModal(ui, loader, tea.KeyMsg{Type: tea.KeyEnter})

	if ui.ViewState != model.StateEditorModal {
		t.Fatalf("expected bare :q to refuse quitting with unsaved changes, got %v", ui.ViewState)
	}
	if ui.StatusMsg == "" {
		t.Fatal("expected a status message explaining why :q was refused")
	}
}

func TestEditorModal_ColonQQuitsWhenClean(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	EditorModal(ui, loader, testKeyMsg("q"))
	EditorModal(ui, loader, tea.KeyMsg{Type: tea.KeyEnter})

	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected :q to quit when there are no unsaved changes, got %v", ui.ViewState)
	}
}

func TestEditorModal_ColonQBangForceQuitsDiscardingChanges(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	ui.Editor.SetValue("abc edited")
	ui.EditorDirty = true
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	EditorModal(ui, loader, testKeyMsg("q"))
	EditorModal(ui, loader, testKeyMsg("!"))
	EditorModal(ui, loader, tea.KeyMsg{Type: tea.KeyEnter})

	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected :q! to force-quit, got %v", ui.ViewState)
	}
	if ui.Editor.Value() != "abc" {
		t.Fatalf("expected :q! to discard unsaved changes, got %q", ui.Editor.Value())
	}
	if len(loader.written) != 0 {
		t.Fatalf("expected :q! to never write to disk, got %v", loader.written)
	}
}

func TestEditorModal_ColonWQSavesThenQuits(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	ui.Editor.SetValue("abc edited")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	EditorModal(ui, loader, testKeyMsg("w"))
	EditorModal(ui, loader, testKeyMsg("q"))
	EditorModal(ui, loader, tea.KeyMsg{Type: tea.KeyEnter})

	if got := loader.written["/tmp/file.txt"]; got != "abc edited" {
		t.Fatalf("expected :wq to persist the buffer, got %q", got)
	}
	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected :wq to leave the editor, got %v", ui.ViewState)
	}
}

func TestEditorModal_UnknownCommandShowsError(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	EditorModal(ui, loader, testKeyMsg("z"))
	EditorModal(ui, loader, tea.KeyMsg{Type: tea.KeyEnter})

	if ui.ViewState != model.StateEditorModal {
		t.Fatalf("expected an unknown command to not leave the editor, got %v", ui.ViewState)
	}
	if !strings.Contains(ui.StatusMsg, "Unknown command") {
		t.Fatalf("expected an 'unknown command' status message, got %q", ui.StatusMsg)
	}
}

func TestEditorModal_CommandModeBackspaceOnEmptyBufferReturnsToNormal(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	EditorModal(ui, loader, tea.KeyMsg{Type: tea.KeyBackspace})

	if ui.EditorMode != model.EditorModeNormal {
		t.Fatalf("expected backspace on an empty command buffer to exit Command mode, got %v", ui.EditorMode)
	}
}

func TestEditorModal_CommandModeEscapeCancels(t *testing.T) {
	ui := newVimEditorTestUI("abc")
	loader := newFakeContentLoader()

	EditorModal(ui, loader, testKeyMsg(":"))
	EditorModal(ui, loader, testKeyMsg("w"))
	EditorModal(ui, loader, tea.KeyMsg{Type: tea.KeyEscape})

	if ui.EditorMode != model.EditorModeNormal {
		t.Fatalf("expected esc to cancel Command mode, got %v", ui.EditorMode)
	}
	if ui.EditorCommandBuffer != "" {
		t.Fatalf("expected esc to clear the command buffer, got %q", ui.EditorCommandBuffer)
	}
	if len(loader.written) != 0 {
		t.Fatal("expected esc to cancel without running the command")
	}
}
