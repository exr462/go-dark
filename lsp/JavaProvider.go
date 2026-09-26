package lsp

import "os/exec"

type JavaProvider struct{}

func (j JavaProvider) Name() string         { return "Java" }
func (j JavaProvider) Extensions() []string { return []string{".java"} }

func (j JavaProvider) GetLSPConfig() LspConfig {
	return LspConfig{
		ServerBinary: "jdtls", // Eclipse JDT Language Server
		Args:         []string{"-data", "~/.workspace"},
	}
}

func (j JavaProvider) GetBuildCommand(filePath string) *exec.Cmd {
	return exec.Command("javac", filePath)
}

func (j JavaProvider) GetRunCommand(filePath string) *exec.Cmd {
	// Simple run logic; real IDEs parse the package/class name
	return exec.Command("java", filePath)
}
