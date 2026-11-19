// Package team keeps the local checkouts of subscribed team manifest
// repositories up to date.
package team

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/exr462/go-dark/config"
)

// syncTimeout bounds each clone/pull so an unreachable host can't stall startup.
const syncTimeout = 20 * time.Second

// Sync clones or fast-forwards every team's manifest repository and records
// the manifest's team name. It returns the updated subscriptions plus one
// error per team that failed; a failed team keeps its previous checkout.
func Sync(teams []config.Team) ([]config.Team, []error) {
	updated := make([]config.Team, len(teams))
	var errs []error
	for i, t := range teams {
		updated[i] = t
		if t.Builtin {
			continue // ships inside go-dark, nothing to pull
		}
		if err := syncOne(t); err != nil {
			errs = append(errs, fmt.Errorf("team %s: %w", t.Name, err))
			continue
		}
		dir, err := config.TeamDir(t)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		m, err := config.LoadManifest(filepath.Join(dir, config.ManifestFileName))
		if err != nil {
			errs = append(errs, fmt.Errorf("team %s: %w", t.Name, err))
			continue
		}
		updated[i].Manifest = m.Team
	}
	return updated, errs
}

// Subscribe adds a team by git URL (if not already subscribed) and syncs it.
func Subscribe(teams []config.Team, gitURL string) ([]config.Team, error) {
	gitURL = strings.TrimSpace(gitURL)
	if gitURL == "" {
		return teams, fmt.Errorf("empty git URL")
	}
	for _, t := range teams {
		if t.GitURL == gitURL {
			return teams, nil
		}
	}
	name := config.TeamNameFromURL(gitURL)
	replacesBuiltin := false
	for _, t := range teams {
		if t.Name == name {
			if !t.Builtin {
				return teams, fmt.Errorf("a team checkout named %q already exists", name)
			}
			replacesBuiltin = true
		}
	}
	if replacesBuiltin {
		// The built-in team's directory holds only the written manifest;
		// make room for the clone.
		if dir, err := config.TeamDir(config.Team{Name: name}); err == nil {
			_ = os.RemoveAll(dir)
		}
	}
	synced, errs := Sync([]config.Team{{Name: name, GitURL: gitURL}})
	if len(errs) > 0 {
		return teams, errs[0]
	}
	var kept []config.Team
	for _, t := range teams {
		if !(t.Builtin && t.Name == name) {
			kept = append(kept, t)
		}
	}
	return append(kept, synced[0]), nil
}

// Unsubscribe removes a team subscription and its local checkout.
func Unsubscribe(teams []config.Team, name string) []config.Team {
	var kept []config.Team
	for _, t := range teams {
		if t.Name == name {
			if dir, err := config.TeamDir(t); err == nil {
				_ = os.RemoveAll(dir)
			}
			continue
		}
		kept = append(kept, t)
	}
	return kept
}

func syncOne(t config.Team) error {
	dir, err := config.TeamDir(t)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return runGit(dir, "pull", "--ff-only", "--quiet")
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	return runGit(filepath.Dir(dir), "clone", "--depth", "1", "--quiet", t.GitURL, dir)
}

func runGit(dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	// Never prompt: a password/passphrase prompt would hang behind the TUI.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("git %s timed out after %s", args[0], syncTimeout)
	}
	if err != nil {
		return fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}
