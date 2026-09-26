package main

import tea "github.com/charmbracelet/bubbletea"

func (m *appModel) fileLoad(cmds []tea.Cmd) []tea.Cmd {
	if len(m.state.TreeNodes) > 0 {
		if m.state.SelectedFile >= len(m.state.TreeNodes) {
			m.state.SelectedFile = 0
		}
		cmds = append(cmds, m.readFileContentCmd())
	} else {
		m.state.FileViewer.SetContent("Empty or uncloned project repository.")
	}
	return cmds
}
