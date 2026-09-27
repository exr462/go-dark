package main

import (
	"os"

	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m appModel) refreshRightPaneFromSelectedProject() {
	if len(m.ui.Config.Projects) == 0 || m.ui.SelectedProject >= len(m.ui.Config.Projects) {
		return
	}

	m.ui.CurrentDirectory = ""
	proj := m.ui.Config.Projects[m.ui.SelectedProject]
	m.ui.TreeNodes = []model.FileNode{}

	if _, err := os.Stat(proj.Path); os.IsNotExist(err) {
		m.ui.FileViewer.SetContent("Project repository has not been cloned yet. Press Ctrl+G to clone/checkout.")
		m.ui.SelectedFile = 0
		return
	}

	m.buildTreeNodes(proj.Path, 0)
	m.ui.SelectedFile = 0
	if len(m.ui.TreeNodes) > 0 {
		_ = m.readFileContentCmd()
	} else {
		m.ui.FileViewer.SetContent("Empty project directory root.")
	}
}
