package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
)

func RenderInstaller(m model.UIState) string {
	var boxContent string

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	labelStyle := lipgloss.NewStyle().Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)

	// Intercept execution and block onboarding if runtime is missing git
	if m.GitMissing {
		boxContent = fmt.Sprintf(
			"%s\n\n%s\n\n'git' command binary executable could not be detected in your current system $PATH environment.\n\nPlease install git via your systems package manager and try running Go-Dark again.\n\n\x1b[90mPress [Ctrl+C] to quit application\x1b[0m",
			titleStyle.Render("🛑 Dependency Error"),
			errorStyle.Render("Missing System Dependency: Git Required!"),
		)
	} else if m.InstallerStep == model.StepSetGlobalPrefs {
		boxContent = fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n\x1b[90m%s\x1b[0m",
			titleStyle.Render("🚀 Go-Dark Installer (Step 1 of 2)"),
			labelStyle.Render("1. Global Workspace Base Path:"),
			m.Inputs[0].View(),
			labelStyle.Render("2. Global Git Username (For your code commits):"),
			m.Inputs[1].View(),
			labelStyle.Render("3. Global Git Email Address:"),
			m.Inputs[2].View(),
			"[Tab] Navigate Fields | [Enter] Continue to Project Setup | [Esc] Quit",
		)
	} else {
		boxContent = fmt.Sprintf(
			"%s\n\nWould you like to register your first project repository? (Optional)\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n\x1b[90m%s\x1b[0m",
			titleStyle.Render("🚀 Go-Dark Installer (Step 2 of 2)"),
			labelStyle.Render("Project Display Name:"),
			m.Inputs[3].View(),
			labelStyle.Render("Relative Folder Name (Appended to Base Path):"),
			m.Inputs[4].View(),
			labelStyle.Render("Project Stack Type (java / docker):"),
			m.Inputs[5].View(),
			labelStyle.Render("Git Repository SSH/HTTPS Clone URL:"),
			m.Inputs[6].View(),
			"[Tab] Navigate Fields | [Enter] Complete Setup & Launch Portal | [Esc] Quit",
		)
	}

	installerBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("45")).
		Background(lipgloss.Color("234")).
		Padding(2, 6, 2, 6).
		Width(75).
		Render(boxContent)

	return lipgloss.Place(
		m.TerminalW, m.TerminalH,
		lipgloss.Center, lipgloss.Center,
		installerBox,
		lipgloss.WithWhitespaceChars(" "),
	)
}
