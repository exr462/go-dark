package kbd

import (
	"github.com/exr462/go-dark/action"
)

// ShortcutField ties a shortcut input of the shortcuts screen to its action.
type ShortcutField struct {
	Field  InputField
	Action action.Action
	Label  string
}

// ShortcutFields lists every editable shortcut in screen order.
var ShortcutFields = []ShortcutField{
	{FuzzyKey, action.OpenFuzzy, "Open Fuzzy Finder"},
	{GitOperationsKey, action.OpenGitOperations, "Open Git Operation Management"},
	{ProfileKey, action.OpenConfiguration, "Open Profile"},
	{SessionKey, action.OpenSession, "Open Session Management"},
	{DockerKey, action.OpenDocker, "Open Docker Management"},
	{MvnKey, action.OpenMvn, "Open Maven Management"},
	{JdkKey, action.OpenJdk, "Open Jdk Management"},
	{BuildKey, action.OpenBuild, "Open Build Screen"},
	{QuitKey, action.QuitApplication, "Quit application"},
	{EditKey, action.OpenEditFile, "Edit"},
	{NewProjectKey, action.OpenNewProject, "New Project"},
	{SubmitKey, action.Save, "Submit action"},
	{CancelKey, action.Escape, "Cancel"},
	{HelpKey, action.OpenHelp, "Open Help Menu"},
	{ToggleKey, action.Toggle, "Toggle action"},
	{EditShortcutsKey, action.OpenEditShortcuts, "Edit shortcuts"},
	{DeployKey, action.OpenDeploy, "Open Deploy"},
	{TerminalKey, action.OpenTerminal, "Open Terminal"},
	{TerminalDetachKey, action.TerminalDetach, "Terminal: back to dashboard (shell keeps running)"},
	{TerminalNewTabKey, action.TerminalNewTab, "Terminal: new tab"},
	{TerminalNextTabKey, action.TerminalNextTab, "Terminal: next tab"},
	{TerminalPrevTabKey, action.TerminalPrevTab, "Terminal: previous tab"},
	{TerminalCloseTabKey, action.TerminalCloseTab, "Terminal: close tab"},
}

// ShortcutsMap maps each shortcut input to its action.
var ShortcutsMap = func() map[InputField]action.Action {
	m := make(map[InputField]action.Action, len(ShortcutFields))
	for _, f := range ShortcutFields {
		m[f.Field] = f.Action
	}
	return m
}()
