package kbd

import (
	"github.com/exr462/go-dark/action"
)

var ShortcutsMap = map[InputField]action.Action{
	FuzzyKey:         action.OpenFuzzy,
	GitOperationsKey: action.OpenGitOperations,
	ProfileKey:       action.OpenConfiguration,
	SessionKey:       action.OpenSession,
	DockerKey:        action.OpenDocker,
	MvnKey:           action.OpenMvn,
	JdkKey:           action.OpenJdk,
	BuildKey:         action.OpenBuild,
	QuitKey:          action.QuitApplication,
	EditShortcutsKey: action.OpenEditShortcuts,
	EditKey:          action.OpenEditFile,
	NewProjectKey:    action.OpenNewProject,
	SubmitKey:        action.Save,
	CancelKey:        action.Escape,
	HelpKey:          action.OpenHelp,
	ToggleKey:        action.Toggle,
	Terminal:         action.OpenTerminal,
}
