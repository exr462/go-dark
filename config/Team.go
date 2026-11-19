package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/examples"
)

// ManifestFileName is the file a team keeps at the root of its manifest repo.
const ManifestFileName = "go-dark.json"

// Team is a subscription to a team manifest living in a git repository.
type Team struct {
	Name   string `json:"name"` // local checkout directory name, derived from GitURL
	GitURL string `json:"git_url"`
	// Manifest is the manifest's own "team" value, recorded at the last
	// successful sync. Project IDs and Sources use it.
	Manifest string `json:"manifest,omitempty"`
	// Builtin marks the default team shipped inside go-dark (no git repo):
	// its manifest is written to the team directory on every refresh.
	Builtin bool `json:"builtin,omitempty"`
}

// BuiltinTeam is the default team a fresh profile starts with: the
// belgiantrain manifest embedded in go-dark (examples/belgiantrain). Once a
// git team providing the same manifest is subscribed, it takes over.
func BuiltinTeam() Team {
	return Team{Name: "belgiantrain", Manifest: "belgiantrain", Builtin: true}
}

// builtinManifests maps a built-in team's directory name to its manifest.
var builtinManifests = map[string][]byte{"belgiantrain": examples.Belgiantrain}

// writeBuiltin materializes a built-in team's manifest in its team
// directory, so everything reading team checkouts (LoadCachedManifests, the
// environment installer) treats it like a synced repo.
func writeBuiltin(t Team) error {
	data, ok := builtinManifests[t.Name]
	if !ok {
		return fmt.Errorf("unknown built-in team %q", t.Name)
	}
	dir, err := TeamDir(t)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ManifestFileName), data, 0o644)
}

// withBuiltin seeds the built-in team into an empty profile (no teams, no
// projects), and drops built-in teams that a subscribed git team replaces.
func withBuiltin(teams []Team, projects []Project) []Team {
	if len(teams) == 0 && len(projects) == 0 {
		return []Team{BuiltinTeam()}
	}
	provided := map[string]bool{}
	for _, t := range teams {
		if !t.Builtin && t.Manifest != "" {
			provided[t.Manifest] = true
		}
	}
	kept := teams[:0:0]
	for _, t := range teams {
		if t.Builtin && provided[t.Manifest] {
			continue
		}
		kept = append(kept, t)
	}
	return kept
}

// Manifest is the shared, team-owned description of a set of projects.
type Manifest struct {
	Team       string            `json:"team"`
	BaseGitURL string            `json:"base_git_url,omitempty"`
	Projects   []ManifestProject `json:"projects"`
	// Environment is the team's local environment (see Environment); nil
	// when the team doesn't declare one.
	Environment *Environment `json:"environment,omitempty"`
}

// ManifestProject is one project as declared by a team. DependsOn entries are
// either a bare name (same team) or "<team>/<name>" (cross-team).
type ManifestProject struct {
	Name       string            `json:"name"`
	GitURL     string            `json:"git_url,omitempty"`
	Type       string            `json:"type"`
	Buildable  bool              `json:"buildable"`
	Deployable bool              `json:"deployable"`
	DependsOn  []string          `json:"depends_on"`
	Commands   map[string]string `json:"commands,omitempty"`
}

// TeamNameFromURL derives a filesystem-safe team directory name from a git
// URL, e.g. "git@bitbucket.org:belgiantrain/go-dark-team.git" -> "go-dark-team".
func TeamNameFromURL(gitURL string) string {
	name := strings.TrimSuffix(strings.TrimRight(gitURL, "/"), ".git")
	if i := strings.LastIndexAny(name, "/:"); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// TeamsDir is where team manifest repositories are checked out.
func TeamsDir() (string, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), "teams"), nil
}

// TeamDir is the local checkout of one team's manifest repository.
func TeamDir(team Team) (string, error) {
	dir, err := TeamsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, team.Name), nil
}

// LoadManifest reads and validates a manifest file, resolving default git URLs.
func LoadManifest(path string) (Manifest, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(bytes, &m); err != nil {
		return Manifest{}, fmt.Errorf("%s: %w", path, err)
	}
	if m.Team == "" {
		return Manifest{}, fmt.Errorf("%s: missing \"team\"", path)
	}
	if strings.Contains(m.Team, "/") || m.Team == LocalSource {
		return Manifest{}, fmt.Errorf("%s: invalid team name %q", path, m.Team)
	}
	for i, p := range m.Projects {
		if p.GitURL == "" && m.BaseGitURL != "" {
			m.Projects[i].GitURL = m.BaseGitURL + p.Name + ".git"
		}
	}
	if m.Environment != nil {
		env := m.Environment.WithDefaults()
		if err := env.Validate(); err != nil {
			return Manifest{}, fmt.Errorf("%s: %w", path, err)
		}
		m.Environment = &env
	}
	return m, nil
}

// LoadCachedManifests reads the manifests of every subscribed team from their
// local checkouts. Broken or missing manifests are reported but do not stop
// the others from loading; the manifest team names of those subscriptions
// are returned in unavailable, so their projects can be kept as they are.
func LoadCachedManifests(teams []Team) (manifests []Manifest, unavailable []string, errs []error) {
	for _, t := range teams {
		m, err := loadTeamManifest(t)
		if err != nil {
			if !os.IsNotExist(err) {
				errs = append(errs, err)
			}
			if t.Manifest != "" {
				unavailable = append(unavailable, t.Manifest)
			}
			continue
		}
		manifests = append(manifests, m)
	}
	return manifests, unavailable, errs
}

func loadTeamManifest(t Team) (Manifest, error) {
	dir, err := TeamDir(t)
	if err != nil {
		return Manifest{}, err
	}
	return LoadManifest(filepath.Join(dir, ManifestFileName))
}

// qualifyDependency turns a manifest dependency into a project ID.
func qualifyDependency(team, dep string) string {
	if strings.Contains(dep, "/") {
		return dep
	}
	return ProjectID(team, dep)
}

// MergeManifests produces the effective project list: local projects plus
// every manifest project, keeping the user's own fields for projects that
// already exist. Projects of the teams listed in keep (subscribed, but whose
// manifest could not be read) are kept unchanged. Other team projects whose
// team is no longer subscribed, or that were dropped from their manifest,
// disappear. A local project with the same
// name and path as a team project is adopted by the team (and dependencies
// on it are rewritten), so subscribing never creates duplicates.
func MergeManifests(projects []Project, manifests []Manifest, keep []string, basePath string) []Project {
	existing := make(map[string]Project, len(projects))
	for _, p := range projects {
		existing[p.ID] = p
	}

	var merged []Project
	adopted := make(map[string]string) // old local ID -> team ID
	for _, m := range manifests {
		for _, mp := range m.Projects {
			id := ProjectID(m.Team, mp.Name)
			deps := make([]string, 0, len(mp.DependsOn))
			for _, d := range mp.DependsOn {
				deps = append(deps, qualifyDependency(m.Team, d))
			}
			p := Project{
				ID:         id,
				Source:     m.Team,
				Name:       mp.Name,
				Path:       ResolvePath(basePath, mp.Name),
				Type:       mp.Type,
				GitURL:     mp.GitURL,
				Buildable:  mp.Buildable,
				Deployable: mp.Deployable,
				DependsOn:  deps,
				Commands:   mp.Commands,
			}

			prev, ok := existing[id]
			if !ok {
				localID := ProjectID(LocalSource, mp.Name)
				if local, found := existing[localID]; found && local.Path == p.Path {
					prev, ok = local, true
					adopted[localID] = id
					delete(existing, localID)
				}
			}
			if ok {
				p.Path = prev.Path
				p.JDKName = prev.JDKName
				p.MavenName = prev.MavenName
				p.Fetched = prev.Fetched
				if prev.DepsOverridden {
					p.DependsOn = prev.DependsOn
					p.DepsOverridden = true
				}
			}
			merged = append(merged, p)
		}
	}

	kept := make(map[string]bool, len(keep))
	for _, k := range keep {
		kept[k] = true
	}
	var result []Project
	for _, p := range projects {
		switch {
		case p.IsLocal():
			if _, stillLocal := existing[p.ID]; stillLocal {
				result = append(result, p)
			}
		case kept[p.Source]:
			result = append(result, p)
		}
	}
	result = append(result, merged...)

	if len(adopted) > 0 {
		for i := range result {
			for j, d := range result[i].DependsOn {
				if to, ok := adopted[d]; ok {
					result[i].DependsOn[j] = to
				}
			}
		}
	}
	return result
}

// UnknownDependencies lists the dependencies of p that match no project.
func UnknownDependencies(p Project, projects []Project) []string {
	var unknown []string
	for _, d := range p.DependsOn {
		if FindProject(projects, d) < 0 {
			unknown = append(unknown, d)
		}
	}
	return unknown
}
