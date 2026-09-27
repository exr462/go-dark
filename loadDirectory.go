package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m appModel) loadDirectory(dirPath string) ([]model.FileNode, error) {
	if len(m.ui.Config.Projects) == 0 || m.ui.SelectedProject >= len(m.ui.Config.Projects) {
		return nil, fmt.Errorf("no project selected")
	}
	proj := m.ui.Config.Projects[m.ui.SelectedProject]
	targetPath := filepath.Join(proj.Path, dirPath)
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, err
	}

	var nodes []model.FileNode
	for _, entry := range entries {
		if entry.Name()[0] == '.' && entry.Name() != ".gitignore" {
			continue
		}

		nodes = append(nodes, model.FileNode{
			Name:     entry.Name(),
			FullPath: filepath.Join(targetPath, entry.Name()),
			IsDir:    entry.IsDir(),
		})
	}

	return nodes, nil
}
