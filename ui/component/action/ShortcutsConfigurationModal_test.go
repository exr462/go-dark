package componentaction

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

func TestShortcutsScreen_EveryFieldReachableAndTerminalKeysEditable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ui := &model.UI{
		Config: config.Config{ShortCuts: append([]action.Shortcut(nil), action.DefaultShortcuts...)},
		Inputs: initializer.MakeInputs(),
	}
	OpenShortcutsConfiguration(ui, model.StateDashboard)

	seen := map[kbd.InputField]bool{ui.FocusedInput: true}
	for range kbd.ShortcutFields {
		ShortcutsConfigurationModal(ui, tea.KeyPressMsg{Code: tea.KeyTab})
		seen[ui.FocusedInput] = true
	}
	if len(seen) != len(kbd.ShortcutFields) {
		t.Fatalf("tab reached %d of %d fields", len(seen), len(kbd.ShortcutFields))
	}
	if ui.FocusedInput != kbd.TerminalCloseTabKey {
		t.Fatal("tab must stop at the last field")
	}
	if got := ui.Inputs[kbd.TerminalDetachKey].Value(); got != action.TerminalDetachShortcut {
		t.Fatalf("fields must be filled from the config, detach shows %q", got)
	}

	ui.Inputs[kbd.TerminalDetachKey].SetValue("ctrl+x")
	ShortcutsConfigurationModal(ui, testKeyMsg("ctrl+s"))

	if action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.TerminalDetach) != "ctrl+x" {
		t.Fatal("the new detach binding must be saved")
	}
	if action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenFuzzy) != action.OpenFuzzyShortcut {
		t.Fatal("other bindings must be untouched")
	}
	if ui.ViewState != model.StateDashboard {
		t.Fatal("save must return to where the screen was opened from")
	}
}

func TestShortcutFields_CoverEveryConfigurableDefault(t *testing.T) {
	fields := map[kbd.InputField]bool{}
	actions := map[action.Action]bool{}
	for _, f := range kbd.ShortcutFields {
		if fields[f.Field] || actions[f.Action] {
			t.Fatalf("duplicate shortcut field %+v", f)
		}
		fields[f.Field], actions[f.Action] = true, true
	}
	for _, d := range action.DefaultShortcuts {
		if d.Action != action.Enter && !actions[d.Action] {
			t.Errorf("default shortcut %v has no field on the shortcuts screen", d.Action)
		}
	}
}
