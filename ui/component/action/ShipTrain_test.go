//go:build !windows

package componentaction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/task"
	"github.com/exr462/go-dark/ui/component"
)

// shipUI is deployTestUI with a ready cluster, a scanned workspace and a
// pipeline command standing in for mvn.
func shipUI(t *testing.T, pipeline string) (*model.UI, string) {
	t.Helper()
	log := fakeCluster(t, true, "https://127.0.0.1:53846")
	ui := deployTestUI(t)
	ui.Config.Projects[0].Buildable = true
	ui.Config.Projects[0].Commands = map[string]string{config.PipelineCommand: pipeline}
	OpenDeploy(ui)
	ClusterChecked(ui, CheckCluster(ui)().(model.DeployPreflightMsg))
	WorkspaceScanned(ui, ScanWorkspace(ui)().(model.WorkspaceScannedMsg))
	return ui, log
}

// drive runs a train command chain to the end, feeding each message back
// like the Update loop does.
func drive(t *testing.T, ui *model.UI, cmd func() any) {
	t.Helper()
	for step := 0; cmd != nil && step < 10; step++ {
		switch msg := cmd().(type) {
		case model.ShipStartMsg:
			next := ShipStart(ui)
			cmd = func() any { return next() }
		case task.PipelineCompleteMsg:
			next := ShipBuilt(ui, msg)
			if next == nil {
				return
			}
			cmd = func() any { return next() }
		case model.DeployFinishedMsg:
			DeployFinished(ui, msg)
			return
		default:
			t.Fatalf("unexpected message %#v", msg)
		}
	}
}

func TestShip_BuildsThenDeploys(t *testing.T) {
	ui, log := shipUI(t, "echo built-order")

	start := DeployModal(ui, testKeyMsg("enter"))
	if start == nil {
		t.Fatalf("enter must start the train, status %q", ui.StatusMsg)
	}
	drive(t, ui, func() any { return start() })

	if ui.ShipStage != model.ShipDone || ui.DeployErr != nil {
		t.Fatalf("expected the train to finish, stage %v err %v status %q", ui.ShipStage, ui.DeployErr, ui.StatusMsg)
	}
	pipeline := ui.Shells.List()[0]
	if pipeline.Tag != "pipeline" || !strings.Contains(pipeline.Text(), "built-order") {
		t.Fatalf("the build stage must run the pipeline as a session first, got %+v", pipeline)
	}
	calls, _ := os.ReadFile(log)
	if !strings.Contains(string(calls), "docker build") || !strings.Contains(string(calls), "helm upgrade") {
		t.Fatalf("images and deploy must follow the build:\n%s", calls)
	}
	if out := ansi.Strip(component.RenderDeployModal(ui)); !strings.Contains(out, "Build ✅") || !strings.Contains(out, "Images & deploy ✅") {
		t.Fatalf("stage line must show both stages done:\n%s", out)
	}
	if ui.ViewState != model.StateDeployModal {
		t.Fatal("the train must not leave the ship screen")
	}
}

func TestShip_BuildFailureStopsBeforeDeploy(t *testing.T) {
	ui, log := shipUI(t, "echo compile-error; exit 1")

	start := DeployModal(ui, testKeyMsg("enter"))
	drive(t, ui, func() any { return start() })

	if ui.ShipStage != model.ShipFailed || !strings.Contains(ui.StatusMsg, "the build failed") {
		t.Fatalf("expected a failed build stage, got %v %q", ui.ShipStage, ui.StatusMsg)
	}
	if calls, _ := os.ReadFile(log); len(calls) > 0 {
		t.Fatalf("nothing may be deployed after a failed build:\n%s", calls)
	}
}

func TestShip_DriftIsPointedOutFirst(t *testing.T) {
	ui, _ := shipUI(t, "true")
	// Pretend the scan found drift: another project references til-order at an old version.
	parent := filepath.Join(t.TempDir(), "svc")
	_ = os.MkdirAll(parent, 0o755)
	_ = os.WriteFile(filepath.Join(ui.Config.Projects[0].Path, "pom.xml"), []byte(`<project><groupId>g</groupId><artifactId>order</artifactId><version>2.0</version></project>`), 0o644)
	_ = os.WriteFile(filepath.Join(parent, "pom.xml"), []byte(`<project><groupId>g</groupId><artifactId>svc</artifactId><version>1</version><dependencies><dependency><groupId>g</groupId><artifactId>order</artifactId><version>1.0</version></dependency></dependencies></project>`), 0o644)
	ui.Config.Projects = append(ui.Config.Projects, config.Project{ID: "til/svc", Name: "svc", Path: parent, Fetched: true})
	WorkspaceScanned(ui, ScanWorkspace(ui)().(model.WorkspaceScannedMsg))

	if DeployModal(ui, testKeyMsg("enter")) != nil || !strings.Contains(ui.StatusMsg, "v to align first") {
		t.Fatalf("the first enter must warn about drift, got %q", ui.StatusMsg)
	}
	if !strings.Contains(ansi.Strip(component.RenderDeployModal(ui)), "Versions ⚠ 1 mixed") {
		t.Fatal("the stage line must show the drift")
	}
	if DeployModal(ui, testKeyMsg("enter")) == nil {
		t.Fatal("the second enter ships anyway")
	}
	DeployModal(ui, testKeyMsg("v"))
	if ui.ViewState != model.StateAlignModal || ui.AlignReturnTo != model.StateDeployModal {
		t.Fatal("v must open the alignment screen and come back here")
	}
}

func TestShip_CancelDuringBuildKillsPipeline(t *testing.T) {
	ui, log := shipUI(t, "echo compiling; sleep 30")

	// Like the Update loop: messages are handled here; only the pipeline
	// command itself runs in the background.
	if _, ok := DeployModal(ui, testKeyMsg("enter"))().(model.ShipStartMsg); !ok {
		t.Fatal("expected the train to start")
	}
	pipeline := ShipStart(ui)
	result := make(chan task.PipelineCompleteMsg, 1)
	go func() { result <- pipeline().(task.PipelineCompleteMsg) }()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !(len(ui.Shells.List()) > 0 && strings.Contains(ui.Shells.List()[0].Text(), "compiling")) {
		time.Sleep(20 * time.Millisecond)
	}

	DeployModal(ui, testKeyMsg("x"))
	select {
	case msg := <-result:
		if ShipBuilt(ui, msg) != nil {
			t.Fatal("a cancelled build must not continue to deploy")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling must end the build stage")
	}
	if ui.ShipStage != model.ShipFailed {
		t.Fatalf("a cancelled build stops the train, stage %v", ui.ShipStage)
	}
	if calls, _ := os.ReadFile(log); len(calls) > 0 {
		t.Fatalf("nothing may be deployed after cancelling:\n%s", calls)
	}
}

func TestShip_StartupPipelineDoesNotAdvanceTrain(t *testing.T) {
	ui, _ := shipUI(t, "true")
	if ShipBuilt(ui, task.PipelineCompleteMsg{Success: true}) != nil || ui.ShipStage != model.ShipIdle {
		t.Fatal("a pipeline run that isn't the train's must not start a deployment")
	}
}

func TestShip_WaitsForARunningPipeline(t *testing.T) {
	ui, log := shipUI(t, "true")
	// e.g. the startup pipeline is still building.
	id, _ := ui.Shells.Start(shellOpts("sleep 30"))
	defer ui.Shells.Close(id)

	if DeployModal(ui, testKeyMsg("enter")) != nil || DeployModal(ui, testKeyMsg("d")) != nil {
		t.Fatal("neither the train nor a deploy may start while a build is running")
	}
	if !strings.Contains(ui.StatusMsg, "still running") {
		t.Fatalf("unexpected status %q", ui.StatusMsg)
	}
	if calls, _ := os.ReadFile(log); len(calls) > 0 {
		t.Fatal("nothing may run")
	}
}

func TestShip_JDKMismatchHint(t *testing.T) {
	ui, _ := shipUI(t, "echo '[ERROR] Fatal error compiling: java.lang.ExceptionInInitializerError: com.sun.tools.javac.code.TypeTag :: UNKNOWN'; exit 1")

	start := DeployModal(ui, testKeyMsg("enter"))
	drive(t, ui, func() any { return start() })

	if !strings.Contains(ui.StatusMsg, "til-order looks like a JDK mismatch - assign the project's JDK with ctrl+j") {
		t.Fatalf("expected a JDK hint, got %q", ui.StatusMsg)
	}
}

func shellOpts(script string) shell.StartOptions {
	return shell.StartOptions{Argv: []string{"sh", "-c", script}, Tag: "pipeline", Dir: os.TempDir()}
}
