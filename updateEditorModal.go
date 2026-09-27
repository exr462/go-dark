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
	rawText := m.state.Editor.Value()
	lines := strings.Split(rawText, "\n")
	currentLineIdx := m.state.Editor.Line()

	var currentLineRunes []rune
	if currentLineIdx < len(lines) {
		currentLineRunes = []rune(lines[currentLineIdx])
	}
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Escape):
		// returning to modal
		m.state.ViewState = model.StateDashboard
		return m, nil

		// 1. Scroll Up a block of lines (Page Up)
	case "pgup":
		// Moves the cursor up by the height of the textarea
		for i := 0; i < m.state.Editor.Height(); i++ {
			m.state.Editor.CursorUp()
		}
		return m, nil

	// 2. Scroll Down a block of lines (Page Down)
	case "pgdown":
		// Moves the cursor down by the height of the textarea
		for i := 0; i < m.state.Editor.Height(); i++ {
			m.state.Editor.CursorDown()
		}
		return m, nil

	// 3. Optional: Jump to the absolute top of the file (Alt + G or Ctrl + Home)
	case "alt+g":
		m.state.Editor.SetCursor(0)
		return m, nil
	case "left":
		if m.state.EditorCol > 0 {
			m.state.EditorCol--
		} else if currentLineIdx > 0 {
			prevLine := []rune(lines[currentLineIdx-1])
			m.state.EditorCol = len(prevLine)
		}
	case "right":
		if m.state.EditorCol < len(currentLineRunes) {
			m.state.EditorCol++
		} else if currentLineIdx < len(lines)-1 {
			m.state.EditorCol = 0
		}
	case "home":
		m.state.EditorCol = 0
	case "end":
		m.state.EditorCol = len(currentLineRunes)
	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save):
		//m.contentLoader.WriteContent()
		m.state.ViewState = model.StateDashboard
		return m, nil
	}
	var cmd tea.Cmd
	m.state.Editor, cmd = m.state.Editor.Update(msg)
	updatedLines := strings.Split(m.state.Editor.Value(), "\n")
	newLineIdx := m.state.Editor.Line()
	if newLineIdx < len(updatedLines) {
		newLineRunes := []rune(updatedLines[newLineIdx])
		if m.state.EditorCol > len(newLineRunes) {
			m.state.EditorCol = len(newLineRunes)
		}
	}
	return m, cmd
}
func (m *appModel) loadFileCmd() tea.Cmd {
	var cmd tea.Cmd
	log.Print("loadFileCmd")
	if len(m.state.TreeNodes) == 0 || m.state.SelectedFile < 0 || m.state.SelectedFile >= len(m.state.TreeNodes) {
		return cmd
	}
	selectedNode := m.state.TreeNodes[m.state.SelectedFile]
	if selectedNode.IsDir {
		return cmd
	}
	languageProvider := m.providerFactory.GetProvider(selectedNode.FullPath)
	m.state.ActiveLanguageProvider = languageProvider
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
