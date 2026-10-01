package action

import (
	"encoding/json"
	"fmt"
)

type Action int

const (
	OpenFuzzy Action = iota
	OpenGitOperations
	OpenConfiguration
	OpenSession
	OpenDocker
	OpenMvn
	OpenJdk
	OpenBuild
	QuitApplication
	OpenEditFile
	OpenNewProject
	Save
	Escape
	OpenHelp
	Toggle
	OpenEditShortcuts
	Enter
	OpenTerminal
	OpenDeploy

	// invalidAction is never assigned to a real shortcut - it's what
	// Action.UnmarshalJSON produces for anything it can't confidently map
	// back to a known action (see the comment there for why).
	invalidAction Action = -1
)

// actionNames is the stable, append-only name for every Action, used for
// JSON (de)serialization instead of the raw iota value. Previously
// Shortcut.Action round-tripped through JSON as a plain integer, so adding a
// new Action anywhere but the very end of the const block silently remapped
// every persisted shortcut in every user's config.json to a *different*
// action (e.g. an old "Enter" shortcut re-read itself as "Save" once the
// enum grew, making Enter quietly save-and-quit the editor). Names are
// immune to that: adding/reordering consts no longer affects already-saved
// configs.
var actionNames = map[Action]string{
	OpenFuzzy:         "OpenFuzzy",
	OpenGitOperations: "OpenGitOperations",
	OpenConfiguration: "OpenConfiguration",
	OpenSession:       "OpenSession",
	OpenDocker:        "OpenDocker",
	OpenMvn:           "OpenMvn",
	OpenJdk:           "OpenJdk",
	OpenBuild:         "OpenBuild",
	QuitApplication:   "QuitApplication",
	OpenEditFile:      "OpenEditFile",
	OpenNewProject:    "OpenNewProject",
	Save:              "Save",
	Escape:            "Escape",
	OpenHelp:          "OpenHelp",
	Toggle:            "Toggle",
	OpenEditShortcuts: "OpenEditShortcuts",
	Enter:             "Enter",
	OpenTerminal:      "OpenTerminal",
	OpenDeploy:        "OpenDeploy",
}

var namesToAction = func() map[string]Action {
	m := make(map[string]Action, len(actionNames))
	for action, name := range actionNames {
		m[name] = action
	}
	return m
}()

// IsValid reports whether a is one of the currently-known actions. Shortcuts
// decoded from JSON land on invalidAction (see UnmarshalJSON) when they came
// from an unknown/legacy source, so callers that reload persisted shortcuts
// should drop anything that fails this check rather than keep it around.
func (a Action) IsValid() bool {
	_, ok := actionNames[a]
	return ok
}

// MarshalJSON writes the Action as its stable name (e.g. "Save") instead of
// the underlying iota int, so config.json survives future reordering of the
// const block above.
func (a Action) MarshalJSON() ([]byte, error) {
	name, ok := actionNames[a]
	if !ok {
		return nil, fmt.Errorf("action: no stable name registered for Action(%d)", int(a))
	}
	return json.Marshal(name)
}

// UnmarshalJSON accepts the current string form ("Save") and gracefully
// degrades anything else (unknown names, or the legacy plain-int form used
// before this fix) to invalidAction rather than erroring out and aborting
// the whole config load. config.LoadConfig already backfills any shortcut
// whose Action doesn't match a currently-known default, so an invalidAction
// shortcut is simply dropped and replaced with the correct current default -
// self-healing old/corrupt config files instead of silently mis-binding keys.
func (a *Action) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		if action, ok := namesToAction[name]; ok {
			*a = action
			return nil
		}
		*a = invalidAction
		return nil
	}

	// Legacy numeric format (or anything else unexpected) - its meaning can
	// no longer be trusted since the enum it was captured against may have
	// been reordered since. See invalidAction's doc comment.
	*a = invalidAction
	return nil
}

const (
	OpenFuzzyShortcut         = "ctrl+f"
	OpenGitOperationsShortcut = "ctrl+g"
	OpenConfigurationShortcut = "ctrl+y"
	OpenSessionShortcut       = "ctrl+o"
	OpenDockerShortcut        = "ctrl+d"
	OpenMvnShortcut           = "ctrl+u"
	OpenJdkShortcut           = "ctrl+j"
	OpenBuildShortcut         = "ctrl+b"
	QuitShortcut              = "ctrl+q"
	OpenEditFileShortcut      = "ctrl+e"
	OpenNewProjectShortcut    = "ctrl+n"
	SaveShortcut              = "ctrl+s"
	EnterShortcut             = "enter"
	EscapeShortcut            = "esc"
	OpenHelpShortcut          = "ctrl+h"
	ToggleShortcut            = "ctrl+tab"
	OpenEditShortcutsShortcut = "ctrl+k"
	TerminalShortcut          = "ctrl+t"
	OpenDeployShortcut        = "ctrl+p"
)

type Shortcut struct {
	Action     Action
	KeyBinding string
}

var DefaultShortcuts = []Shortcut{
	{
		Action:     OpenFuzzy,
		KeyBinding: OpenFuzzyShortcut,
	},
	{
		Action:     OpenGitOperations,
		KeyBinding: OpenGitOperationsShortcut,
	},
	{
		Action:     OpenConfiguration,
		KeyBinding: OpenConfigurationShortcut,
	},
	{
		Action:     OpenSession,
		KeyBinding: OpenSessionShortcut,
	},
	{
		Action:     OpenDocker,
		KeyBinding: OpenDockerShortcut,
	},
	{
		Action:     OpenBuild,
		KeyBinding: OpenBuildShortcut,
	},
	{
		Action:     OpenJdk,
		KeyBinding: OpenJdkShortcut,
	},
	{
		Action:     QuitApplication,
		KeyBinding: QuitShortcut,
	},
	{
		Action:     OpenMvn,
		KeyBinding: OpenMvnShortcut,
	},
	{
		Action:     OpenEditFile,
		KeyBinding: OpenEditFileShortcut,
	},
	{
		Action:     OpenNewProject,
		KeyBinding: OpenNewProjectShortcut,
	},
	{
		Action:     Save,
		KeyBinding: SaveShortcut,
	},
	{
		Action:     Escape,
		KeyBinding: EscapeShortcut,
	},
	{
		Action:     OpenHelp,
		KeyBinding: OpenHelpShortcut,
	},
	{
		Action:     Toggle,
		KeyBinding: ToggleShortcut,
	},
	{
		Action:     OpenEditShortcuts,
		KeyBinding: OpenEditShortcutsShortcut,
	},
	{
		Action:     Enter,
		KeyBinding: EnterShortcut,
	},
	{
		Action:     OpenTerminal,
		KeyBinding: TerminalShortcut,
	},
	{
		Action:     OpenDeploy,
		KeyBinding: OpenDeployShortcut,
	},
}

func GetShortcutKeyBinding(shortcuts []Shortcut, action Action) string {
	for _, shortcut := range shortcuts {
		if shortcut.Action == action {
			return shortcut.KeyBinding
		}
	}

	return ""
}
