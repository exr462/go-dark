package config

type ShortcutAction int

const (
	FuzzyKeyBind ShortcutAction = iota
	GitOperationsKeyBind
	ProfileKeyBind
	SessionKeyBind
	DockerKeyBind
	MvnKeyBind
	JdkKeyBind
	BuildKeyBind
	QuitKeyBind
	EditKeyBind
	NewProjectKeyBind
	SubmitKeyBind
	CancelKeyBind
	HelpKeyBind
	ToggleKeyBind
	EditShortcutsKeyBind
)

type Shortcut struct {
	Action     ShortcutAction
	KeyBinding string
}

var DefaultShortcuts = []Shortcut{
	{
		Action:     FuzzyKeyBind,
		KeyBinding: "ctrl+f",
	},
	{
		Action:     GitOperationsKeyBind,
		KeyBinding: "ctrl+g",
	},
	{
		Action:     ProfileKeyBind,
		KeyBinding: "ctrl+y",
	},
	{
		Action:     SessionKeyBind,
		KeyBinding: "ctrl+s",
	},
	{
		Action:     DockerKeyBind,
		KeyBinding: "ctrl+d",
	},
	{
		Action:     BuildKeyBind,
		KeyBinding: "ctrl+b",
	},
	{
		Action:     JdkKeyBind,
		KeyBinding: "ctrl+j",
	},
	{
		Action:     QuitKeyBind,
		KeyBinding: "q",
	},
	{
		Action:     MvnKeyBind,
		KeyBinding: "ctrl+u",
	},
	{
		Action:     EditKeyBind,
		KeyBinding: "ctrl+e",
	},
	{
		Action:     NewProjectKeyBind,
		KeyBinding: "ctrl+n",
	},
	{
		Action:     SubmitKeyBind,
		KeyBinding: "enter",
	},
	{
		Action:     CancelKeyBind,
		KeyBinding: "esc",
	},
	{
		Action:     HelpKeyBind,
		KeyBinding: "?",
	},
	{
		Action:     ToggleKeyBind,
		KeyBinding: "ctrl+t",
	},
	{
		Action:     EditShortcutsKeyBind,
		KeyBinding: "ctrl+k",
	},
}

func GetShortcutKeyBinding(shortcuts []Shortcut, action ShortcutAction) string {
	for _, shortcut := range shortcuts {
		if shortcut.Action == action {
			return shortcut.KeyBinding
		}
	}

	return ""
}
