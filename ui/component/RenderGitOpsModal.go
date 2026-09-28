package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	gitLocalTag   = lipgloss.NewStyle().Foreground(color.Teal).Bold(true)
	gitRemoteTag  = lipgloss.NewStyle().Foreground(color.Overlay0)
	gitProjHeader = lipgloss.NewStyle().Foreground(color.Blue).Bold(true)
	gitPathHeader = lipgloss.NewStyle().Foreground(color.Subtext0).Italic(true)
	gitHintStyle  = lipgloss.NewStyle().Foreground(color.Overlay0)
	gitKeyHint    = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderGitOpsModal(ui *model.UI) string {
	var modalBody strings.Builder

	modalBody.WriteString(decorator.Title.Render("⚡ Git Hub Central Control (Ctrl+G)") + "\n\n")

	switch ui.GitOperationStep {
	case 0:
		modalBody.WriteString(decorator.Section.Render("👉 Step 1: Select Target Project Repository:") + "\n\n")

		maxVisible := max(ui.WindowHeight-14, 6)
		startIdx := 0
		if ui.SelectedGitProject >= maxVisible {
			startIdx = ui.SelectedGitProject - maxVisible + 1
		}
		endIdx := min(startIdx+maxVisible, len(ui.Config.Projects))

		if startIdx > 0 {
			modalBody.WriteString(gitHintStyle.Render(fmt.Sprintf("  ↑ ... (%d more above)", startIdx)) + "\n")
		}

		for i := startIdx; i < endIdx; i++ {
			p := ui.Config.Projects[i]
			statusBadge := gitLocalTag.Render("[local]")
			if !p.Fetched {
				statusBadge = gitRemoteTag.Render("[remote]")
			}

			lineText := fmt.Sprintf("%s %s (%s)", p.Name, statusBadge, p.GitURL)
			if len(lineText) > ui.WindowWidth-14 {
				lineText = lineText[:ui.WindowWidth-17] + "..."
			}

			if i == ui.SelectedGitProject {
				modalBody.WriteString(decorator.Selected.Render("> "+lineText) + "\n")
			} else {
				modalBody.WriteString(decorator.Inactive.Render("  "+lineText) + "\n")
			}
		}

		if endIdx < len(ui.Config.Projects) {
			modalBody.WriteString(gitHintStyle.Render(fmt.Sprintf("  ↓ ... (%d more below)", len(ui.Config.Projects)-endIdx)) + "\n")
		}

		modalBody.WriteString("\n" + gitHintStyle.Render(fmt.Sprintf(
			"[%s] Navigate | [%s] Select Project & Continue | [%s] Close",
			gitKeyHint.Render("↑/↓/j/k"),
			gitKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			gitKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)))

	case 1:
		targetProj := ui.Config.Projects[ui.SelectedGitProject]
		statusDesc := gitLocalTag.Render("Cloned locally")
		if !targetProj.Fetched {
			statusDesc = gitRemoteTag.Render("Remote repository (will clone to local workspace on checkout)")
		}

		modalBody.WriteString(fmt.Sprintf("📦 Project: %s  |  Status: %s\n", gitProjHeader.Render(targetProj.Name), statusDesc))
		modalBody.WriteString(fmt.Sprintf("📁 Local Path: %s\n\n", gitPathHeader.Render(targetProj.Path)))
		modalBody.WriteString(decorator.Section.Render("👉 Step 2: Choose Git Operation to Execute:") + "\n\n")

		for i, cmd := range ui.GitCommands {
			cmdDesc := ""
			switch cmd {
			case "checkout":
				if !targetProj.Fetched {
					cmdDesc = " - Clone repo and checkout target branch"
				} else {
					cmdDesc = " - Switch active working branch"
				}
			case "clone":
				cmdDesc = " - Clone repository to local workspace folder"
			case "pull":
				cmdDesc = " - Fast-forward pull changes from remote origin"
			case "fetch":
				cmdDesc = " - Fetch all remote branches and tags"
			case "status":
				cmdDesc = " - Inspect modified, staged, and untracked files"
			case "reset":
				cmdDesc = " - Reset working tree (--hard)"
			}

			lineText := fmt.Sprintf("git %s%s", cmd, cmdDesc)
			if i == ui.SelectedGitCommand {
				modalBody.WriteString(decorator.Selected.Render("> "+lineText) + "\n")
			} else {
				modalBody.WriteString(decorator.Inactive.Render("  "+lineText) + "\n")
			}
		}

		modalBody.WriteString("\n" + gitHintStyle.Render(fmt.Sprintf(
			"[%s] Navigate | [%s] Execute / Proceed | [%s] Back to Projects",
			gitKeyHint.Render("↑/↓/j/k"),
			gitKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			gitKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)))

	case 2:
		targetProj := ui.Config.Projects[ui.SelectedGitProject]
		modalBody.WriteString(fmt.Sprintf("📦 Project: %s\n", gitProjHeader.Render(targetProj.Name)))
		modalBody.WriteString(fmt.Sprintf("📁 Destination: %s\n\n", gitPathHeader.Render(targetProj.Path)))
		modalBody.WriteString(decorator.Section.Render("👉 Step 3: Select Branch to Checkout:") + "\n\n")

		if len(ui.AvailableBranches) == 0 {
			modalBody.WriteString(decorator.Info.Render("  ⏳ Loading branch list from repository...") + "\n")
		} else {
			maxVisible := max(ui.WindowHeight-14, 5)
			startIdx := 0
			if ui.SelectedGitBranch >= maxVisible {
				startIdx = ui.SelectedGitBranch - maxVisible + 1
			}
			endIdx := min(startIdx+maxVisible, len(ui.AvailableBranches))

			if startIdx > 0 {
				modalBody.WriteString(gitHintStyle.Render(fmt.Sprintf("  ↑ ... (%d more above)", startIdx)) + "\n")
			}

			for i := startIdx; i < endIdx; i++ {
				branch := ui.AvailableBranches[i]
				if i == ui.SelectedGitBranch {
					modalBody.WriteString(decorator.Selected.Render("> "+branch) + "\n")
				} else {
					modalBody.WriteString(decorator.Inactive.Render("  "+branch) + "\n")
				}
			}

			if endIdx < len(ui.AvailableBranches) {
				modalBody.WriteString(gitHintStyle.Render(fmt.Sprintf("  ↓ ... (%d more below)", len(ui.AvailableBranches)-endIdx)) + "\n")
			}
		}

		modalBody.WriteString("\n" + gitHintStyle.Render(fmt.Sprintf(
			"[%s] Navigate | [%s] Confirm Checkout | [%s] Back to Operations",
			gitKeyHint.Render("↑/↓/j/k"),
			gitKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			gitKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)),
		)))

	case 3:
		targetProj := ui.Config.Projects[ui.SelectedGitProject]
		modalBody.WriteString(fmt.Sprintf("📦 Project: %s\n", gitProjHeader.Render(targetProj.Name)))
		modalBody.WriteString(decorator.Section.Render("📊 Current Working Tree Status:") + "\n\n")

		lines := strings.Split(ui.GitStatusOutput, "\n")
		for _, line := range lines {
			if len(line) < 3 {
				modalBody.WriteString(decorator.Clean.Render(line) + "\n")
				continue
			}

			code := line[:2]
			file := line[2:]

			switch {
			case strings.Contains(code, "M"):
				modalBody.WriteString(decorator.Modified.Render(" 📝 M ") + file + "\n")
			case strings.Contains(code, "??"):
				modalBody.WriteString(decorator.Untracked.Render(" ❓ ?? ") + file + "\n")
			case code == "A " || code == "M ":
				modalBody.WriteString(decorator.Staged.Render(" 🟩 Staged: ") + file + "\n")
			default:
				modalBody.WriteString("  " + line + "\n")
			}
		}

		modalBody.WriteString("\n" + gitHintStyle.Render(fmt.Sprintf("[%s] Back to Operations", gitKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))))
	}

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(min(ui.WindowWidth-4, 100)).Render(modalBody.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
