package model

import (
	"path/filepath"
	"slices"

	"github.com/exr462/go-dark/deploy"
	"github.com/exr462/go-dark/env"
	"github.com/exr462/go-dark/kube"
)

// LocalTarget is where the ship screen deploys: the team environment when
// a subscribed manifest declares one, otherwise the legacy local_cluster of
// config.json (the bash sandbox's rancher-lcl).
type LocalTarget struct {
	Cluster    kube.Cluster
	Env        string // chart environment, e.g. lcl
	ConfigFrom string // real environment the charts are configured like
	Team       string // "" for the legacy target
	Legacy     bool
}

// DeployTarget resolves the ship screen's target.
func (ui *UI) DeployTarget() LocalTarget {
	if e, ok := ui.CurrentEnv(); ok {
		return LocalTarget{Cluster: kube.Cluster{Context: e.Context(), Namespace: e.Namespace, Container: e.Container()}, Env: e.Name, ConfigFrom: e.ConfigFrom, Team: e.Team}
	}
	c := ui.Config.Cluster()
	return LocalTarget{Cluster: kube.Cluster{Context: c.Context, Namespace: c.Namespace, Container: c.Container}, Env: c.Env, ConfigFrom: c.ConfigFrom, Legacy: true}
}

// DeployEnv is how the charts are configured for the target, given the
// node's architecture.
func (ui *UI) DeployEnv(arch string) deploy.Env {
	if e, ok := ui.CurrentEnv(); ok {
		return deploy.EnvFromLocal(e, arch)
	}
	return deploy.EnvFromConfig(ui.Config.Cluster(), arch)
}

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

// ShipTargets flattens the deploy waves, in screen order.
func (ui *UI) ShipTargets() []deploy.Target {
	var out []deploy.Target
	for _, wave := range ui.DeployWaves {
		out = append(out, wave...)
	}
	return out
}

// ShipEnabled reports whether a service is switched on for the umbrella
// (the switches are remembered in config.json).
func (ui *UI) ShipEnabled(t deploy.Target) bool {
	return !slices.Contains(ui.Config.ShipDisabled, t.Project.ID)
}

// Umbrella is the one-release deployment of the team environment: the
// chart is generated into the install dir's charts/. Nil for the legacy
// target, which keeps one release per service.
func (ui *UI) Umbrella() *deploy.Umbrella {
	e, ok := ui.CurrentEnv()
	if !ok {
		return nil
	}
	return &deploy.Umbrella{Application: e.Application, Dir: filepath.Join(EnvDir(e), env.ChartsDir), Enabled: ui.ShipEnabled}
}
