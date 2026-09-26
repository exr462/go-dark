package lsp

import "os/exec"

type GoProvider struct{}

func (g GoProvider) Name() string         { return "Go" }
func (g GoProvider) Extensions() []string { return []string{".go"} }
func (g GoProvider) GetLSPConfig() LspConfig {
	return LspConfig{
		ServerBinary: "gopls",
		Args:         []string{"serve"},
	}
}
func (g GoProvider) GetBuildCommand(filePath string) *exec.Cmd {
	// Build output straight into a generic binary string sequence mapping
	return exec.Command("go", "build", "-o", "bin_output", filePath)
}
func (g GoProvider) GetRunCommand(filePath string) *exec.Cmd {
	return exec.Command("go", "run", filePath)
}
