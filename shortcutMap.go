package main

import (
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

var shortcutsMap = map[model.InputField]config.ShortcutAction{
	model.FuzzyKey:         config.FuzzyKeyBind,
	model.GitOperationsKey: config.GitOperationsKeyBind,
	model.ProfileKey:       config.ProfileKeyBind,
	model.SessionKey:       config.SessionKeyBind,
	model.DockerKey:        config.DockerKeyBind,
	model.MvnKey:           config.MvnKeyBind,
	model.JdkKey:           config.JdkKeyBind,
	model.BuildKey:         config.BuildKeyBind,
	model.QuitKey:          config.QuitKeyBind,
	model.EditShortcutsKey: config.EditShortcutsKeyBind,
	model.EditKey:          config.EditKeyBind,
	model.NewProjectKey:    config.NewProjectKeyBind,
	model.SubmitKey:        config.SubmitKeyBind,
	model.CancelKey:        config.CancelKeyBind,
	model.HelpKey:          config.HelpKeyBind,
	model.ToggleKey:        config.ToggleKeyBind,
}
