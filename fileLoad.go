package main

import tea "github.com/charmbracelet/bubbletea"

func (m *appModel) fileLoad(cmds []tea.Cmd) []tea.Cmd {
	if len(m.ui.TreeNodes) > 0 {
		if m.ui.SelectedFile >= len(m.ui.TreeNodes) {
			m.ui.SelectedFile = 0
		}
		cmds = append(cmds, m.readFileContentCmd())
	} else {
		m.ui.FileViewer.SetContent("Empty or uncloned project repository.")
	}
	return cmds
}
