package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) syncFuzzyPreviewPane() {
	if len(m.state.FuzzyResults) == 0 || m.state.SelectedFuzzy >= len(m.state.FuzzyResults) {
		m.state.FuzzyViewer.SetContent("No file selected for preview.")
		return
	}

	res := m.state.FuzzyResults[m.state.SelectedFuzzy]
	ext := strings.ToLower(filepath.Ext(res.FileName))
	isTextFile := ext == ".go" || ext == ".kt" || ext == ".java" || ext == ".xml" || ext == ".json" ||
		ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".jsx" || ext == ".tsx" || ext == ".ts" || ext == ".txt" ||
		ext == ".md" || ext == ".sh" || ext == ".sql" || ext == ".mod" || ext == ".sum" || res.FileName == "Dockerfile" || res.FileName == "pom.xml"

	if !isTextFile {
		notice := fmt.Sprintf("\n  📦 [BINARY ARTIFACT] Previews Blocked\n\n  File: %s\n\n  Fuzzy preview is disabled for compiled binary artifacts.", res.FileName)
		m.state.FuzzyViewer.SetContent(lipgloss.NewStyle().Foreground(color.Overlay0).Italic(true).Render(notice))
		return
	}

	data, err := os.ReadFile(res.FullPath)
	if err != nil {
		m.state.FuzzyViewer.SetContent(fmt.Sprintf("❌ Error opening preview: %v", err))
		return
	}

	m.state.FuzzyViewer.SetContent(string(data))

	if m.state.FuzzyMode == model.FuzzyModeContent && res.LineNum > 0 {
		m.state.FuzzyViewer.GotoTop()
		for i := 0; i < res.LineNum-3 && i < m.state.FuzzyViewer.Height; i++ {
			m.state.FuzzyViewer.LineDown(1)
		}
	} else {
		m.state.FuzzyViewer.GotoTop()
	}
}
