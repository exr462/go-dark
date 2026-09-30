package componentaction

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func EditorModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	rawText := ui.Editor.Value()
	lines := strings.Split(rawText, "\n")
	currentLineIdx := ui.Editor.Line()

	var currentLineRunes []rune
	if currentLineIdx < len(lines) {
		currentLineRunes = []rune(lines[currentLineIdx])
	}
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		// returning to modal
		ui.ViewState = model.StateDashboard
		return nil

		// 1. Scroll Up a block of lines (Page Up)
	case "pgup":
		// Moves the cursor up by the height of the textarea
		for i := 0; i < ui.Editor.Height(); i++ {
			ui.Editor.CursorUp()
		}
		return nil

	// 2. Scroll Down a block of lines (Page Down)
	case "pgdown":
		// Moves the cursor down by the height of the textarea
		for i := 0; i < ui.Editor.Height(); i++ {
			ui.Editor.CursorDown()
		}
		return nil

	// 3. Optional: Jump to the absolute top of the file (Alt + G or Ctrl + Home)
	case "alt+g":
		ui.Editor.SetCursor(0)
		return nil
	case "left":
		if ui.EditorCol > 0 {
			ui.EditorCol--
		} else if currentLineIdx > 0 {
			prevLine := []rune(lines[currentLineIdx-1])
			ui.EditorCol = len(prevLine)
		}
	case "right":
		if ui.EditorCol < len(currentLineRunes) {
			ui.EditorCol++
		} else if currentLineIdx < len(lines)-1 {
			ui.EditorCol = 0
		}
	case "home":
		ui.EditorCol = 0
	case "end":
		ui.EditorCol = len(currentLineRunes)
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		//contentLoader.WriteContent()
		ui.ViewState = model.StateDashboard
		return nil
	}
	var cmd tea.Cmd
	ui.Editor, cmd = ui.Editor.Update(msg)
	updatedLines := strings.Split(ui.Editor.Value(), "\n")
	newLineIdx := ui.Editor.Line()
	if newLineIdx < len(updatedLines) {
		newLineRunes := []rune(updatedLines[newLineIdx])
		if ui.EditorCol > len(newLineRunes) {
			ui.EditorCol = len(newLineRunes)
		}
	}
	return cmd
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
