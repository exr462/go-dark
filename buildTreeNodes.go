package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m appModel) buildTreeNodes(currentPath string, depth int) {
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

		fullNodePath := filepath.Join(currentPath, name)

		node := model.FileNode{
			Name:     name,
			FullPath: fullNodePath,
			IsDir:    e.IsDir(),
			Depth:    depth,
		}
		m.state.TreeNodes = append(m.state.TreeNodes, node)
	}
}
