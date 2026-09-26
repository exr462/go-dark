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

//goland:noinspection GoMixedReceiverTypes
func (m appModel) readFileContentCmd() tea.Cmd {
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
