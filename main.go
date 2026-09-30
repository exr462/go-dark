package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/storage"
)

func main() {
	f := initLogger()
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	ui.ViewState = model.StateSystemCheckModal
	m := &appModel{
		ui:            ui,
		contentLoader: storage.NewContentLoader(),
	}
	m.ui.Inputs = initializer.MakeInputs()
	m.ui.FuzzyQueryInput.Placeholder = "Type lookup phrase (e.g. controller)..."
	m.ui.FuzzyQueryInput.CharLimit = 50

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
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
