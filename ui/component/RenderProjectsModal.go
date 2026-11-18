package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

// projectsModalChrome is the number of rows the projects modal uses around
// the project list (title, rule, hints, borders, padding).
const projectsModalChrome = 14

// RenderProjectsModal lists every workspace project and the add form.
func RenderProjectsModal(ui *model.UI) string {
	var body strings.Builder
	body.WriteString(decorator.Title.Render("📦 Workspace Projects") + "\n")
	body.WriteString(decorator.Description.Render("Team projects come from their manifest. Local projects are yours to add and remove.") + "\n")
	body.WriteString(strings.Repeat("─", max(ui.WindowWidth-8, 20)) + "\n\n")

	if ui.AddingProject {
		body.WriteString(renderProjectForm(ui))
	} else {
		body.WriteString(renderProjectList(ui))
	}

	if ui.StatusMsg != "" {
		body.WriteString(decorator.Info.Render(ui.StatusMsg) + "\n")
	}
	var hint string
	if ui.AddingProject {
		hint = fmt.Sprintf("[%s] Next field | [%s] Change type | [%s] Save | [%s] Cancel",
			deckKeyHint.Render("Tab"), deckKeyHint.Render("←/→"),
			deckKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save)),
			deckKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))
	} else {
		hint = fmt.Sprintf("[%s] Navigate | [%s] Add local project | [%s] Remove local project | [%s] Back",
			deckKeyHint.Render("↑/↓"), deckKeyHint.Render("a"), deckKeyHint.Render("d"),
			deckKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))
	}
	body.WriteString(deckHintStyle.Render(hint) + "\n")

	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4+decorator.BorderFrame).Render(body.String()),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}

func renderProjectList(ui *model.UI) string {
	if len(ui.Config.Projects) == 0 {
		return decorator.Meta.Render("  (No projects yet - press a to add one, or subscribe to a team manifest)") + "\n\n"
	}

	// Window the list around the selection so long workspaces fit the screen.
	visible := max(ui.WindowHeight-projectsModalChrome, 3)
	start := max(min(ui.SelectedManagedProject-visible/2, len(ui.Config.Projects)-visible), 0)
	end := min(start+visible, len(ui.Config.Projects))

	// Box border + padding + selection marker leave this much room per row.
	rowWidth := max(ui.WindowWidth-14, 20)
	var b strings.Builder
	for i := start; i < end; i++ {
		p := ui.Config.Projects[i]
		status := "remote"
		if p.Fetched {
			status = "cloned"
		}
		row := ansi.Truncate(fmt.Sprintf("%-40s %-7s %-7s %s", p.ID, p.Type, status, p.Path), rowWidth, "…")
		if i == ui.SelectedManagedProject {
			b.WriteString(decorator.Selected.Render("> "+row) + "\n")
		} else {
			b.WriteString(decorator.InactivePlain.Render("  "+row) + "\n")
		}
	}
	if len(ui.Config.Projects) > visible {
		b.WriteString(decorator.Meta.Render(fmt.Sprintf("  %d-%d of %d", start+1, end, len(ui.Config.Projects))) + "\n")
	}
	return b.String() + "\n"
}

func renderProjectForm(ui *model.UI) string {
	var b strings.Builder
	labels := []string{"Name:", "Git clone URL:", "Path:"}
	fields := []kbd.InputField{kbd.ProjectName, kbd.ProjectGitURL, kbd.ProjectPath}
	for i, f := range fields {
		b.WriteString(decorator.Bold.Render(labels[i]) + "\n")
		b.WriteString(ui.Inputs[f].View() + "\n\n")
	}

	b.WriteString(decorator.Bold.Render("Type:") + "\n")
	var types []string
	for i, name := range config.PresetNames {
		switch {
		case i == ui.ProjectFormType && ui.ProjectFormField == len(fields):
			types = append(types, decorator.Selected.Render(name))
		case i == ui.ProjectFormType:
			types = append(types, decorator.SelectedSubtle.Render(name))
		default:
			types = append(types, decorator.Inactive.Render(name))
		}
	}
	b.WriteString("  " + strings.Join(types, " ") + "\n")
	preset := config.Project{Name: "<name>", Type: config.PresetNames[ui.ProjectFormType]}
	b.WriteString(decorator.Meta.Render("  build: "+config.ResolveCommand(preset, config.PipelineCommand)) + "\n\n")
	return b.String()
}
