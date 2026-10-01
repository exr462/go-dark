package component

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

var (
	normalModeStyle  = lipgloss.NewStyle().Foreground(color.Crust).Background(color.Green).Bold(true).Padding(0, 1)
	insertModeStyle  = lipgloss.NewStyle().Foreground(color.Crust).Background(color.Yellow).Bold(true).Padding(0, 1)
	commandLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(MochaTeal)).Bold(true)
)

const (
	// editorBoxMargin is how much of the viewport is left outside the editor's
	// own border, on both width and height - kept tiny so the editor behaves
	// like a maximized, full-screen window.
	editorBoxMargin = 2

	// editorChromeOverhead is editorBoxMargin + the outer box's own
	// border/padding (4 rows) + the header row + the footer/status row + the
	// editor view's own trailing blank line, i.e. everything that isn't an
	// actual visible line of file content. maxVisibleLines is sized to fill
	// exactly what's left so the editor uses the full available height.
	editorChromeOverhead = editorBoxMargin + 4 + 1 + 1 + 1
)

// RenderOpenEditor renders the in-app code editor (a bubbles/textarea with
// chroma-based syntax highlighting), including the active file path, dirty
// indicator, a Vim-style Normal/Insert/Command mode indicator and key hints.
func RenderOpenEditor(ui *model.UI) string {
	// The embedded textarea must be explicitly focused or it silently drops
	// every key event (typing, hjkl movement, scrolling) - bubbletea's
	// textarea.Update() no-ops entirely while unfocused. Nothing else in the
	// app ever focuses it, so it must happen here, every render, exactly like
	// the terminal cockpit's input field does.
	ui.Editor.Focus()

	rawEditorText := ui.Editor.Value()

	// Split raw plain text into individual rows
	rawLines := strings.Split(rawEditorText, "\n")
	var editorBuilder strings.Builder

	activeLineIdx := ui.Editor.Line()

	// CORRECT METHOD: Fetches the column character position natively from the textarea engine
	activeColIdx := ui.Editor.LineInfo().CharOffset

	// Use nearly the entire viewport so the editor feels like a maximized,
	// full-screen window rather than a small centered popup. The outer box
	// below reserves editorChromeOverhead rows for its own border/padding
	// plus the header/status-line bars; maxVisibleLines is sized to exactly
	// fill whatever remains so there's no wasted blank space at the bottom.
	maxVisibleLines := ui.WindowHeight - editorChromeOverhead
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

		// Render the cursor precisely onto the current active line. A solid
		// block mimics Vim's Normal-mode cursor; a thin bar mimics Insert
		// mode, so the mode is visible at a glance even without reading text.
		cursorColor := lipgloss.Color("205")
		isBlockCursor := ui.EditorMode != model.EditorModeInsert
		if i == activeLineIdx && ui.Editor.Focused() {
			lineRunes := []rune(rawLine)

			if activeColIdx >= len(lineRunes) {
				// Cursor is at the absolute end of the line
				highlightedText := highlightCode(rawLine, ui.ActiveLanguageProvider.Name())
				glyph := "│"
				if isBlockCursor {
					glyph = "█"
				}
				cursorBlock := lipgloss.NewStyle().Foreground(cursorColor).Render(glyph)
				finalLineContent = highlightedText + cursorBlock
			} else {
				// Cursor is nested inside the text content
				leftStr := string(lineRunes[:activeColIdx])
				charAtCursor := string(lineRunes[activeColIdx])
				rightStr := string(lineRunes[activeColIdx+1:])

				// Highlight text blocks *separately* around the plain cursor coordinate
				highlightedLeft := highlightCode(leftStr, ui.ActiveLanguageProvider.Name())
				highlightedRight := highlightCode(rightStr, ui.ActiveLanguageProvider.Name())

				var cursorBlock string
				if isBlockCursor {
					// Create the highlighted reverse-block cursor style
					cursorBlock = lipgloss.NewStyle().Background(cursorColor).Foreground(lipgloss.Color("0")).Render(charAtCursor)
				} else {
					thinCursor := lipgloss.NewStyle().Foreground(cursorColor).Render("│")
					cursorBlock = thinCursor + highlightCode(charAtCursor, ui.ActiveLanguageProvider.Name())
				}

				// Combine the elements cleanly without mixing ANSI offsets
				finalLineContent = highlightedLeft + cursorBlock + highlightedRight
			}
		} else {
			// This line doesn't have focus, highlight normally
			finalLineContent = highlightCode(rawLine, ui.ActiveLanguageProvider.Name())
		}

		editorBuilder.WriteString(gutter + finalLineContent + "\n")
	}

	// Pad short files with Vim-style "~" filler rows so the content area
	// always occupies exactly maxVisibleLines, regardless of file length.
	// Without this, the footer/shortcut bar would float immediately below
	// whatever little text is on screen instead of staying pinned to the
	// bottom of the editor, which is what made it look "misplaced".
	fillerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	for rendered := endLine - startLine; rendered < maxVisibleLines; rendered++ {
		editorBuilder.WriteString(fillerStyle.Render("    ~") + "\n")
	}

	editorView := lipgloss.NewStyle().Padding(0, 1).Render(editorBuilder.String())

	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(MochaTeal)).Bold(true).Padding(0, 1)
	dirtyBadge := ""
	if ui.EditorDirty {
		dirtyBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).Render(" ● unsaved")
	}
	filePath := ui.ActiveFilePath
	if filePath == "" {
		filePath = "(no file loaded)"
	}
	header := headerStyle.Render("📝 "+filePath) + dirtyBadge

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(MochaOverlay)).Padding(0, 1)
	footer := footerStyle.Render(editorStatusLine(ui))

	fullView := lipgloss.JoinVertical(lipgloss.Left, header, editorView, footer)

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-editorBoxMargin).Height(ui.WindowHeight-editorBoxMargin).Render(fullView),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}

// editorStatusLine renders a Vim-style status/command line: a bold mode
// indicator (-- NORMAL --, -- INSERT --) on the left, or the live ":"
// command buffer while a command is being typed, plus a short key-hint
// reminder on the right.
func editorStatusLine(ui *model.UI) string {
	var left string
	switch ui.EditorMode {
	case model.EditorModeInsert:
		left = insertModeStyle.Render("-- INSERT --")
	case model.EditorModeCommand:
		left = commandLineStyle.Render(":" + ui.EditorCommandBuffer)
	default:
		left = normalModeStyle.Render("-- NORMAL --")
	}

	var hint string
	switch ui.EditorMode {
	case model.EditorModeInsert:
		hint = "esc normal mode"
	case model.EditorModeCommand:
		hint = "enter run · esc cancel"
	default:
		hint = "i insert · hjkl/gg/G move · dd delete · :w save · :q quit · :wq save & quit"
	}

	return left + "   " + hint
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
