package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) settingWelcomePanel() tea.Cmd {
	inputs := initializer.MakeInputs()
	initialState := model.StateSystemCheckModal
	if m.isFirstRun || m.state.GitMissing {
		initialState = model.StateGitConfigurationModal
		if !m.state.GitMissing {
			inputs[0].Focus()
		}
	}
	m.state.ViewState = initialState
	m.state.Inputs = inputs
	return func() tea.Msg {
		return config.VoidMsg{}
	}
}
