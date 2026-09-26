package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m appModel) rebuildActiveTree() {
	if len(m.state.Config.Projects) == 0 {
		return
	}
	if m.state.SelectedProject < 0 || m.state.SelectedProject >= len(m.state.Config.Projects) {
		return
	}
	proj := m.state.Config.Projects[m.state.SelectedProject]
	rootPath := proj.Path

	expandedPaths := make(map[string]bool)
	for _, n := range m.state.TreeNodes {
		if n.IsDir && n.IsExpanded {
			expandedPaths[n.FullPath] = true
		}
	}

	var freshTree []model.FileNode

	var walkDir func(string, int)
	walkDir = func(currentPath string, depth int) {
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return
		}
		for _, e := range entries {
			//goland:noinspection DuplicatedCode
			name := e.Name()
			if strings.HasPrefix(name, ".") && name != ".gitignore" {
				continue
			}

			if e.IsDir() && (name == "target" || name == "build" || name == "node_modules" || name == "bin" || name == ".git") {
				continue
			}

			ext := strings.ToLower(filepath.Ext(name))
			if ext == ".jar" || ext == ".war" || ext == ".zip" || ext == ".class" || ext == ".exe" || ext == ".png" || ext == ".jpg" {
				continue
			}
			fullP := filepath.Join(currentPath, name)
			isExp := expandedPaths[fullP]

			node := model.FileNode{
				Name:       name,
				FullPath:   fullP,
				IsDir:      e.IsDir(),
				IsExpanded: isExp,
				Depth:      depth,
			}
			freshTree = append(freshTree, node)

			if node.IsDir && isExp {
				walkDir(fullP, depth+1)
			}
		}
	}

	walkDir(rootPath, 0)
	m.state.TreeNodes = freshTree
}
