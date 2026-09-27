package main

import (
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

var shortcutsMap = map[model.InputField]action.Action{
	model.FuzzyKey:         action.OpenFuzzy,
	model.GitOperationsKey: action.OpenGitOperations,
	model.ProfileKey:       action.OpenConfiguration,
	model.SessionKey:       action.OpenSession,
	model.DockerKey:        action.OpenDocker,
	model.MvnKey:           action.OpenMvn,
	model.JdkKey:           action.OpenJdk,
	model.BuildKey:         action.OpenBuild,
	model.QuitKey:          action.QuitApplication,
	model.EditShortcutsKey: action.OpenEditShortcuts,
	model.EditKey:          action.OpenEditFile,
	model.NewProjectKey:    action.OpenNewProject,
	model.SubmitKey:        action.Save,
	model.CancelKey:        action.Escape,
	model.HelpKey:          action.OpenHelp,
	model.ToggleKey:        action.Toggle,
}
