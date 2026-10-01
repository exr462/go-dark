package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/exr462/go-dark/action"
)

// ensureConfigDir mimics what LoadConfig() does internally so SaveConfig can
// be called directly in a test without a prior LoadConfig() call.
func ensureConfigDir(t *testing.T) {
	t.Helper()
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
}

func TestLoadConfig_FirstRunWhenNoConfigFileExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, isFirstRun := LoadConfig()

	if !isFirstRun {
		t.Fatal("expected isFirstRun to be true when no config file exists yet")
	}
	if cfg.Projects == nil {
		t.Fatal("expected an initialized (possibly empty) Projects slice")
	}
}

func TestLoadConfig_BackfillsNewShortcutsForExistingConfigs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)

	// Simulate an existing user config saved before action.OpenDeploy existed.
	legacy := Config{
		BasePath: "/workspace",
		ShortCuts: []action.Shortcut{
			{Action: action.OpenFuzzy, KeyBinding: action.OpenFuzzyShortcut},
		},
	}
	if err := SaveConfig(legacy); err != nil {
		t.Fatalf("failed to seed legacy config: %v", err)
	}

	cfg, isFirstRun := LoadConfig()
	if isFirstRun {
		t.Fatal("expected isFirstRun to be false once BasePath is set")
	}

	found := false
	for _, s := range cfg.ShortCuts {
		if s.Action == action.OpenDeploy {
			found = true
			if s.KeyBinding != action.OpenDeployShortcut {
				t.Errorf("expected backfilled OpenDeploy binding %q, got %q", action.OpenDeployShortcut, s.KeyBinding)
			}
		}
	}
	if !found {
		t.Fatal("expected a newly introduced action (OpenDeploy) to be auto-backfilled into an existing config's ShortCuts")
	}
}

func TestSaveConfig_ThenLoadConfig_RoundTripsProjects(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)

	original := Config{
		BasePath: "/workspace",
		Projects: []Project{{Name: "custom-project", Path: "/workspace/custom-project"}},
	}
	if err := SaveConfig(original); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	cfg, _ := LoadConfig()

	found := false
	for _, p := range cfg.Projects {
		if p.Name == "custom-project" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected previously saved custom project to survive a save/load round trip")
	}
}

// TestLoadConfig_SelfHealsLegacyNumericShortcutActions is a regression test
// for a real bug: Shortcut.Action used to (de)serialize as a plain JSON
// integer, which only meant what it meant for a given snapshot of the
// action.Action const block. Adding a new Action anywhere but the very end
// shifted every later iota value, so an old persisted shortcut silently
// became a *different* action on the next load - e.g. a legacy "Enter"
// shortcut (bound to "enter") got reinterpreted as "Save", making the Enter
// key save-and-quit the editor instead of inserting a newline. Action now
// (de)serializes by stable name, and any shortcut that still comes back as
// the old bare-integer form must be dropped and backfilled with the correct
// current default instead of trusted.
func TestLoadConfig_SelfHealsLegacyNumericShortcutActions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)

	// This is a byte-for-byte reproduction of the shape of a real corrupted
	// config.json: "Action" persisted as a bare int (11, which used to mean
	// Enter but now means Save), "KeyBinding" of "enter".
	legacyJSON := []byte(`{
		"base_path": "/workspace",
		"short_cuts": [
			{"Action": 11, "KeyBinding": "enter"},
			{"Action": 12, "KeyBinding": "ctrl+w"}
		]
	}`)
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	if err := os.WriteFile(path, legacyJSON, 0o644); err != nil {
		t.Fatalf("failed to seed legacy config file: %v", err)
	}

	cfg, _ := LoadConfig()

	saveBinding := action.GetShortcutKeyBinding(cfg.ShortCuts, action.Save)
	if saveBinding != action.SaveShortcut {
		t.Fatalf("expected the corrupted legacy numeric shortcut to self-heal to the current Save default %q, got %q", action.SaveShortcut, saveBinding)
	}

	escapeBinding := action.GetShortcutKeyBinding(cfg.ShortCuts, action.Escape)
	if escapeBinding != action.EscapeShortcut {
		t.Fatalf("expected the corrupted legacy numeric shortcut to self-heal to the current Escape default %q, got %q", action.EscapeShortcut, escapeBinding)
	}

	// Re-loading from the now-healed, on-disk config should be stable and
	// keep producing the correct binding (i.e. the fix actually got
	// persisted, not just patched in memory for one run).
	cfg2, _ := LoadConfig()
	if got := action.GetShortcutKeyBinding(cfg2.ShortCuts, action.Save); got != action.SaveShortcut {
		t.Fatalf("expected the self-healed config to stay healed after a second load, got %q", got)
	}
}
