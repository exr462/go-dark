package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) settingWelcomePanel() tea.Cmd {
	inputs := initializer.MakeInputs()
	initialState := model.StateSystemCheckModal
	if m.ui.IsFirstRun || m.ui.GitMissing {
		initialState = model.StateGitConfigurationModal
		if !m.ui.GitMissing {
			inputs[0].Focus()
		}
	}
	m.ui.ViewState = initialState
	m.ui.Inputs = inputs
	return func() tea.Msg {
		return config.VoidMsg{}
	}
}
