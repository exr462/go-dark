package main

import (
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/env"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/storage"
)

func main() {
	// `go-dark env ...` is the installer's command-line side: the installer
	// screen runs the lifecycle actions through it as sessions.
	if len(os.Args) > 1 && os.Args[1] == "env" {
		os.Exit(env.CLI(os.Args[2:], os.Stdout))
	}
	f := initLogger()
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	ui.ViewState = model.StateSystemCheckModal
	m := &appModel{
		ui:            ui,
		contentLoader: storage.NewContentLoader(),
	}
	m.ui.Shells = shell.NewManager()
	// Don't leave shells or builds running behind once go-dark exits.
	defer m.ui.Shells.Shutdown()
	m.ui.Inputs = initializer.MakeInputs()
	m.ui.FuzzyQueryInput.Placeholder = "Type lookup phrase (e.g. controller)..."
	m.ui.FuzzyQueryInput.CharLimit = 50
	m.ui.FuzzyQueryInput.SetWidth(m.ui.FuzzyQueryInput.CharLimit)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		m.ui.Shells.Shutdown()
		os.Exit(1)
	}
}

func initLogger() *os.File {
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize log file: %v\n", err)
		os.Exit(1)
	}
	log.SetOutput(f)
	log.Println("--- Go Dark Started ---")
	return f
}
