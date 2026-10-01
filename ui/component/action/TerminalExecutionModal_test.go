package componentaction

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/terminal"
)

func newTerminalTestUI() *model.UI {
	ui := newDeployTestUI()
	inputs := make([]textinput.Model, kbd.Ceiling)
	for i := range inputs {
		inputs[i] = textinput.New()
	}
	ui.Inputs = inputs
	return ui
}

func TestTerminalExecutionModal_EmptyCommandDoesNothing(t *testing.T) {
	ui := newTerminalTestUI()

	cmd := TerminalExecutionModal(ui, testKeyMsg("enter"))

	if cmd != nil {
		t.Fatal("expected no command for an empty input line")
	}
	if len(ui.TerminalLogs) != 0 {
		t.Fatalf("expected no history to be written for an empty command, got %+v", ui.TerminalLogs)
	}
}

func TestTerminalExecutionModal_SubmittingCommandEchoesPromptAndStartsSession(t *testing.T) {
	ui := newTerminalTestUI()
	ui.Inputs[kbd.Terminal].SetValue("echo hi")

	cmd := TerminalExecutionModal(ui, testKeyMsg("enter"))

	if cmd == nil {
		t.Fatal("expected a session-spawning command to be returned")
	}
	if !ui.IsBuilding {
		t.Fatal("expected IsBuilding to be set true while the command runs")
	}
	if ui.Inputs[kbd.Terminal].Value() != "" {
		t.Fatalf("expected the input line to be cleared after submit, got %q", ui.Inputs[kbd.Terminal].Value())
	}
	if len(ui.TerminalLogs) != 1 {
		t.Fatalf("expected exactly one echoed prompt line, got %+v", ui.TerminalLogs)
	}
	echoed := ui.TerminalLogs[0]
	if !echoed.IsPrompt {
		t.Fatalf("expected the echoed command line to be marked as a prompt, got %+v", echoed)
	}
	if !strings.Contains(echoed.Text, "echo hi") || !strings.Contains(echoed.Text, "parent") {
		t.Fatalf("expected prompt echo to contain project name and command, got %q", echoed.Text)
	}
}

func TestTerminalExecutionModal_HistoryAccumulatesAcrossCommands(t *testing.T) {
	ui := newTerminalTestUI()
	ui.TerminalLogs = []terminal.LogLine{{Text: "previous output"}}

	ui.Inputs[kbd.Terminal].SetValue("ls")
	TerminalExecutionModal(ui, testKeyMsg("enter"))

	if len(ui.TerminalLogs) != 2 {
		t.Fatalf("expected the new prompt to be appended to existing history, not replace it, got %+v", ui.TerminalLogs)
	}
	if ui.TerminalLogs[0].Text != "previous output" {
		t.Fatalf("expected prior history to be preserved, got %+v", ui.TerminalLogs)
	}
}

func TestTerminalExecutionModal_EnterIgnoredWhileCommandIsRunning(t *testing.T) {
	ui := newTerminalTestUI()
	ui.IsBuilding = true
	ui.Inputs[kbd.Terminal].SetValue("ls")

	cmd := TerminalExecutionModal(ui, testKeyMsg("enter"))

	if cmd != nil {
		t.Fatal("expected no new command to be spawned while one is already running")
	}
	if len(ui.TerminalLogs) != 0 {
		t.Fatalf("expected no history change while a command is running, got %+v", ui.TerminalLogs)
	}
}

func TestTerminalExecutionModal_EscapeReturnsToDashboardWhenIdle(t *testing.T) {
	ui := newTerminalTestUI()
	ui.ViewState = model.StateTerminalCockpit

	TerminalExecutionModal(ui, testKeyMsg("esc"))

	if ui.ViewState != model.StateDashboard {
		t.Fatalf("expected esc to return to the dashboard, got %v", ui.ViewState)
	}
}

func TestTerminalExecutionModal_EscapeBlockedWhileCommandIsRunning(t *testing.T) {
	ui := newTerminalTestUI()
	ui.ViewState = model.StateTerminalCockpit
	ui.IsBuilding = true

	TerminalExecutionModal(ui, testKeyMsg("esc"))

	if ui.ViewState != model.StateTerminalCockpit {
		t.Fatalf("expected esc to be ignored while a command is running, got %v", ui.ViewState)
	}
}
