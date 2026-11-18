package model

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/exr462/go-dark/maven"
)

// AlignTarget is what a drifting project's references should be set to.
type AlignTarget int

const (
	AlignToWorkspace AlignTarget = iota // the version in the project's own pom
	AlignToRelease                      // the newest release tag of its repo
	AlignKeep                           // leave its references alone
)

// AlignRow is one producing project whose version other projects
// reference inconsistently.
type AlignRow struct {
	ProjectID        string
	WorkspaceVersion string
	ReleaseVersion   string         // "" when the repo has no release tag
	Versions         map[string]int // version in use -> number of references
	Editable         []maven.Ref    // references go-dark can rewrite
	Locked           []maven.Ref    // drifting references it can't (no editable location)
}

// Target returns the version a choice resolves to ("" for keep, or when a
// release version is chosen but none exists).
func (r AlignRow) Target(t AlignTarget) string {
	switch t {
	case AlignToWorkspace:
		return r.WorkspaceVersion
	case AlignToRelease:
		return r.ReleaseVersion
	}
	return ""
}

// AlignRows lists the projects other projects reference at more than one
// version, or at a version other than their own, sorted by project ID.
func AlignRows(w *maven.Workspace, releases map[string]string) []AlignRow {
	if w == nil {
		return nil
	}
	var rows []AlignRow
	for id, p := range w.Projects {
		if p.Version == "" {
			continue
		}
		row := AlignRow{ProjectID: id, WorkspaceVersion: p.Version, ReleaseVersion: releases[id], Versions: map[string]int{}}
		drift := false
		for _, r := range w.RefsTo(id) {
			if r.Version == "" {
				continue // managed elsewhere, nothing written here
			}
			row.Versions[r.Version]++
			if r.Version != p.Version {
				drift = true
			}
			if r.Edit != nil {
				row.Editable = append(row.Editable, r)
			} else if r.Version != p.Version {
				row.Locked = append(row.Locked, r)
			}
		}
		if drift || len(row.Versions) > 1 {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ProjectID < rows[j].ProjectID })
	return rows
}

// Downgrades counts the references a target would move to an older version.
func (r AlignRow) Downgrades(t AlignTarget) int {
	target := r.Target(t)
	if target == "" {
		return 0
	}
	n := 0
	for _, ref := range r.Editable {
		if maven.CompareVersions(ref.Edit.Current, target) > 0 {
			n++
		}
	}
	return n
}

// IsDowngrade reports whether a change moves to an older version.
func IsDowngrade(c AlignChange) bool {
	return maven.CompareVersions(c.Current, c.Value) > 0
}

// AlignChange is one pom edit of an alignment plan.
type AlignChange = maven.Change

// AlignChanges turns the chosen targets into pom edits, grouped by the
// project (repo) that has to change.
func AlignChanges(rows []AlignRow, choices map[string]AlignTarget) map[string][]maven.Change {
	out := map[string][]maven.Change{}
	for _, row := range rows {
		target := row.Target(choices[row.ProjectID])
		if target == "" {
			continue
		}
		for _, r := range row.Editable {
			if r.Edit.Current == target {
				continue
			}
			out[r.ProjectID] = append(out[r.ProjectID], maven.Change{Edit: *r.Edit, Value: target})
		}
	}
	return out
}

// AlignResult reports what version alignment did in one repo.
type AlignResult struct {
	ProjectID string
	Dir       string
	Files     []string // edited files (absolute)
	Commit    string   // short hash when committed
	Skipped   string   // why nothing was done, when skipped
	Err       error
}

// OK reports whether the repo was committed.
func (r AlignResult) OK() bool { return r.Commit != "" }

// AlignedMsg delivers the results of an alignment run.
type AlignedMsg struct{ Results []AlignResult }

// DiffLine is one changed line of a preview.
type DiffLine struct {
	File      string
	Line      int
	Old, New  string
	Downgrade bool
}

// PreviewChanges shows each change as the full line before and after.
// Several changes on one line (rare) are shown once per change.
func PreviewChanges(changes []maven.Change) []DiffLine {
	var out []DiffLine
	cache := map[string][]byte{}
	for _, c := range changes {
		data, ok := cache[c.File]
		if !ok {
			var err error
			if data, err = os.ReadFile(c.File); err != nil {
				continue
			}
			cache[c.File] = data
		}
		if c.Span.End > len(data) {
			continue
		}
		start := bytes.LastIndexByte(data[:c.Span.Start], '\n') + 1
		end := c.Span.End + bytes.IndexByte(data[c.Span.End:], '\n')
		if end < c.Span.End {
			end = len(data)
		}
		old := string(data[start:end])
		updated := string(data[start:c.Span.Start]) + c.Value + string(data[c.Span.End:end])
		out = append(out, DiffLine{File: c.File, Line: bytes.Count(data[:start], []byte("\n")) + 1, Old: old, New: updated, Downgrade: IsDowngrade(c)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return dedupeLines(out)
}

func dedupeLines(lines []DiffLine) []DiffLine {
	var out []DiffLine
	for _, l := range lines {
		if n := len(out); n > 0 && out[n-1].File == l.File && out[n-1].Line == l.Line {
			continue
		}
		out = append(out, l)
	}
	return out
}

// RelPath shows a file relative to its project directory when possible.
func RelPath(projectDir, file string) string {
	if rel, err := filepath.Rel(projectDir, file); err == nil {
		return rel
	}
	return file
}

// VersionSummary renders a version histogram like "3.4.9 ×3, 3.5.7 ×9".
func VersionSummary(versions map[string]int) string {
	keys := make([]string, 0, len(versions))
	for v := range versions {
		keys = append(keys, v)
	}
	sort.Slice(keys, func(i, j int) bool { return maven.CompareVersions(keys[i], keys[j]) < 0 })
	s := ""
	for i, v := range keys {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprintf("%s ×%d", v, versions[v])
	}
	return s
}
