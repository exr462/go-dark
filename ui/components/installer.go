package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/renderer"
)

func RenderInstaller(m *model.UIState) string {
	var boxContent string

	// Intercept execution and block onboarding if runtime is missing git
	if m.GitMissing {
		boxContent = fmt.Sprintf(
			"%s\n\n%s\n\n'git' command binary executable could not be detected in your current system $PATH environment.\n\nPlease install git via your systems package manager and try running Go-Dark again.\n\n\x1b[90mPress [Ctrl+C] to quit application\x1b[0m",
			renderer.Title.Render("🛑 Dependency Error"),
			renderer.Error.Render("Missing System Dependency: Git Required!"),
		)
	} else if m.Config.GitUsername == "" && m.Config.GitEmail == "" && m.Config.BasePath == "" {
		boxContent = fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n\x1b[90m%s\x1b[0m",
			renderer.Title.Render("🚀 Go-Dark Installer (Step 1 of 2)"),
			renderer.Bold.Render("1. Global Workspace Base Path:"),
			m.Inputs[0].View(),
			renderer.Bold.Render("2. Global Git Username (For your code commits):"),
			m.Inputs[1].View(),
			renderer.Bold.Render("3. Global Git Email Address:"),
			m.Inputs[2].View(),
			"[Tab] Navigate Fields | [Enter] Continue to Project Setup | [Esc] Quit",
		)
	}

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		renderer.ModalBox.Width(m.WindowWidth-4).Render(boxContent),
		renderer.WhiteSpace,
	)
}
