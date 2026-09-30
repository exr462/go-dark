package commandfile

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
)

func LoadFileCmd(ui *model.UI, providerFactory *lsp.ProviderFactory) tea.Cmd {
	var cmd tea.Cmd
	log.Print("loadFileCmd")
	if len(ui.TreeNodes) == 0 || ui.SelectedFile < 0 || ui.SelectedFile >= len(ui.TreeNodes) {
		return cmd
	}
	selectedNode := ui.TreeNodes[ui.SelectedFile]
	if selectedNode.IsDir {
		return cmd
	}
	languageProvider := providerFactory.GetProvider(selectedNode.FullPath)
	ui.ActiveLanguageProvider = languageProvider
	log.Printf("loadFileCmd: loading file %sand language provider: %v", selectedNode.FullPath, languageProvider)
	return func() tea.Msg {
		bytes, err := os.ReadFile(selectedNode.FullPath)
		if err != nil {
			return lsp.FileErrorMsg{Err: err}
		}
		return lsp.FileLoadedMsg{Path: selectedNode.FullPath, Content: string(bytes), Provider: languageProvider}
	}
}
