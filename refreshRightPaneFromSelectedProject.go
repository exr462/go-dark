package main

import (
	"os"

	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m appModel) refreshRightPaneFromSelectedProject() {
	if len(m.state.Config.Projects) == 0 || m.state.SelectedProject >= len(m.state.Config.Projects) {
		return
	}

	m.state.CurrentDirectory = ""
	proj := m.state.Config.Projects[m.state.SelectedProject]
	m.state.TreeNodes = []model.FileNode{}

	if _, err := os.Stat(proj.Path); os.IsNotExist(err) {
		m.state.FileViewer.SetContent("Project repository has not been cloned yet. Press Ctrl+G to clone/checkout.")
		m.state.SelectedFile = 0
		return
	}

	m.buildTreeNodes(proj.Path, 0)
	m.state.SelectedFile = 0
	if len(m.state.TreeNodes) > 0 {
		_ = m.readFileContentCmd()
	} else {
		m.state.FileViewer.SetContent("Empty project directory root.")
	}
}
