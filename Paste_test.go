package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

func TestPaste_LandsInFocusedConfigurationInput(t *testing.T) {
	ui := &model.UI{ViewState: model.StateGitConfigurationModal, Inputs: initializer.MakeInputs()}
	ui.FocusedInput = kbd.GitEmail
	ui.Inputs[kbd.GitEmail].Focus()
	m := &appModel{ui: ui}

	m.Update(tea.PasteMsg{Content: "dev@example.com"})

	if got := ui.Inputs[kbd.GitEmail].Value(); got != "dev@example.com" {
		t.Fatalf("expected pasted value in focused input, got %q", got)
	}
}

func TestPaste_IgnoredInEditorNormalMode(t *testing.T) {
	ta := textarea.New()
	ta.Focus()
	ui := &model.UI{ViewState: model.StateEditorModal, Editor: ta, EditorMode: model.EditorModeNormal}
	m := &appModel{ui: ui}

	m.Update(tea.PasteMsg{Content: "rm -rf"})

	if ui.Editor.Value() != "" || ui.EditorDirty {
		t.Fatalf("paste must not edit the buffer in Normal mode, got %q", ui.Editor.Value())
	}
}

func TestPaste_InsertsInEditorInsertMode(t *testing.T) {
	ta := textarea.New()
	ta.Focus()
	ui := &model.UI{ViewState: model.StateEditorModal, Editor: ta, EditorMode: model.EditorModeInsert}
	m := &appModel{ui: ui}

	m.Update(tea.PasteMsg{Content: "hello"})

	if ui.Editor.Value() != "hello" || !ui.EditorDirty {
		t.Fatalf("expected pasted text and dirty flag, got %q dirty=%v", ui.Editor.Value(), ui.EditorDirty)
	}
}

func TestMakeInputs_PlaceholdersRenderInFull(t *testing.T) {
	inputs := initializer.MakeInputs()
	for i, in := range inputs {
		if in.Placeholder == "" || in.Value() != "" {
			continue
		}
		if !strings.Contains(ansi.Strip(in.View()), in.Placeholder[1:]) {
			t.Errorf("input %d: placeholder %q truncated, view %q", i, in.Placeholder, ansi.Strip(in.View()))
		}
	}
}
