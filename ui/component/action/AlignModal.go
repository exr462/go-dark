package componentaction

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/align"
	"github.com/exr462/go-dark/model"
)

// OpenAlign shows the version alignment screen (rescanning the poms first
// so the edit locations are fresh) and returns to returnTo on exit.
func OpenAlign(ui *model.UI, returnTo model.ApplicationViewState) tea.Cmd {
	ui.ViewState = model.StateAlignModal
	ui.AlignReturnTo = returnTo
	ui.AlignPreview, ui.AlignConfirm, ui.AlignResults = false, false, nil
	if ui.AlignChoices == nil {
		ui.AlignChoices = map[string]model.AlignTarget{}
	}
	return ScanWorkspace(ui)
}

// AlignModal: up/down select, left/right change the target, p previews the
// edits, enter (twice) commits them, esc goes back.
func AlignModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	if ui.AlignRunning {
		return nil
	}
	rows := model.AlignRows(ui.Workspace, ui.ReleaseTags)
	key := msg.String()

	if key == binding(ui, action.Escape) {
		switch {
		case ui.AlignConfirm:
			ui.AlignConfirm = false
		case ui.AlignPreview:
			ui.AlignPreview = false
		default:
			ui.ViewState = ui.AlignReturnTo
		}
		return nil
	}
	if len(rows) == 0 || ui.GraphScanning {
		return nil
	}
	ui.SelectedAlign = min(ui.SelectedAlign, len(rows)-1)
	row := rows[ui.SelectedAlign]

	if ui.AlignConfirm {
		if key == "enter" || key == binding(ui, action.Save) {
			ui.AlignConfirm = false
			return runAlign(ui, rows)
		}
		return nil
	}

	switch key {
	case "up", "k":
		ui.SelectedAlign = max(ui.SelectedAlign-1, 0)
	case "down", "j":
		ui.SelectedAlign = min(ui.SelectedAlign+1, len(rows)-1)
	case "left", "h":
		ui.AlignChoices[row.ProjectID] = cycleTarget(row, ui.AlignChoices[row.ProjectID], -1)
	case "right", "l":
		ui.AlignChoices[row.ProjectID] = cycleTarget(row, ui.AlignChoices[row.ProjectID], 1)
	case "p":
		ui.AlignPreview = !ui.AlignPreview
	case "enter", binding(ui, action.Save):
		if len(model.AlignChanges(rows, ui.AlignChoices)) == 0 {
			ui.StatusMsg = "Nothing to change with the chosen targets"
			return nil
		}
		ui.AlignResults = nil
		ui.AlignConfirm = true
	}
	return nil
}

// cycleTarget moves to the next target, skipping "release" when the repo
// has no release tag.
func cycleTarget(row model.AlignRow, current model.AlignTarget, step int) model.AlignTarget {
	options := []model.AlignTarget{model.AlignToWorkspace}
	if row.ReleaseVersion != "" && row.ReleaseVersion != row.WorkspaceVersion {
		options = append(options, model.AlignToRelease)
	}
	options = append(options, model.AlignKeep)
	i := 0
	for j, o := range options {
		if o == current {
			i = j
		}
	}
	return options[(i+step+len(options))%len(options)]
}

func runAlign(ui *model.UI, rows []model.AlignRow) tea.Cmd {
	changes := model.AlignChanges(rows, ui.AlignChoices)
	dirs := map[string]string{}
	for id, p := range ui.Workspace.Projects {
		dirs[id] = p.Dir
	}
	who := align.Identity{Name: ui.Config.GitUsername, Email: ui.Config.GitEmail}
	ui.AlignRunning = true
	return func() tea.Msg {
		return model.AlignedMsg{Results: align.Apply(changes, dirs, who)}
	}
}

// Aligned stores the results and rescans so the screen shows what's left.
func Aligned(ui *model.UI, msg model.AlignedMsg) tea.Cmd {
	ui.AlignRunning = false
	ui.AlignResults = msg.Results
	ok := 0
	for _, r := range msg.Results {
		if r.OK() {
			ok++
		}
	}
	ui.StatusMsg = fmt.Sprintf("✅ Aligned versions: %d of %d repos committed on %s (not pushed)", ok, len(msg.Results), align.Branch)
	return ScanWorkspace(ui)
}
