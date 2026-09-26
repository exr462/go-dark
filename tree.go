package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
)

type WorkspaceRefreshedMsg struct {
	Files []string
}

func (m *appModel) updateWorkspaceFiles() tea.Cmd {
	if len(m.state.Config.Projects) == 0 {
		return nil
	}

	idx := m.state.SelectedProject
	if idx < 0 || idx >= len(m.state.Config.Projects) {
		return nil
	}

	proj := m.state.Config.Projects[idx]
	fullPath := proj.Path

	return func() tea.Msg {
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return WorkspaceRefreshedMsg{Files: []string{}}
		}

		var projectFiles []string
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				projectFiles = append(projectFiles, e.Name())
			}
		}

		return WorkspaceRefreshedMsg{
			Files: projectFiles,
		}
	}
}

func (m *appModel) readFileContentCmd() tea.Cmd {
	return func() tea.Msg {
		if len(m.state.TreeNodes) == 0 || m.state.SelectedFile >= len(m.state.TreeNodes) {
			return model.StatusMsg("No files in active workspace hierarchy.")
		}

		node := m.state.TreeNodes[m.state.SelectedFile]
		if node.IsDir {
			return model.StatusMsg(fmt.Sprintf("Directory selected: %s", node.Name))
		}

		ext := strings.ToLower(filepath.Ext(node.Name))
		isTextFile := ext == ".go" || ext == ".java" || ext == ".xml" || ext == ".json" ||
			ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".txt" ||
			ext == ".md" || ext == ".sh" || ext == ".sql" || ext == ".kt" || ext == ".mod" ||
			ext == ".sum" || ext == ".html" || ext == ".css" || ext == ".js" || ext == ".ts" ||
			node.Name == "Dockerfile" || node.Name == "pom.xml" || node.Name == ".gitignore"

		if !isTextFile {
			notice := fmt.Sprintf("\n  📦 [BINARY ARTIFACT] Previews Blocked\n\n  File: %s\n  Type: Compiled Binary / Asset\n\n  Go-Dark blocks loading binary formats to prevent terminal encoding distortion.", node.Name)
			m.state.FileViewer.SetContent(lipgloss.NewStyle().Foreground(color.Overlay0).Italic(true).Render(notice))
			return model.StatusMsg(fmt.Sprintf("⚠️ Blocked binary file preview: %s", node.Name))
		}

		data, err := os.ReadFile(node.FullPath)
		if err != nil {
			return model.StatusMsg(fmt.Sprintf("Failed to read file: %v", err))
		}

		m.state.FileViewer.SetContent(string(data))
		m.state.ActiveCodeBuffer = string(data)
		return model.StatusMsg(fmt.Sprintf("Inspecting file: %s", node.Name))
	}
}

func (m *appModel) buildTreeNodes(currentPath string, depth int) {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return
	}

	for _, e := range entries {
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

func (m *appModel) rebuildActiveTree() {
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

func (m *appModel) loadDirectory(dirPath string) ([]model.FileNode, error) {
	if len(m.state.Config.Projects) == 0 || m.state.SelectedProject >= len(m.state.Config.Projects) {
		return nil, fmt.Errorf("no project selected")
	}
	proj := m.state.Config.Projects[m.state.SelectedProject]
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

func (m *appModel) refreshRightPaneFromSelectedProject() {
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
