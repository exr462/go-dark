package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/storage"
)

func newTestAppModel(t *testing.T) *appModel {
	t.Helper()
	ta := textarea.New()
	u := &model.UI{
		Editor:   ta,
		Spinner:  spinner.New(),
		Progress: progress.New(),
	}
	return &appModel{ui: u, contentLoader: storage.NewContentLoader()}
}

// TestFileLoadedMsg_ResetsCursorToStartOfDocument guards against the
// "editor doesn't open at the start of the document" bug: bubbles/textarea's
// SetValue leaves the cursor at the END of the inserted text by default.
func TestFileLoadedMsg_ResetsCursorToStartOfDocument(t *testing.T) {
	m := newTestAppModel(t)

	_, _ = m.Update(lsp.FileLoadedMsg{Path: "/tmp/foo.go", Content: "line one\nline two\nline three"})

	if got := m.ui.Editor.Line(); got != 0 {
		t.Fatalf("expected cursor to be on line 0 after loading a file, got line %d", got)
	}
	if got := m.ui.Editor.LineInfo().CharOffset; got != 0 {
		t.Fatalf("expected cursor column 0 after loading a file, got %d", got)
	}
	if m.ui.ActiveFilePath != "/tmp/foo.go" {
		t.Fatalf("expected ActiveFilePath to be tracked, got %q", m.ui.ActiveFilePath)
	}
	if m.ui.EditorDirty {
		t.Fatal("expected a freshly loaded file to not be marked dirty")
	}
}

// TestPreflight_LastStepPopulatesDashboardImmediately guards against the
// "blank screen after SystemCheck" bug: the final precheck step used to run
// the (slow) build pipeline inside a tea.Sequence that blocked the dashboard
// transition. It must now switch to the dashboard and eagerly populate the
// selected project's tree in the very same Update() call.
func TestPreflight_LastStepPopulatesDashboardImmediately(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("failed to seed temp project: %v", err)
	}

	m := newTestAppModel(t)
	m.ui.ViewState = model.StateSystemCheckModal
	m.ui.Config = config.Config{Projects: []config.Project{{Name: "demo", Path: tmpDir, Fetched: true}}}
	m.ui.SelectedProject = 0
	m.ui.Prechecks = []model.Precheck{
		{Name: "step-1", Load: func() tea.Cmd { return nil }},
	}
	m.ui.Index = 0

	_, cmd := m.Update(preflightMsg{Function: func() tea.Cmd { return nil }})

	if m.ui.ViewState != model.StateDashboard {
		t.Fatalf("expected ViewState to switch to Dashboard immediately, got %v", m.ui.ViewState)
	}
	if !m.ui.Done {
		t.Fatal("expected Done to be true once every precheck step has run")
	}
	if cmd == nil {
		t.Fatal("expected a non-nil follow-up command batch")
	}
	if len(m.ui.TreeNodes) == 0 {
		t.Fatal("expected the selected project's tree to be populated immediately, not left blank")
	}
}
