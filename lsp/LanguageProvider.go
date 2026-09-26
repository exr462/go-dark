package lsp

import "os/exec"

type LanguageProvider interface {
	Name() string
	Extensions() []string
	GetLSPConfig() LspConfig
	GetBuildCommand(filePath string) *exec.Cmd
	GetRunCommand(filePath string) *exec.Cmd
}
