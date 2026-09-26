package lsp

import "os/exec"

type LanguageProvider interface {
	Name() string
	Extensions() []string
	GetLSPConfig() LSPConfig
	GetBuildCommand(filePath string) *exec.Cmd
	GetRunCommand(filePath string) *exec.Cmd
}

type LSPConfig struct {
	ServerBinary string
	Args         []string
}
