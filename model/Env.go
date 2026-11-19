package model

import "github.com/exr462/go-dark/env"

// CurrentEnv is the team environment the installer screen shows.
func (ui *UI) CurrentEnv() (env.Env, bool) {
	if ui.EnvIndex < 0 || ui.EnvIndex >= len(ui.Config.Environments) {
		return env.Env{}, false
	}
	return env.New(ui.Config.Environments[ui.EnvIndex], ui.Config.LocalCluster.Database.Password), true
}

// EnvDir is an environment's install directory ("" when the home directory
// is unknown).
func EnvDir(e env.Env) string {
	dir, err := e.InstallDir()
	if err != nil {
		return ""
	}
	return dir
}

// EnvProjects counts a team's deployable projects and how many are cloned.
func (ui *UI) EnvProjects(team string) (cloned, total int) {
	for _, p := range ui.Config.Projects {
		if p.Source == team && p.Deployable {
			total++
			if p.Fetched {
				cloned++
			}
		}
	}
	return cloned, total
}
