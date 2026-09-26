package lsp

// DefaultProvider acts as a generic fallback for unsupported file types
import (
	"os/exec"
)

type DefaultProvider struct{}

func (d DefaultProvider) Name() string         { return "Plain Text" }
func (d DefaultProvider) Extensions() []string { return []string{".txt"} }
func (d DefaultProvider) GetLSPConfig() LspConfig {
	return LspConfig{ServerBinary: "", Args: []string{}} // No LSP support
}
func (d DefaultProvider) GetBuildCommand(filePath string) *exec.Cmd { return nil }
func (d DefaultProvider) GetRunCommand(filePath string) *exec.Cmd   { return nil }
