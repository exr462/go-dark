package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/exr462/go-dark/model"
)

var uiState = &model.UIState{
	InstallerStep:   model.StepSetGlobalPrefs,
	ActiveFocus:     model.FocusProjects,
	FileViewer:      viewport.New(30, 20),
	GitCommands:     []string{"checkout", "clone", "pull", "fetch", "status", "reset"},
	BuildOptions:    []string{"clean", "test", "compile", "package", "without tests", "full"},
	BuildLogs:       []string{"Console ready. Select option step to launch..."},
	Sessions:        make(map[int]*model.BuildSession),
	FuzzyQueryInput: textinput.New(),
	FuzzyViewer:     viewport.New(30, 20),
}
