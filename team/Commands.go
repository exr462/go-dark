package team

import (
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/config"
)

// SyncedMsg reports the result of syncing the subscribed teams.
type SyncedMsg struct {
	Teams []config.Team
	Errs  []error
}

// SubscribedMsg reports the result of subscribing to a new team.
type SubscribedMsg struct {
	Teams []config.Team
	Err   error
}

// SyncCmd syncs the given teams in the background.
func SyncCmd(teams []config.Team) tea.Cmd {
	if len(teams) == 0 {
		return nil
	}
	snapshot := append([]config.Team(nil), teams...)
	return func() tea.Msg {
		updated, errs := Sync(snapshot)
		return SyncedMsg{Teams: updated, Errs: errs}
	}
}

// SubscribeCmd subscribes to (clones) a team manifest in the background.
func SubscribeCmd(teams []config.Team, gitURL string) tea.Cmd {
	snapshot := append([]config.Team(nil), teams...)
	return func() tea.Msg {
		updated, err := Subscribe(snapshot, gitURL)
		return SubscribedMsg{Teams: updated, Err: err}
	}
}

// ApplySynced folds sync results into the current subscriptions, matching
// by checkout name so teams added or removed meanwhile are left alone.
func ApplySynced(current []config.Team, synced []config.Team) []config.Team {
	byName := make(map[string]config.Team, len(synced))
	for _, t := range synced {
		byName[t.Name] = t
	}
	out := make([]config.Team, len(current))
	for i, t := range current {
		if s, ok := byName[t.Name]; ok && s.Manifest != "" {
			t.Manifest = s.Manifest
		}
		out[i] = t
	}
	return out
}
