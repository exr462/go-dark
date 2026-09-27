package components

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

type EditFileMsg struct {
	Content string
	Err     error
}

// RenderOpenEditor initializes the external editor process using tea.ExecProcess
func RenderOpenEditor(ui *model.UI) string {
	rawEditorText := ui.Editor.Value()

	// Split raw plain text into individual rows
	rawLines := strings.Split(rawEditorText, "\n")
	var editorBuilder strings.Builder

	activeLineIdx := ui.Editor.Line()

	// CORRECT METHOD: Fetches the column character position natively from the textarea engine
	activeColIdx := ui.Editor.LineInfo().CharOffset

	// Configure visible scroll viewport boundaries
	maxVisibleLines := ui.WindowHeight - 8
	if maxVisibleLines < 1 {
		maxVisibleLines = 1
	}

	startLine := 0
	if activeLineIdx >= maxVisibleLines {
		startLine = activeLineIdx - maxVisibleLines + 1
	}
	endLine := startLine + maxVisibleLines
	if endLine > len(rawLines) {
		endLine = len(rawLines)
	}

	for i := startLine; i < endLine; i++ {
		lineNumberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
		if i == activeLineIdx {
			lineNumberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		}
		gutter := lineNumberStyle.Render(fmt.Sprintf("%3d │ ", i+1))

		rawLine := rawLines[i]
		var finalLineContent string

		// Render the cursor precisely onto the current active line
		if i == activeLineIdx && ui.Editor.Focused() {
			lineRunes := []rune(rawLine)

			if activeColIdx >= len(lineRunes) {
				// Cursor is at the absolute end of the line
				highlightedText := highlightCode(rawLine, ui.ActiveLanguageProvider.Name())
				cursorBlock := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("█")
				finalLineContent = highlightedText + cursorBlock
			} else {
				// Cursor is nested inside the text content
				leftStr := string(lineRunes[:activeColIdx])
				charAtCursor := string(lineRunes[activeColIdx])
				rightStr := string(lineRunes[activeColIdx+1:])

				// Highlight text blocks *separately* around the plain cursor coordinate
				highlightedLeft := highlightCode(leftStr, ui.ActiveLanguageProvider.Name())
				highlightedRight := highlightCode(rightStr, ui.ActiveLanguageProvider.Name())

				// Create the highlighted reverse-block cursor style
				cursorBlock := lipgloss.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0")).Render(charAtCursor)

				// Combine the elements cleanly without mixing ANSI offsets
				finalLineContent = highlightedLeft + cursorBlock + highlightedRight
			}
		} else {
			// This line doesn't have focus, highlight normally
			finalLineContent = highlightCode(rawLine, ui.ActiveLanguageProvider.Name())
		}

		editorBuilder.WriteString(gutter + finalLineContent + "\n")
	}

	editorView := lipgloss.NewStyle().Padding(0, 1).Render(editorBuilder.String())

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4).Height(ui.WindowHeight-4).Render(editorView),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}

func highlightCode(content string, language string) string {
	// 1. Match the right lexer generically based on our language string
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// 2. Select a classic terminal-friendly theme (e.g., monokai, dracula, native)
	styleRegistry := styles.Get("monokai")
	if styleRegistry == nil {
		styleRegistry = styles.Fallback
	}

	// 3. Request 256-color TrueColor ANSI console sequencing strings
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, styleRegistry, iterator)
	if err != nil {
		return content
	}

	return buf.String()
}
