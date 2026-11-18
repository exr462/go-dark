// Package session starts project commands (builds, the startup pipeline)
// as sessions of the shell.Manager, so they can be watched live, attached
// to in the terminal pane and inspected afterwards.
package session

import (
	"fmt"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/runner"
	"github.com/exr462/go-dark/shell"
)

// Session tags used by the UI to tell builds apart from interactive shells.
const (
	TagBuild    = "build"
	TagPipeline = "pipeline"
)

// StartBuild runs the project's command for commandKey (see
// config.CommandKeys) in a new session tagged tag. Finished sessions with
// the same tag and project are closed first, unless keep reports that
// they're still wanted (e.g. open as a terminal tab), so re-running doesn't
// pile up old logs.
func StartBuild(sessions *shell.Manager, cfg config.Config, proj config.Project, commandKey, tag string, width, height int, keep func(id int) bool) (int, error) {
	cmdStr := config.ResolveCommand(proj, commandKey)
	if cmdStr == "" {
		return 0, fmt.Errorf("no %q command configured for %s", commandKey, proj.Name)
	}
	for _, s := range sessions.List() {
		if s.Tag == tag && s.ProjectID == proj.ID && !s.Running() && (keep == nil || !keep(s.ID)) {
			_ = sessions.Close(s.ID)
		}
	}
	return sessions.Start(shell.StartOptions{
		ProjectID:   proj.ID,
		ProjectName: proj.Name,
		Dir:         proj.Path,
		Argv:        runner.ShellArgv(cmdStr),
		Env:         runner.Env(cfg, proj),
		Title:       fmt.Sprintf("%s · %s", proj.Name, commandKey),
		Tag:         tag,
		Width:       width,
		Height:      height,
	})
}

// LatestBuild returns the most recent session with the given tag for a
// project, or nil.
func LatestBuild(sessions *shell.Manager, projectID, tag string) *shell.Session {
	if sessions == nil {
		return nil
	}
	list := sessions.List()
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Tag == tag && list[i].ProjectID == projectID {
			return list[i]
		}
	}
	return nil
}

// RunningBuild returns the project's running build or pipeline session, or nil.
func RunningBuild(sessions *shell.Manager, projectID string) *shell.Session {
	if sessions == nil {
		return nil
	}
	for _, s := range sessions.List() {
		if !s.Interactive && s.ProjectID == projectID && s.Running() {
			return s
		}
	}
	return nil
}

// RunningBuilds counts the running non-interactive sessions.
func RunningBuilds(sessions *shell.Manager) int {
	if sessions == nil {
		return 0
	}
	n := 0
	for _, s := range sessions.List() {
		if !s.Interactive && s.Running() {
			n++
		}
	}
	return n
}
