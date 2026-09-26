package components

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/lsp"
)

type EditFileMsg struct {
	Content string
	Err     error
}

// OpenEditorComponent initializes the external editor process using tea.ExecProcess
func OpenEditorComponent(provider lsp.LanguageProvider, initialContent string) tea.Cmd {
	// FIX: Change function signature from func(ctx context.Context) to func()
	return func() tea.Msg {
		// 1. Determine extension and establish a secure temporary workspace file
		ext := ".txt"
		if len(provider.Extensions()) > 0 {
			ext = provider.Extensions()[0]
		}

		tmpDir, err := os.MkdirTemp("", "lsp-editor-*")
		if err != nil {
			return EditFileMsg{Err: fmt.Errorf("failed to create workspace: %w", err)}
		}
		// Note: defer os.RemoveAll(tmpDir) cannot be cleanly called here because
		// tea.ExecProcess executes asynchronously after this wrapper yields.
		// Instead, let the completion callback remove the file.

		filePath := filepath.Join(tmpDir, "buffer"+ext)
		if err := os.WriteFile(filePath, []byte(initialContent), 0o600); err != nil {
			err := os.RemoveAll(tmpDir)
			if err != nil {
				return nil
			}
			return EditFileMsg{Err: fmt.Errorf("failed to stage code file: %w", err)}
		}

		// 2. Fetch system editor or fall back to standard text configurations
		editorPath := os.Getenv("EDITOR")
		if editorPath == "" {
			editorPath = "vim"
		}

		editorCmd := exec.Command(editorPath, filePath)

		// 3. Optional: Wire up the LSP server command background configuration if needed
		lspConfig := provider.GetLSPConfig()
		if lspConfig.ServerBinary != "" {
			_ = exec.Command(lspConfig.ServerBinary, lspConfig.Args...)
		}

		// 4. Wrap execution parameters inside a Bubble Tea Exec Command
		execCmd := tea.ExecProcess(editorCmd, func(err error) tea.Msg {
			// Clean up workspace directory when editor finishes execution loop
			defer func(path string) {
				_ = os.RemoveAll(path)
			}(tmpDir)

			if err != nil {
				return EditFileMsg{Err: fmt.Errorf("editor session crashed: %w", err)}
			}

			// Read updated modifications once the editor wraps up its cycle
			updatedContent, readErr := os.ReadFile(filePath)
			if readErr != nil {
				return EditFileMsg{Err: fmt.Errorf("failed to read workspace buffer: %w", readErr)}
			}

			return EditFileMsg{Content: string(updatedContent)}
		})

		// FIX: Invoke the Exec target directly without a manual context injection pass
		return execCmd()
	}
}
