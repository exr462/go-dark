package maven

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/exr462/go-dark/config"
)

// Artifact is one artifact produced by a workspace project.
type Artifact struct {
	GA
	ProjectID string
	Version   string
	POM       string
}

// Edit is a change to the text at Span in File.
type Edit struct {
	File     string
	Span     Span
	Current  string
	Property string // set when the version comes from a <properties> entry
}

// Ref is a reference from one project to another project's artifact.
type Ref struct {
	ProjectID string // the referencing project
	Target    string // the producing project
	POM       string
	GA        GA
	Kind      RefKind
	Version   string // resolved version ("" when inherited from dependencyManagement)
	Edit      *Edit  // nil when the version isn't editable here (absent, ${project.version}, mixed text...)
}

// Project is the pom view of one workspace project.
type Project struct {
	ID      string
	Dir     string
	Version string // version of the repo's root pom
	POMs    []*POM
}

// Workspace indexes every pom of the scanned projects.
type Workspace struct {
	Projects  map[string]*Project
	Artifacts map[GA]Artifact
	Refs      []Ref
}

var skipDirs = map[string]bool{"target": true, "node_modules": true, ".git": true, ".idea": true}

// ScanProject parses every pom.xml in dir (recursively, skipping build output).
func ScanProject(dir string) ([]*POM, []error) {
	var poms []*POM
	var errs []error
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != dir && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == "pom.xml" {
			p, perr := ParsePOM(path)
			if perr != nil {
				errs = append(errs, perr)
				return nil
			}
			poms = append(poms, p)
		}
		return nil
	})
	return poms, errs
}

// Scan indexes the cloned projects of the workspace.
func Scan(projects []config.Project) (*Workspace, []error) {
	w := &Workspace{Projects: map[string]*Project{}, Artifacts: map[GA]Artifact{}}
	var errs []error
	for _, cp := range projects {
		if !cp.Fetched || cp.Path == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(cp.Path, "pom.xml")); err != nil {
			continue
		}
		poms, perrs := ScanProject(cp.Path)
		errs = append(errs, perrs...)
		proj := &Project{ID: cp.ID, Dir: cp.Path, POMs: poms}
		for _, p := range poms {
			if filepath.Dir(p.Path) == filepath.Clean(cp.Path) {
				proj.Version = resolveValue(p.Self.Version, p, poms)
			}
			ga := p.Self.GA
			if prev, dup := w.Artifacts[ga]; dup && prev.ProjectID != cp.ID {
				errs = append(errs, fmt.Errorf("%s is produced by both %s and %s", ga, prev.ProjectID, cp.ID))
				continue
			}
			w.Artifacts[ga] = Artifact{GA: ga, ProjectID: cp.ID, Version: resolveValue(p.Self.Version, p, poms), POM: p.Path}
		}
		w.Projects[cp.ID] = proj
	}

	for _, proj := range w.Projects {
		for _, p := range proj.POMs {
			coords := make([]Dependency, 0, len(p.Dependencies)+1)
			if p.Parent != nil {
				coords = append(coords, Dependency{Coord: *p.Parent, Kind: KindParent})
			}
			coords = append(coords, p.Dependencies...)
			for _, c := range coords {
				art, ok := w.Artifacts[c.GA]
				if !ok || art.ProjectID == proj.ID {
					continue // not a workspace artifact, or inside the same repo
				}
				w.Refs = append(w.Refs, Ref{
					ProjectID: proj.ID,
					Target:    art.ProjectID,
					POM:       p.Path,
					GA:        c.GA,
					Kind:      c.Kind,
					Version:   resolveValue(c.Version, p, proj.POMs),
					Edit:      editFor(c, p, proj.POMs),
				})
			}
		}
	}
	sort.Slice(w.Refs, func(i, j int) bool {
		a, b := w.Refs[i], w.Refs[j]
		if a.ProjectID != b.ProjectID {
			return a.ProjectID < b.ProjectID
		}
		if a.POM != b.POM {
			return a.POM < b.POM
		}
		return a.GA.String() < b.GA.String()
	})
	return w, errs
}

// SuggestedDeps returns the projects whose artifacts the given project
// references, sorted.
func (w *Workspace) SuggestedDeps(projectID string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range w.Refs {
		if r.ProjectID == projectID && !seen[r.Target] {
			seen[r.Target] = true
			out = append(out, r.Target)
		}
	}
	sort.Strings(out)
	return out
}

// RefsTo returns every reference to the given producing project.
func (w *Workspace) RefsTo(target string) []Ref {
	var out []Ref
	for _, r := range w.Refs {
		if r.Target == target {
			out = append(out, r)
		}
	}
	return out
}

// RefsFrom returns every reference made by the given project.
func (w *Workspace) RefsFrom(projectID string) []Ref {
	var out []Ref
	for _, r := range w.Refs {
		if r.ProjectID == projectID {
			out = append(out, r)
		}
	}
	return out
}

// propertyName returns "x" for exactly "${x}".
func propertyName(v string) (string, bool) {
	if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") && strings.Count(v, "${") == 1 {
		return v[2 : len(v)-1], true
	}
	return "", false
}

// lookupProperty finds a property in the pom or its parents within the repo.
func lookupProperty(name string, p *POM, poms []*POM) (*POM, Property, bool) {
	for depth := 0; p != nil && depth < 20; depth++ {
		if prop, ok := p.Properties[name]; ok {
			return p, prop, true
		}
		p = repoParent(p, poms)
	}
	return nil, Property{}, false
}

// repoParent finds p's parent pom among the repo's poms (by coordinates).
func repoParent(p *POM, poms []*POM) *POM {
	if p.Parent == nil {
		return nil
	}
	for _, q := range poms {
		if q.Self.GA == p.Parent.GA {
			return q
		}
	}
	return nil
}

// resolveValue expands a single "${prop}" version (one level of nesting is
// followed too); anything it can't resolve is returned as written.
func resolveValue(v string, p *POM, poms []*POM) string {
	for range 3 {
		name, ok := propertyName(v)
		if !ok {
			return v
		}
		switch name {
		case "project.version", "pom.version":
			v = p.Self.Version
			continue
		case "project.parent.version":
			if p.Parent == nil {
				return v
			}
			v = p.Parent.Version
			continue
		}
		owner, prop, found := lookupProperty(name, p, poms)
		if !found {
			return v
		}
		v, p = prop.Value, owner
	}
	return v
}

// editFor says where the referenced version can be changed: the literal
// <version>, or the <properties> entry it points to. Built-in project
// properties and mixed text (e.g. "${x}-SNAPSHOT") aren't editable.
func editFor(c Dependency, p *POM, poms []*POM) *Edit {
	if c.Version == "" || !c.VersionSpan.Valid() {
		return nil
	}
	name, isProp := propertyName(c.Version)
	if !isProp {
		if strings.Contains(c.Version, "${") {
			return nil
		}
		return &Edit{File: p.Path, Span: c.VersionSpan, Current: c.Version}
	}
	if strings.HasPrefix(name, "project.") || strings.HasPrefix(name, "pom.") {
		return nil
	}
	owner, prop, found := lookupProperty(name, p, poms)
	if !found || !prop.Span.Valid() || strings.Contains(prop.Value, "${") {
		return nil
	}
	return &Edit{File: owner.Path, Span: prop.Span, Current: prop.Value, Property: name}
}
