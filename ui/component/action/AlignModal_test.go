package componentaction

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/align"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/component"
)

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func alignTestUI(t *testing.T) *model.UI {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	root := t.TempDir()
	poms := map[string]string{
		"parent": `<project><groupId>g</groupId><artifactId>parent</artifactId><version>4.0.0.ALPHA</version></project>`,
		"svc":    "<project>\n  <parent><groupId>g</groupId><artifactId>parent</artifactId><version>3.4.9</version></parent>\n  <artifactId>svc</artifactId><version>1.0</version>\n</project>\n",
	}
	var projects []config.Project
	for _, name := range []string{"parent", "svc"} {
		dir := filepath.Join(root, name)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(poms[name]), 0o644)
		gitIn(t, dir, "init", "-q", "-b", "main")
		gitIn(t, dir, "add", ".")
		gitIn(t, dir, "-c", "user.name=t", "-c", "user.email=t@e", "commit", "-q", "-m", "init")
		projects = append(projects, config.Project{ID: "local/" + name, Source: config.LocalSource, Name: name, Path: dir, Fetched: true})
	}
	gitIn(t, projects[0].Path, "tag", "parent-3.5.7")
	return &model.UI{
		Config: config.Config{
			ShortCuts: action.DefaultShortcuts, Projects: projects,
			GitUsername: "Dev", GitEmail: "dev@example.com",
		},
		WindowWidth: 160, WindowHeight: 45,
	}
}

func TestAlign_ChooseReleasePreviewConfirmAndCommit(t *testing.T) {
	ui := alignTestUI(t)
	cmd := OpenAlign(ui, model.StateDashboard)
	WorkspaceScanned(ui, cmd().(model.WorkspaceScannedMsg))

	out := ansi.Strip(component.RenderAlignModal(ui))
	if !strings.Contains(out, "workspace 4.0.0.ALPHA") || !strings.Contains(out, "release 3.5.7") || !strings.Contains(out, "3.4.9 ×1") || !strings.Contains(out, "→ 4.0.0.ALPHA") {
		t.Fatalf("expected the drift row with workspace target by default:\n%s", out)
	}

	AlignModal(ui, testKeyMsg("l")) // workspace -> release
	AlignModal(ui, testKeyMsg("p"))
	out = ansi.Strip(component.RenderAlignModal(ui))
	if !strings.Contains(out, "→ 3.5.7") || !strings.Contains(out, "- <parent><groupId>g</groupId><artifactId>parent</artifactId><version>3.4.9") ||
		!strings.Contains(out, "+ <parent><groupId>g</groupId><artifactId>parent</artifactId><version>3.5.7") || !strings.Contains(out, "svc/pom.xml:2") {
		t.Fatalf("expected the release target and a line preview:\n%s", out)
	}

	AlignModal(ui, testKeyMsg("enter"))
	if !ui.AlignConfirm || !strings.Contains(ansi.Strip(component.RenderAlignModal(ui)), "Commit these edits to 1 repo") {
		t.Fatal("the first enter must ask for confirmation")
	}
	run := AlignModal(ui, testKeyMsg("enter"))
	if run == nil || !ui.AlignRunning {
		t.Fatal("the second enter must start committing")
	}
	rescan := Aligned(ui, run().(model.AlignedMsg))
	WorkspaceScanned(ui, rescan().(model.WorkspaceScannedMsg))

	svc := ui.Config.Projects[1].Path
	if gitIn(t, svc, "rev-parse", "--abbrev-ref", "HEAD") != align.Branch || gitIn(t, svc, "log", "-1", "--format=%an") != "Dev" {
		t.Fatal("expected a commit by the configured identity on the alignment branch")
	}
	pom, _ := os.ReadFile(filepath.Join(svc, "pom.xml"))
	if !strings.Contains(string(pom), "<version>3.5.7</version>") {
		t.Fatalf("pom not aligned:\n%s", pom)
	}
	out = ansi.Strip(component.RenderAlignModal(ui))
	if !strings.Contains(out, "✅ svc: committed") || !strings.Contains(ui.StatusMsg, "1 of 1 repos committed") {
		t.Fatalf("expected the run summary:\n%s", out)
	}
	// 3.5.7 is still not the workspace version, so the row remains (now 3.5.7 ×1).
	if !strings.Contains(out, "3.5.7 ×1") {
		t.Fatalf("the rescan must show the new state:\n%s", out)
	}
}

func TestAlign_EscBacksOutStepByStepAndKeepIsANoop(t *testing.T) {
	ui := alignTestUI(t)
	cmd := OpenAlign(ui, model.StateGraphModal)
	WorkspaceScanned(ui, cmd().(model.WorkspaceScannedMsg))

	AlignModal(ui, testKeyMsg("h")) // workspace -> keep (wraps backwards)
	AlignModal(ui, testKeyMsg("enter"))
	if ui.AlignConfirm || !strings.Contains(ui.StatusMsg, "Nothing to change") {
		t.Fatal("keep means nothing to commit")
	}

	AlignModal(ui, testKeyMsg("l")) // keep -> workspace
	AlignModal(ui, testKeyMsg("p"))
	AlignModal(ui, testKeyMsg("enter"))
	AlignModal(ui, testKeyMsg("esc"))
	if ui.AlignConfirm || !ui.AlignPreview || ui.ViewState != model.StateAlignModal {
		t.Fatal("esc first cancels the confirmation")
	}
	AlignModal(ui, testKeyMsg("esc"))
	AlignModal(ui, testKeyMsg("esc"))
	if ui.ViewState != model.StateGraphModal {
		t.Fatal("esc finally returns to where the screen was opened from")
	}
	if gitIn(t, ui.Config.Projects[1].Path, "rev-parse", "--abbrev-ref", "HEAD") != "main" {
		t.Fatal("nothing may be committed without confirming")
	}
}

func TestAlign_DowngradesAreFlagged(t *testing.T) {
	ui := alignTestUI(t)
	// svc already uses a parent newer than the release tag.
	pom := filepath.Join(ui.Config.Projects[1].Path, "pom.xml")
	data, _ := os.ReadFile(pom)
	_ = os.WriteFile(pom, []byte(strings.Replace(string(data), "3.4.9", "3.5.8", 1)), 0o644)
	gitIn(t, ui.Config.Projects[1].Path, "-c", "user.name=t", "-c", "user.email=t@e", "commit", "-qam", "newer parent")
	cmd := OpenAlign(ui, model.StateDashboard)
	WorkspaceScanned(ui, cmd().(model.WorkspaceScannedMsg))

	AlignModal(ui, testKeyMsg("l")) // release 3.5.7 < 3.5.8
	AlignModal(ui, testKeyMsg("p"))
	AlignModal(ui, testKeyMsg("enter"))
	out := ansi.Strip(component.RenderAlignModal(ui))
	for _, want := range []string{"↓ 1 downgrade", "svc/pom.xml:2 ↓ downgrade", "This includes 1 downgrade"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}
