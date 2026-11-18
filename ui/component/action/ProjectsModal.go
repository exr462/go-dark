package componentaction

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

// projectFormInputs are the text rows of the add-project form; the row
// after them (index len(projectFormInputs)) is the project type selector.
var projectFormInputs = []kbd.InputField{kbd.ProjectName, kbd.ProjectGitURL, kbd.ProjectPath}

// ProjectsModal lists every workspace project: "a" adds a local project,
// "d" removes the highlighted one (local projects only - team projects are
// managed by their manifest).
func ProjectsModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	if ui.AddingProject {
		return addProjectForm(ui, msg)
	}

	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.ViewState = model.StateConfigDeckModal
	case "up", "k":
		if ui.SelectedManagedProject > 0 {
			ui.SelectedManagedProject--
		}
	case "down", "j":
		if ui.SelectedManagedProject < len(ui.Config.Projects)-1 {
			ui.SelectedManagedProject++
		}
	case "a":
		ui.AddingProject = true
		ui.ProjectFormType = 0
		for _, f := range projectFormInputs {
			ui.Inputs[f].SetValue("")
			ui.Inputs[f].Blur()
		}
		focusProjectFormField(ui, 0)
		return textinput.Blink
	case "d":
		if ui.SelectedManagedProject >= len(ui.Config.Projects) {
			return nil
		}
		p := ui.Config.Projects[ui.SelectedManagedProject]
		if !p.IsLocal() {
			ui.StatusMsg = fmt.Sprintf("⚠️ %s is managed by team %s - unsubscribe the team or change its manifest", p.Name, p.Source)
			return nil
		}
		ui.Config.Projects = append(ui.Config.Projects[:ui.SelectedManagedProject:ui.SelectedManagedProject], ui.Config.Projects[ui.SelectedManagedProject+1:]...)
		ui.SelectedManagedProject = max(min(ui.SelectedManagedProject, len(ui.Config.Projects)-1), 0)
		ui.StatusMsg = fmt.Sprintf("🗑 Removed %s (files on disk are untouched)", p.Name)
		return ApplyProjectChange(ui)
	}
	return nil
}

func addProjectForm(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	typeRow := len(projectFormInputs)
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.AddingProject = false
		focusProjectFormField(ui, -1)
		return nil
	case "tab", "down":
		focusProjectFormField(ui, (ui.ProjectFormField+1)%(typeRow+1))
		return nil
	case "shift+tab", "up":
		focusProjectFormField(ui, (ui.ProjectFormField+typeRow)%(typeRow+1))
		return nil
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		return saveLocalProject(ui)
	}

	if ui.ProjectFormField == typeRow {
		switch msg.String() {
		case "left", "h":
			ui.ProjectFormType = (ui.ProjectFormType + len(config.PresetNames) - 1) % len(config.PresetNames)
		case "right", "l", "space":
			ui.ProjectFormType = (ui.ProjectFormType + 1) % len(config.PresetNames)
		}
		return nil
	}

	field := projectFormInputs[ui.ProjectFormField]
	var cmd tea.Cmd
	ui.Inputs[field], cmd = ui.Inputs[field].Update(msg)
	return cmd
}

// focusProjectFormField moves focus to the given form row (-1 blurs all).
func focusProjectFormField(ui *model.UI, row int) {
	for _, f := range projectFormInputs {
		ui.Inputs[f].Blur()
	}
	ui.ProjectFormField = max(row, 0)
	if row >= 0 && row < len(projectFormInputs) {
		ui.FocusedInput = projectFormInputs[row]
		ui.Inputs[ui.FocusedInput].Focus()
	}
}

func saveLocalProject(ui *model.UI) tea.Cmd {
	name := strings.TrimSpace(ui.Inputs[kbd.ProjectName].Value())
	switch {
	case name == "":
		ui.StatusMsg = "❌ A project name is required"
		return nil
	case strings.Contains(name, "/"):
		ui.StatusMsg = "❌ Project names can't contain \"/\""
		return nil
	}
	id := config.ProjectID(config.LocalSource, name)
	if config.FindProject(ui.Config.Projects, id) >= 0 {
		ui.StatusMsg = fmt.Sprintf("❌ A local project named %s already exists", name)
		return nil
	}

	path := strings.TrimSpace(ui.Inputs[kbd.ProjectPath].Value())
	if path == "" {
		path = name
	}
	ui.Config.Projects = append(ui.Config.Projects, config.Project{
		ID:        id,
		Source:    config.LocalSource,
		Name:      name,
		Type:      config.PresetNames[ui.ProjectFormType],
		GitURL:    strings.TrimSpace(ui.Inputs[kbd.ProjectGitURL].Value()),
		Path:      config.ResolvePath(ui.Config.BasePath, path),
		Buildable: true,
	})
	ui.AddingProject = false
	focusProjectFormField(ui, -1)
	cmd := ApplyProjectChange(ui)
	// Merging may reorder (or adopt) the project, so look it up again.
	ui.SelectedManagedProject = max(config.FindProject(ui.Config.Projects, id), 0)
	ui.StatusMsg = fmt.Sprintf("✅ Added local project %s", name)
	return cmd
}
