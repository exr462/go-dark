package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func main() {
	f, err := initLogger()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize log file: %v\n", err)
		os.Exit(1)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	cfg, isFirstRun := config.LoadConfig()
	_, gitErr := exec.LookPath("git")
	gitMissing := gitErr != nil
	home, _ := os.UserHomeDir()

	inputs := make([]textinput.Model, model.Ceiling)
	for i := range inputs {
		inputs[i] = textinput.New()
	}

	inputs[model.GitWorkspace].Placeholder = "Global Workspace Base Path"
	inputs[model.GitWorkspace].SetValue(filepath.Join(home, "workspace"))
	inputs[model.GitUsername].Placeholder = "e.g. John Doe"
	inputs[model.GitEmail].Placeholder = "e.g. john@example.com"
	inputs[model.JdkName].Placeholder = "Profile Name (e.g. Java-17)"
	inputs[model.JdkPath].Placeholder = "JAVA_HOME path (e.g. /usr/lib/jvm/...)"
	inputs[model.MvnName].Placeholder = "Maven Profile Name (e.g. Maven-3.9)"
	inputs[model.MvnPath].Placeholder = "MAVEN_HOME directory path"

	initialState := model.StateDashboard
	if isFirstRun || gitMissing {
		initialState = model.StateGitConfigurationModal
		if !gitMissing {
			inputs[0].Focus()
		}
	}

	uiState.Config = cfg
	uiState.ViewState = initialState
	uiState.Inputs = inputs
	uiState.GitMissing = gitMissing

	m := &appModel{
		state: uiState,
	}
	m.state.FuzzyQueryInput.Placeholder = "Type lookup phrase (e.g. controller)..."
	m.state.FuzzyQueryInput.CharLimit = 50

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}

func initLogger() (*os.File, error) {
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	log.SetOutput(f)
	log.Println("--- TUI Engine Session Started ---")
	return f, nil
}
