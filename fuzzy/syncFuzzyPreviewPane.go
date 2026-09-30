package fuzzy

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
func FuzzyPreviewPane(ui *model.UI) {
	if len(ui.FuzzyResults) == 0 || ui.SelectedFuzzy >= len(ui.FuzzyResults) {
		ui.FuzzyViewer.SetContent("No file selected for preview.")
		return
	}

	res := ui.FuzzyResults[ui.SelectedFuzzy]
	ext := strings.ToLower(filepath.Ext(res.FileName))
	isTextFile := ext == ".go" || ext == ".kt" || ext == ".java" || ext == ".xml" || ext == ".json" ||
		ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".jsx" || ext == ".tsx" || ext == ".ts" || ext == ".txt" ||
		ext == ".md" || ext == ".sh" || ext == ".sql" || ext == ".mod" || ext == ".sum" || res.FileName == "Dockerfile" || res.FileName == "pom.xml"

	if !isTextFile {
		notice := fmt.Sprintf("\n  📦 [BINARY ARTIFACT] Previews Blocked\n\n  File: %s\n\n  Fuzzy preview is disabled for compiled binary artifacts.", res.FileName)
		ui.FuzzyViewer.SetContent(lipgloss.NewStyle().Foreground(color.Overlay0).Italic(true).Render(notice))
		return
	}

	data, err := os.ReadFile(res.FullPath)
	if err != nil {
		ui.FuzzyViewer.SetContent(fmt.Sprintf("❌ Error opening preview: %v", err))
		return
	}

	ui.FuzzyViewer.SetContent(string(data))

	if ui.FuzzyMode == model.FuzzyModeContent && res.LineNum > 0 {
		ui.FuzzyViewer.GotoTop()
		for i := 0; i < res.LineNum-3 && i < ui.FuzzyViewer.Height; i++ {
			ui.FuzzyViewer.ScrollDown(1)
		}
	} else {
		ui.FuzzyViewer.GotoTop()
	}
}
