package componentaction

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

func GitCheckoutComplete(ui *model.UI, msg config.GitCheckoutCompleteMsg) tea.Cmd {
	if msg.Err != nil {
		ui.StatusMsg = fmt.Sprintf("❌ %v", msg.Err)
	} else {
		ui.StatusMsg = fmt.Sprintf("✅ %s", strings.TrimSpace(msg.Output))
		// Refresh fetched status across projects
		for i := range ui.Config.Projects {
			gitDir := filepath.Join(ui.Config.Projects[i].Path, ".git")
			if _, err := os.Stat(gitDir); err == nil {
				ui.Config.Projects[i].Fetched = true
			}
		}
		_ = config.SaveConfig(ui.Config)
	}
	ui.ViewState = model.StateDashboard
	return initializer.InitializeWorkspace(ui).OnAction()
}
