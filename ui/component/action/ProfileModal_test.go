package componentaction

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

var ctrlS = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}

func profileTestUI(t *testing.T) *model.UI {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return &model.UI{
		Config:            config.Config{ShortCuts: action.DefaultShortcuts},
		Inputs:            initializer.MakeInputs(),
		PreviousViewState: model.StateDashboard,
	}
}

func TestJDKModal_SavingAProfileLeavesTheScreen(t *testing.T) {
	ui := profileTestUI(t)
	ui.ViewState, ui.JDKStep = model.StateJDKConfigModal, model.StepSelectJDKAction
	JDKModal(ui, ctrlS) // "Add New JDK Version Profile"

	ui.Inputs[kbd.JdkPath].SetValue("/opt/jdk-21")
	JDKModal(ui, ctrlS)
	if ui.ViewState != model.StateJDKConfigModal || ui.JDKStep != model.StepAddNewJDKVersion || len(ui.Config.JDKs) != 0 {
		t.Fatal("an incomplete profile must keep the form open and save nothing")
	}

	ui.Inputs[kbd.JdkName].SetValue("jdk-21")
	JDKModal(ui, ctrlS)
	if ui.ViewState != model.StateDashboard || ui.JDKStep != model.StepSelectJDKAction {
		t.Fatalf("saving must return to the previous screen, got view %v step %v", ui.ViewState, ui.JDKStep)
	}
	if len(ui.Config.JDKs) != 1 || !strings.Contains(ui.StatusMsg, "jdk-21") {
		t.Fatalf("the profile must be added and announced, got %+v / %q", ui.Config.JDKs, ui.StatusMsg)
	}
	if saved, _ := config.LoadConfig(); len(saved.JDKs) != 1 || saved.JDKs[0].Path != "/opt/jdk-21" {
		t.Fatalf("the profile must be saved to config.json, got %+v", saved.JDKs)
	}
}

func TestMvnModal_SavingAProfileLeavesTheScreen(t *testing.T) {
	ui := profileTestUI(t)
	ui.ViewState, ui.MavenStep = model.StateMavenConfigModal, model.StepSelectMvnAction
	MvnModal(ui, ctrlS)

	ui.Inputs[kbd.MvnName].SetValue("mvn-3.9")
	ui.Inputs[kbd.MvnPath].SetValue("/opt/maven")
	MvnModal(ui, ctrlS)
	if ui.ViewState != model.StateDashboard || ui.MavenStep != model.StepSelectMvnAction || len(ui.Config.Mavens) != 1 {
		t.Fatalf("saving must add the profile and return, got view %v step %v %+v", ui.ViewState, ui.MavenStep, ui.Config.Mavens)
	}
}
