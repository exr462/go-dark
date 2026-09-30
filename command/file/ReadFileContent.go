package commandfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/state"
	"github.com/exr462/go-dark/ui/color"
)

//goland:noinspection GoMixedReceiverTypes
func ReadFileContent(ui *model.UI) tea.Cmd {
	return func() tea.Msg {
		if len(ui.TreeNodes) == 0 || ui.SelectedFile >= len(ui.TreeNodes) {
			return state.StatusMsg("No files in active workspace hierarchy.")
		}

		node := ui.TreeNodes[ui.SelectedFile]
		if node.IsDir {
			return state.StatusMsg(fmt.Sprintf("Directory selected: %s", node.Name))
		}

		ext := strings.ToLower(filepath.Ext(node.Name))
		isTextFile := ext == ".go" || ext == ".java" || ext == ".xml" || ext == ".json" ||
			ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".txt" ||
			ext == ".md" || ext == ".sh" || ext == ".sql" || ext == ".kt" || ext == ".mod" ||
			ext == ".sum" || ext == ".html" || ext == ".css" || ext == ".js" || ext == ".ts" ||
			node.Name == "Dockerfile" || node.Name == "pom.xml" || node.Name == ".gitignore"

		if !isTextFile {
			notice := fmt.Sprintf("\n  📦 [BINARY ARTIFACT] Previews Blocked\n\n  File: %s\n  Type: Compiled Binary / Asset\n\n  Go-Dark blocks loading binary formats to prevent terminal encoding distortion.", node.Name)
			ui.FileViewer.SetContent(lipgloss.NewStyle().Foreground(color.Overlay0).Italic(true).Render(notice))
			return state.StatusMsg(fmt.Sprintf("⚠️ Blocked binary file preview: %s", node.Name))
		}

		data, err := os.ReadFile(node.FullPath)
		if err != nil {
			return state.StatusMsg(fmt.Sprintf("Failed to read file: %v", err))
		}

		ui.FileViewer.SetContent(string(data))
		ui.ActiveCodeBuffer = string(data)
		return state.StatusMsg(fmt.Sprintf("Inspecting file: %s", node.Name))
	}
}
