package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/renderer"
)

var (
	installerHint = lipgloss.NewStyle().Foreground(color.Overlay0)
	installerKey  = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderInstaller(m *model.UIState) string {
	var boxContent string

	if m.GitMissing {
		boxContent = fmt.Sprintf(
			"%s\n\n%s\n\n'git' command binary executable could not be detected in your current system $PATH environment.\n\nPlease install git via your system's package manager and try running Go-Dark again.\n\n%s",
			renderer.Title.Render("🛑 Dependency Error"),
			renderer.Error.Render("Missing System Dependency: Git Required!"),
			installerHint.Render(fmt.Sprintf("Press [%s] to quit application", installerKey.Render("Ctrl+C"))),
		)
	} else if m.InstallerStep == model.StepSetGlobalPrefs {
		boxContent = fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
			renderer.Title.Render("🚀 Global Preferences Setup (Step 1 of 2)"),
			renderer.Bold.Render("1. Global Workspace Base Path:"),
			m.Inputs[0].View(),
			renderer.Bold.Render("2. Global Git Username (For your code commits):"),
			m.Inputs[1].View(),
			renderer.Bold.Render("3. Global Git Email Address:"),
			m.Inputs[2].View(),
			installerHint.Render(fmt.Sprintf(
				"[%s] Navigate Fields | [%s] Continue to Project Setup | [%s] Exit",
				installerKey.Render("Tab"),
				installerKey.Render("Enter"),
				installerKey.Render("Esc"),
			)),
		)
	} else {
		boxContent = fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
			renderer.Title.Render("📁 Add First Workspace Project (Step 2 of 2)"),
			renderer.Bold.Render("1. Project Name:"),
			m.Inputs[3].View(),
			renderer.Bold.Render("2. Relative Folder:"),
			m.Inputs[4].View(),
			renderer.Bold.Render("3. Stack Type (java, docker):"),
			m.Inputs[5].View(),
			renderer.Bold.Render("4. Git Clone URL:"),
			m.Inputs[6].View(),
			installerHint.Render(fmt.Sprintf(
				"[%s] Navigate | [%s] Complete Setup & Launch | [%s] Skip",
				installerKey.Render("Tab"),
				installerKey.Render("Enter"),
				installerKey.Render("Esc"),
			)),
		)
	}

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		renderer.ModalBox.Width(min(m.WindowWidth-4, 85)).Render(boxContent),
		renderer.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
