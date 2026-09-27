package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateEditorModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rawText := m.ui.Editor.Value()
	lines := strings.Split(rawText, "\n")
	currentLineIdx := m.ui.Editor.Line()

	var currentLineRunes []rune
	if currentLineIdx < len(lines) {
		currentLineRunes = []rune(lines[currentLineIdx])
	}
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
		// returning to modal
		m.ui.ViewState = model.StateDashboard
		return m, nil

		// 1. Scroll Up a block of lines (Page Up)
	case "pgup":
		// Moves the cursor up by the height of the textarea
		for i := 0; i < m.ui.Editor.Height(); i++ {
			m.ui.Editor.CursorUp()
		}
		return m, nil

	// 2. Scroll Down a block of lines (Page Down)
	case "pgdown":
		// Moves the cursor down by the height of the textarea
		for i := 0; i < m.ui.Editor.Height(); i++ {
			m.ui.Editor.CursorDown()
		}
		return m, nil

	// 3. Optional: Jump to the absolute top of the file (Alt + G or Ctrl + Home)
	case "alt+g":
		m.ui.Editor.SetCursor(0)
		return m, nil
	case "left":
		if m.ui.EditorCol > 0 {
			m.ui.EditorCol--
		} else if currentLineIdx > 0 {
			prevLine := []rune(lines[currentLineIdx-1])
			m.ui.EditorCol = len(prevLine)
		}
	case "right":
		if m.ui.EditorCol < len(currentLineRunes) {
			m.ui.EditorCol++
		} else if currentLineIdx < len(lines)-1 {
			m.ui.EditorCol = 0
		}
	case "home":
		m.ui.EditorCol = 0
	case "end":
		m.ui.EditorCol = len(currentLineRunes)
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
		//m.contentLoader.WriteContent()
		m.ui.ViewState = model.StateDashboard
		return m, nil
	}
	var cmd tea.Cmd
	m.ui.Editor, cmd = m.ui.Editor.Update(msg)
	updatedLines := strings.Split(m.ui.Editor.Value(), "\n")
	newLineIdx := m.ui.Editor.Line()
	if newLineIdx < len(updatedLines) {
		newLineRunes := []rune(updatedLines[newLineIdx])
		if m.ui.EditorCol > len(newLineRunes) {
			m.ui.EditorCol = len(newLineRunes)
		}
	}
	return m, cmd
}
func (m *appModel) loadFileCmd() tea.Cmd {
	var cmd tea.Cmd
	log.Print("loadFileCmd")
	if len(m.ui.TreeNodes) == 0 || m.ui.SelectedFile < 0 || m.ui.SelectedFile >= len(m.ui.TreeNodes) {
		return cmd
	}
	selectedNode := m.ui.TreeNodes[m.ui.SelectedFile]
	if selectedNode.IsDir {
		return cmd
	}
	languageProvider := m.providerFactory.GetProvider(selectedNode.FullPath)
	m.ui.ActiveLanguageProvider = languageProvider
	log.Printf("loadFileCmd: loading file %sand language provider: %v", selectedNode.FullPath, languageProvider)
	return func() tea.Msg {
		bytes, err := os.ReadFile(selectedNode.FullPath)
		if err != nil {
			return lsp.FileErrorMsg{Err: err}
		}
		return lsp.FileLoadedMsg{Path: selectedNode.FullPath, Content: string(bytes), Provider: languageProvider}
	}
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
