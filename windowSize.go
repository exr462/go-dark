package main

import tea "github.com/charmbracelet/bubbletea"

func (m *appModel) windowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.state.WindowWidth = msg.Width
	m.state.WindowHeight = msg.Height
	m.state.FileViewer.Width = (msg.Width / 2) - 4
	m.state.FileViewer.Height = max(msg.Height-8, 5)
	m.state.FuzzyViewer.Width = (msg.Width / 2) - 4
	m.state.FuzzyViewer.Height = max(msg.Height-12, 5)
	return m, nil
}
