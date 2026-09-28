package action

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
)

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
	SaveShortcut              = "enter"
	EscapeShortcut            = "esc"
	OpenHelpShortcut          = "ctrl+h"
	ToggleShortcut            = "ctrl+t"
	OpenEditShortcutsShortcut = "ctrl+k"
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
}

func GetShortcutKeyBinding(shortcuts []Shortcut, action Action) string {
	for _, shortcut := range shortcuts {
		if shortcut.Action == action {
			return shortcut.KeyBinding
		}
	}

	return ""
}
