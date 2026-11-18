package config

// LocalSource marks a project the user added themselves, as opposed to one
// that comes from a subscribed team manifest.
const LocalSource = "local"

// Project is one entry of the effective workspace: either a personal (local)
// project or one merged in from a team manifest. Team-provided fields are
// refreshed from the manifest on every load; Path, JDKName, MavenName and
// (when DepsOverridden) DependsOn are the user's own and survive refreshes.
type Project struct {
	ID         string            `json:"id"`     // "<team>/<name>" or "local/<name>"
	Source     string            `json:"source"` // team name, or LocalSource
	Name       string            `json:"name"`
	Path       string            `json:"path"`
	Type       string            `json:"type"` // preset name, see presets.go
	GitURL     string            `json:"git_url"`
	JDKName    string            `json:"jdk_name"`
	MavenName  string            `json:"maven_name"`
	Buildable  bool              `json:"buildable"`
	Deployable bool              `json:"deployable"`
	DependsOn  []string          `json:"depends_on"` // project IDs
	Commands   map[string]string `json:"commands,omitempty"`
	Fetched    bool              `json:"fetched"`

	// DepsOverridden records that the user edited DependsOn of a team
	// project locally, so the manifest's list no longer replaces it.
	DepsOverridden bool `json:"deps_overridden,omitempty"`
}

// ProjectID builds the stable identifier used for lookups and dependencies.
func ProjectID(source, name string) string {
	return source + "/" + name
}

// IsLocal reports whether the user owns this project (and may remove it).
func (p Project) IsLocal() bool {
	return p.Source == LocalSource
}

// FindProject returns the index of the project with the given ID, or -1.
func FindProject(projects []Project, id string) int {
	for i := range projects {
		if projects[i].ID == id {
			return i
		}
	}
	return -1
}
