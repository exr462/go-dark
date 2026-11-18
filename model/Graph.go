package model

import (
	"slices"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/maven"
)

// GraphRow is one project on the dependency graph screen.
type GraphRow struct {
	Wave    int
	Project config.Project
	Version string // root pom version ("" when not a scanned Maven project)
	Scanned bool   // the project's poms were scanned
	// Suggested are dependencies the poms show but DependsOn lacks (+).
	Suggested []string
	// Unconfirmed are DependsOn entries the poms don't reference (?). Only
	// computed for scanned projects whose dependency is scanned too.
	Unconfirmed []string
	// Drift are references whose version differs from what the referenced
	// project currently is (⚠).
	Drift []maven.Ref
}

// BuildGraph lays the cloned projects out in dependency waves (from the
// manifest / local DependsOn) and annotates each with what the pom scan
// found. w may be nil before the first scan.
func BuildGraph(projects []config.Project, w *maven.Workspace) ([]GraphRow, error) {
	waves, err := DependencyWaves(projects)
	if err != nil {
		return nil, err
	}
	var rows []GraphRow
	for i, wave := range waves {
		for _, p := range wave {
			rows = append(rows, graphRow(i, p, w))
		}
	}
	return rows, nil
}

func graphRow(wave int, p config.Project, w *maven.Workspace) GraphRow {
	row := GraphRow{Wave: wave, Project: p}
	if w == nil {
		return row
	}
	mp, ok := w.Projects[p.ID]
	if !ok {
		return row
	}
	row.Scanned = true
	row.Version = mp.Version

	suggested := w.SuggestedDeps(p.ID)
	for _, s := range suggested {
		if !slices.Contains(p.DependsOn, s) {
			row.Suggested = append(row.Suggested, s)
		}
	}
	for _, d := range p.DependsOn {
		if _, scanned := w.Projects[d]; scanned && !slices.Contains(suggested, d) {
			row.Unconfirmed = append(row.Unconfirmed, d)
		}
	}
	for _, r := range w.RefsFrom(p.ID) {
		target := w.Projects[r.Target]
		if r.Version != "" && target != nil && target.Version != "" && r.Version != target.Version {
			row.Drift = append(row.Drift, r)
		}
	}
	return row
}

// WorkspaceScannedMsg delivers the result of an asynchronous pom scan.
type WorkspaceScannedMsg struct {
	Workspace *maven.Workspace
	Errors    []error
}
