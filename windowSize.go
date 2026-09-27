package main

import tea "github.com/charmbracelet/bubbletea"

func (m *appModel) windowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.ui.WindowWidth = msg.Width
	m.ui.WindowHeight = msg.Height
	m.ui.FileViewer.Width = (msg.Width / 2) - 4
	m.ui.FileViewer.Height = max(msg.Height-8, 5)
	m.ui.FuzzyViewer.Width = (msg.Width / 2) - 4
	m.ui.FuzzyViewer.Height = max(msg.Height-12, 5)
	return m, nil
}
