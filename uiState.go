package main

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"github.com/exr462/go-dark/model"
)

var ui = &model.UI{
	ActiveFocus:     model.FocusProjects,
	FileViewer:      viewport.New(viewport.WithWidth(30), viewport.WithHeight(20)),
	GitCommands:     []string{"checkout", "clone", "pull", "fetch", "status", "reset"},
	BuildLogs:       []string{"Console ready. Select option step to launch..."},
	Sessions:        make(map[int]*model.BuildSession),
	FuzzyQueryInput: textinput.New(),
	FuzzyViewer:     viewport.New(viewport.WithWidth(30), viewport.WithHeight(20)),
}
