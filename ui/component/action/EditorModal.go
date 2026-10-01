package componentaction

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/storage"
)

// EditorModal drives every keypress while model.StateEditorModal is active.
// The editor is a small Vim-style modal editor:
//
//   - Normal mode: single-key navigation/commands (h/j/k/l, 0/$, gg/G, dd, x,
//     i/a/A/I/o/O to enter Insert mode, ":" to enter Command mode).
//   - Insert mode: keystrokes are forwarded as-is to the embedded textarea,
//     which already natively understands typing, arrows, backspace, etc.
//     Escape returns to Normal mode without leaving the editor.
//   - Command mode: a classic ":" command line accepting :w, :q, :q! and :wq
//     (plus :x as an alias of :wq), mirroring real Vim behavior including
//     refusing a bare :q when there are unsaved changes.
func EditorModal(ui *model.UI, contentLoader storage.ContentLoader, msg tea.KeyMsg) tea.Cmd {
	switch ui.EditorMode {
	case model.EditorModeInsert:
		return handleInsertMode(ui, contentLoader, msg)
	case model.EditorModeCommand:
		return handleCommandMode(ui, contentLoader, msg)
	default:
		return handleNormalMode(ui, contentLoader, msg)
	}
}

// handleInsertMode forwards keystrokes straight to the textarea so typing,
// arrows, backspace etc. behave exactly as before. Escape (and ctrl+s, as a
// convenience alias for :w + :q together) are the only intercepted keys.
func handleInsertMode(ui *model.UI, contentLoader storage.ContentLoader, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.EditorMode = model.EditorModeNormal
		return nil
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		return saveEdit(ui, contentLoader)
	}

	var cmd tea.Cmd
	ui.Editor, cmd = ui.Editor.Update(msg)
	ui.EditorDirty = ui.Editor.Value() != ui.EditorOriginalContent
	return cmd
}

// handleNormalMode implements the Vim-style Normal mode command set.
func handleNormalMode(ui *model.UI, contentLoader storage.ContentLoader, msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// Resolve a pending two-key combo (gg, dd) started on a previous keypress.
	if ui.EditorPendingKey != "" {
		pending := ui.EditorPendingKey
		ui.EditorPendingKey = ""
		switch {
		case pending == "g" && key == "g":
			jumpToTop(ui)
		case pending == "d" && key == "d":
			deleteCurrentLine(ui)
		}
		return nil
	}

	switch key {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		return cancelEdit(ui)

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		return saveEdit(ui, contentLoader)

	case ":":
		ui.EditorMode = model.EditorModeCommand
		ui.EditorCommandBuffer = ""
		return nil

	case "i":
		ui.EditorMode = model.EditorModeInsert
		return nil
	case "a":
		cmd := forwardKey(ui, tea.KeyMsg{Type: tea.KeyRight})
		ui.EditorMode = model.EditorModeInsert
		return cmd
	case "A":
		cmd := forwardKey(ui, tea.KeyMsg{Type: tea.KeyEnd})
		ui.EditorMode = model.EditorModeInsert
		return cmd
	case "I":
		cmd := forwardKey(ui, tea.KeyMsg{Type: tea.KeyHome})
		ui.EditorMode = model.EditorModeInsert
		return cmd
	case "o":
		cmds := []tea.Cmd{
			forwardKey(ui, tea.KeyMsg{Type: tea.KeyEnd}),
			forwardKey(ui, tea.KeyMsg{Type: tea.KeyEnter}),
		}
		ui.EditorMode = model.EditorModeInsert
		ui.EditorDirty = ui.Editor.Value() != ui.EditorOriginalContent
		return tea.Batch(cmds...)
	case "O":
		cmds := []tea.Cmd{
			forwardKey(ui, tea.KeyMsg{Type: tea.KeyHome}),
			forwardKey(ui, tea.KeyMsg{Type: tea.KeyEnter}),
		}
		ui.Editor.CursorUp()
		ui.EditorMode = model.EditorModeInsert
		ui.EditorDirty = ui.Editor.Value() != ui.EditorOriginalContent
		return tea.Batch(cmds...)

	case "h":
		return forwardKey(ui, tea.KeyMsg{Type: tea.KeyLeft})
	case "l":
		return forwardKey(ui, tea.KeyMsg{Type: tea.KeyRight})
	case "j":
		return forwardKey(ui, tea.KeyMsg{Type: tea.KeyDown})
	case "k":
		return forwardKey(ui, tea.KeyMsg{Type: tea.KeyUp})
	case "0":
		return forwardKey(ui, tea.KeyMsg{Type: tea.KeyHome})
	case "$":
		return forwardKey(ui, tea.KeyMsg{Type: tea.KeyEnd})
	case "G":
		return forwardKey(ui, tea.KeyMsg{Type: tea.KeyCtrlEnd})
	case "x":
		cmd := forwardKey(ui, tea.KeyMsg{Type: tea.KeyDelete})
		ui.EditorDirty = ui.Editor.Value() != ui.EditorOriginalContent
		return cmd

	case "g":
		ui.EditorPendingKey = "g"
	case "d":
		ui.EditorPendingKey = "d"
	}

	return nil
}

// handleCommandMode drives the ":" command line: character entry, backspace,
// escape-to-cancel and enter-to-execute.
func handleCommandMode(ui *model.UI, contentLoader storage.ContentLoader, msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		ui.EditorMode = model.EditorModeNormal
		ui.EditorCommandBuffer = ""
		return nil

	case tea.KeyEnter:
		cmdStr := ui.EditorCommandBuffer
		ui.EditorCommandBuffer = ""
		ui.EditorMode = model.EditorModeNormal
		return executeEditorCommand(ui, contentLoader, cmdStr)

	case tea.KeyBackspace:
		if len(ui.EditorCommandBuffer) > 0 {
			ui.EditorCommandBuffer = ui.EditorCommandBuffer[:len(ui.EditorCommandBuffer)-1]
		} else {
			ui.EditorMode = model.EditorModeNormal
		}
		return nil

	case tea.KeyRunes:
		ui.EditorCommandBuffer += string(msg.Runes)
		return nil
	}

	return nil
}

// executeEditorCommand parses and runs a Vim-style ":" command line.
func executeEditorCommand(ui *model.UI, contentLoader storage.ContentLoader, cmdStr string) tea.Cmd {
	switch cmdStr {
	case "w":
		writeBuffer(ui, contentLoader)
	case "q":
		quitEdit(ui, false)
	case "q!":
		quitEdit(ui, true)
	case "wq", "x":
		if writeBuffer(ui, contentLoader) {
			quitEdit(ui, true)
		}
	case "":
		// Bare ":" followed directly by enter - nothing to do.
	default:
		ui.StatusMsg = fmt.Sprintf("❌ Unknown command: :%s", cmdStr)
	}
	return nil
}

// forwardKey synthesizes the given key message straight into the textarea's
// own Update method, reusing its native (but otherwise unexported) cursor
// movement and editing behavior instead of reimplementing it here.
func forwardKey(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	ui.Editor, cmd = ui.Editor.Update(msg)
	return cmd
}

// jumpToTop moves the cursor to row 0, column 0 (Vim's "gg").
func jumpToTop(ui *model.UI) {
	for ui.Editor.Line() > 0 {
		ui.Editor.CursorUp()
	}
	ui.Editor.CursorStart()
}

// deleteCurrentLine removes the line the cursor is on (Vim's "dd").
func deleteCurrentLine(ui *model.UI) {
	lines := strings.Split(ui.Editor.Value(), "\n")
	row := ui.Editor.Line()
	if row < 0 || row >= len(lines) {
		return
	}

	lines = append(lines[:row], lines[row+1:]...)
	if len(lines) == 0 {
		lines = []string{""}
	}

	ui.Editor.SetValue(strings.Join(lines, "\n"))
	ui.EditorDirty = ui.Editor.Value() != ui.EditorOriginalContent

	target := row
	if target >= len(lines) {
		target = len(lines) - 1
	}
	for ui.Editor.Line() > target {
		ui.Editor.CursorUp()
	}
	ui.Editor.CursorStart()
}

// cancelEdit discards any unsaved changes, restores the original buffer and
// returns to the dashboard. Used by Normal-mode Escape (a quick, no-prompt
// "get me out of here" safety valve alongside the stricter :q/:q!).
func cancelEdit(ui *model.UI) tea.Cmd {
	if ui.EditorDirty {
		ui.Editor.SetValue(ui.EditorOriginalContent)
		ui.EditorDirty = false
		ui.StatusMsg = "✋ Edit cancelled, changes discarded."
	}
	ui.ViewState = model.StateDashboard
	return nil
}

// saveEdit persists the current buffer to disk and returns to the dashboard
// on success - the ctrl+s convenience alias for ":wq". On failure, the
// editor stays open so the user doesn't lose their edits.
func saveEdit(ui *model.UI, contentLoader storage.ContentLoader) tea.Cmd {
	if writeBuffer(ui, contentLoader) {
		ui.ViewState = model.StateDashboard
	}
	return nil
}

// writeBuffer is the shared ":w" / ctrl+s implementation: it persists the
// current buffer via the injected ContentLoader and reports whether the
// write succeeded.
func writeBuffer(ui *model.UI, contentLoader storage.ContentLoader) bool {
	if ui.ActiveFilePath == "" {
		ui.StatusMsg = "❌ Editor: no active file to save."
		return false
	}

	content := ui.Editor.Value()
	if err := contentLoader.WriteContent(content, ui.ActiveFilePath); err != nil {
		ui.StatusMsg = fmt.Sprintf("❌ Editor: failed to save %s: %v", ui.ActiveFilePath, err)
		ui.LastError = err
		return false
	}

	ui.EditorOriginalContent = content
	ui.EditorDirty = false
	ui.StatusMsg = fmt.Sprintf("💾 \"%s\" written", ui.ActiveFilePath)
	return true
}

// quitEdit implements ":q" (refuses to quit on unsaved changes) and ":q!"
// (force-quits, discarding unsaved changes).
func quitEdit(ui *model.UI, force bool) {
	if ui.EditorDirty && !force {
		ui.StatusMsg = "❌ No write since last change (use :q! to discard, or :wq to save and quit)"
		return
	}

	if ui.EditorDirty {
		ui.Editor.SetValue(ui.EditorOriginalContent)
		ui.EditorDirty = false
	}
	ui.ViewState = model.StateDashboard
}

// Asynchronous multi-stage compiler output trigger tracking loops
//
//goland:noinspection GoUnusedFunction
func executeCommandPipelineCmd(actionType string, lp lsp.LanguageProvider, filePath string) tea.Cmd {
	return func() tea.Msg {
		if lp == nil || filePath == "" {
			return lsp.ExecutionResultMsg{Output: "No active file available to execute."}
		}

		var cmd *exec.Cmd
		if actionType == "build" {
			cmd = lp.GetBuildCommand(filePath)
		} else {
			cmd = lp.GetRunCommand(filePath)
		}

		if cmd == nil {
			return lsp.ExecutionResultMsg{Output: fmt.Sprintf("Action [%s] not supported for language: %s", actionType, lp.Name())}
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			return lsp.ExecutionResultMsg{Output: fmt.Sprintf("[%s] TRIPPED ERROR:\n%s\n%v", strings.ToUpper(actionType), string(out), err)}
		}
		return lsp.ExecutionResultMsg{Output: fmt.Sprintf("[%s] COMPLETED (%s):\n%s", strings.ToUpper(actionType), lp.Name(), string(out))}
	}
}
