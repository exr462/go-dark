package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/exr462/go-dark/examples"
)

func writeManifest(t *testing.T, team Team, body string) {
	t.Helper()
	dir, err := TeamDir(team)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadManifest_DefaultsGitURLFromBase(t *testing.T) {
	path := filepath.Join(t.TempDir(), ManifestFileName)
	_ = os.WriteFile(path, []byte(`{"team":"rail","base_git_url":"git@host:rail/","projects":[
		{"name":"api","type":"maven"},
		{"name":"web","type":"npm","git_url":"https://other/web.git"}]}`), 0o644)

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Projects[0].GitURL != "git@host:rail/api.git" {
		t.Errorf("expected default git URL, got %q", m.Projects[0].GitURL)
	}
	if m.Projects[1].GitURL != "https://other/web.git" {
		t.Errorf("explicit git URL must win, got %q", m.Projects[1].GitURL)
	}
}

func TestLoadManifest_RejectsMissingOrReservedTeam(t *testing.T) {
	for _, body := range []string{`{"projects":[]}`, `{"team":"local"}`, `{"team":"a/b"}`, `{not json`} {
		path := filepath.Join(t.TempDir(), ManifestFileName)
		_ = os.WriteFile(path, []byte(body), 0o644)
		if _, err := LoadManifest(path); err == nil {
			t.Errorf("expected an error for manifest %s", body)
		}
	}
}

func TestTeamNameFromURL(t *testing.T) {
	cases := map[string]string{
		"git@bitbucket.org:belgiantrain/go-dark-team.git": "go-dark-team",
		"https://github.com/acme/manifests.git":           "manifests",
		"https://github.com/acme/manifests/":              "manifests",
	}
	for in, want := range cases {
		if got := TeamNameFromURL(in); got != want {
			t.Errorf("TeamNameFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMergeManifests_QualifiesDepsAndKeepsUserFields(t *testing.T) {
	manifests := []Manifest{
		{Team: "rail", Projects: []ManifestProject{
			{Name: "parent", Type: "maven", Buildable: true},
			{Name: "api", Type: "maven", Buildable: true, DependsOn: []string{"parent", "infra/base"}},
		}},
	}
	existing := []Project{
		{ID: "rail/api", Source: "rail", Name: "api", Path: "/custom/api", JDKName: "jdk-17", DependsOn: []string{"stale"}},
		{ID: "local/tool", Source: LocalSource, Name: "tool", Path: "/ws/tool"},
		{ID: "gone/svc", Source: "gone", Name: "svc"},
	}

	merged := MergeManifests(existing, manifests, nil, "/ws")

	ids := make([]string, len(merged))
	for i, p := range merged {
		ids[i] = p.ID
	}
	if !slices.Equal(ids, []string{"local/tool", "rail/parent", "rail/api"}) {
		t.Fatalf("unexpected merged projects %v", ids)
	}
	api := merged[FindProject(merged, "rail/api")]
	if api.Path != "/custom/api" || api.JDKName != "jdk-17" {
		t.Errorf("user fields must survive the merge, got path=%q jdk=%q", api.Path, api.JDKName)
	}
	if !slices.Equal(api.DependsOn, []string{"rail/parent", "infra/base"}) {
		t.Errorf("manifest deps must be qualified and replace stale ones, got %v", api.DependsOn)
	}
	if parent := merged[FindProject(merged, "rail/parent")]; parent.Path != "/ws/parent" {
		t.Errorf("new team projects default to <base>/<name>, got %q", parent.Path)
	}
}

func TestMergeManifests_KeepsLocalDependencyOverride(t *testing.T) {
	manifests := []Manifest{{Team: "rail", Projects: []ManifestProject{
		{Name: "api", DependsOn: []string{"parent"}},
	}}}
	existing := []Project{{ID: "rail/api", Source: "rail", Name: "api", DependsOn: []string{"local/lib"}, DepsOverridden: true}}

	merged := MergeManifests(existing, manifests, nil, "/ws")

	if !slices.Equal(merged[0].DependsOn, []string{"local/lib"}) || !merged[0].DepsOverridden {
		t.Fatalf("overridden deps must win over the manifest, got %+v", merged[0])
	}
}

func TestMergeManifests_AdoptsMatchingLocalProject(t *testing.T) {
	manifests := []Manifest{{Team: "rail", Projects: []ManifestProject{{Name: "parent"}}}}
	existing := []Project{
		{ID: "local/parent", Source: LocalSource, Name: "parent", Path: "/ws/parent", MavenName: "mvn-3.9", Fetched: true},
		{ID: "local/app", Source: LocalSource, Name: "app", Path: "/ws/app", DependsOn: []string{"local/parent"}},
	}

	merged := MergeManifests(existing, manifests, nil, "/ws")

	if FindProject(merged, "local/parent") >= 0 {
		t.Fatal("the local duplicate must be adopted by the team project")
	}
	parent := merged[FindProject(merged, "rail/parent")]
	if parent.MavenName != "mvn-3.9" || !parent.Fetched {
		t.Errorf("adopted project must keep the user's fields, got %+v", parent)
	}
	app := merged[FindProject(merged, "local/app")]
	if !slices.Equal(app.DependsOn, []string{"rail/parent"}) {
		t.Errorf("dependencies on the adopted project must be rewritten, got %v", app.DependsOn)
	}
}

func TestMergeManifests_KeepsProjectsOfUnavailableManifest(t *testing.T) {
	existing := []Project{{ID: "rail/api", Source: "rail", Name: "api", JDKName: "jdk-21"}}

	merged := MergeManifests(existing, nil, []string{"rail"}, "/ws")

	if len(merged) != 1 || merged[0].JDKName != "jdk-21" {
		t.Fatalf("a subscribed team whose manifest can't be read must keep its projects, got %+v", merged)
	}
}

func TestUnknownDependencies(t *testing.T) {
	projects := []Project{{ID: "rail/a", DependsOn: []string{"rail/b", "infra/missing"}}, {ID: "rail/b"}}
	if got := UnknownDependencies(projects[0], projects); !slices.Equal(got, []string{"infra/missing"}) {
		t.Fatalf("expected only the missing dependency, got %v", got)
	}
}

func TestLoadConfig_MergesCachedTeamManifests(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)
	rail := Team{Name: "rail-team", GitURL: "git@host:rail/rail-team.git", Manifest: "rail"}
	writeManifest(t, rail, `{"team":"rail","projects":[{"name":"api","type":"maven","buildable":true}]}`)
	_ = SaveConfig(Config{BasePath: "/ws", Teams: []Team{rail}, Projects: []Project{
		{ID: "local/tool", Source: LocalSource, Name: "tool", Type: "go"},
	}})

	cfg, _ := LoadConfig()

	if FindProject(cfg.Projects, "rail/api") < 0 || FindProject(cfg.Projects, "local/tool") < 0 {
		t.Fatalf("expected team and local projects, got %+v", cfg.Projects)
	}
}

func TestLoadConfig_RemovedLocalProjectStaysRemoved(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)
	_ = SaveConfig(Config{BasePath: "/ws", Projects: []Project{
		{ID: "local/keep", Source: LocalSource, Name: "keep", Type: "go"},
	}})

	cfg, _ := LoadConfig()

	if len(cfg.Projects) != 1 {
		t.Fatalf("no hardcoded projects may be re-added, got %+v", cfg.Projects)
	}
}

func TestLoadConfig_MigratesLegacyProjects(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)
	path, _ := GetConfigPath()
	// Shape of a config written before project IDs existed.
	_ = os.WriteFile(path, []byte(`{"base_path":"/ws","projects":[
		{"name":"ypto-maven-parent","path":"/ws/ypto-maven-parent","type":"java","jdk_name":"jdk-17"},
		{"name":"til-order","path":"/ws/til-order","type":"java","deployable":true},
		{"name":"Ypto Framework","path":"/ws/ypto","type":"java"}]}`), 0o644)

	cfg, _ := LoadConfig()

	order := cfg.Projects[FindProject(cfg.Projects, "local/til-order")]
	if order.Type != "maven" || !order.Buildable || !order.Deployable {
		t.Errorf("legacy registry metadata must be restored, got %+v", order)
	}
	if !slices.Equal(order.DependsOn, []string{"local/contract-first-api-parent"}) {
		t.Errorf("legacy deps must become local IDs, got %v", order.DependsOn)
	}
	if parent := cfg.Projects[FindProject(cfg.Projects, "local/ypto-maven-parent")]; parent.JDKName != "jdk-17" {
		t.Errorf("user fields must survive migration, got %+v", parent)
	}
	if custom := FindProject(cfg.Projects, "local/Ypto Framework"); custom < 0 || !cfg.Projects[custom].Buildable {
		t.Errorf("unknown legacy projects become buildable local projects, got %+v", cfg.Projects)
	}

	// The migration is persisted, so a second load sees the same result.
	again, _ := LoadConfig()
	if len(again.Projects) != 3 || FindProject(again.Projects, "local/til-order") < 0 {
		t.Fatalf("migration must persist, got %+v", again.Projects)
	}
}

func TestLoadConfig_DependencyEditSurvivesRestart(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)
	rail := Team{Name: "rail-team", Manifest: "rail"}
	writeManifest(t, rail, `{"team":"rail","projects":[{"name":"api","depends_on":["parent"]},{"name":"parent"}]}`)
	_ = SaveConfig(Config{BasePath: "/ws", Teams: []Team{rail}})

	cfg, _ := LoadConfig()
	i := FindProject(cfg.Projects, "rail/api")
	cfg.Projects[i].DependsOn = nil
	cfg.Projects[i].DepsOverridden = true
	_ = SaveConfig(cfg)

	reloaded, _ := LoadConfig()
	if api := reloaded.Projects[FindProject(reloaded.Projects, "rail/api")]; len(api.DependsOn) != 0 {
		t.Fatalf("dependency edit must survive a restart, got %v", api.DependsOn)
	}
}

func TestRefreshProjects_FreshProfileGetsTheBuiltinTeam(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)
	builtin, err := LoadManifest(writeTemp(t, examples.Belgiantrain))
	if err != nil {
		t.Fatal(err)
	}

	// What the first-run form does once the workspace path is entered.
	cfg := Config{BasePath: "/ws", Projects: []Project{}}
	cfg.RefreshProjects()

	if len(cfg.Teams) != 1 || !cfg.Teams[0].Builtin || cfg.Teams[0].Manifest != builtin.Team {
		t.Fatalf("a fresh profile must start with the built-in team, got %+v", cfg.Teams)
	}
	if len(cfg.Projects) != len(builtin.Projects) || FindProject(cfg.Projects, "belgiantrain/til-order") < 0 {
		t.Fatalf("expected the %d built-in projects, got %d", len(builtin.Projects), len(cfg.Projects))
	}
	if p := cfg.Projects[FindProject(cfg.Projects, "belgiantrain/til-order")]; p.Path != filepath.Join("/ws", "til-order") {
		t.Fatalf("projects resolve against the workspace, got %q", p.Path)
	}
	if len(cfg.Environments) != 1 {
		t.Fatalf("the built-in manifest's environment must be available, got %d", len(cfg.Environments))
	}

	// Saved and loaded again: still there, not duplicated.
	_ = SaveConfig(cfg)
	again, firstRun := LoadConfig()
	if firstRun || len(again.Teams) != 1 || len(again.Projects) != len(builtin.Projects) {
		t.Fatalf("reload must keep one built-in team and its projects, got %d teams, %d projects", len(again.Teams), len(again.Projects))
	}
}

func TestRefreshProjects_NoBuiltinWhenTheProfileHasProjects(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)
	cfg := Config{BasePath: "/ws", Projects: []Project{{ID: "local/tool", Source: LocalSource, Name: "tool", Type: "go"}}}
	cfg.RefreshProjects()
	if len(cfg.Teams) != 0 || len(cfg.Projects) != 1 {
		t.Fatalf("a profile with its own projects is left alone, got %+v", cfg.Teams)
	}
}

func TestRefreshProjects_GitTeamReplacesTheBuiltin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ensureConfigDir(t)
	real := Team{Name: "go-dark-team", GitURL: "git@host:bt/go-dark-team.git", Manifest: "belgiantrain"}
	writeManifest(t, real, `{"team":"belgiantrain","projects":[{"name":"til-order","type":"maven"}]}`)

	cfg := Config{BasePath: "/ws", Teams: []Team{BuiltinTeam(), real}}
	cfg.RefreshProjects()

	if len(cfg.Teams) != 1 || cfg.Teams[0].Builtin || len(cfg.Projects) != 1 {
		t.Fatalf("the subscribed repo replaces the built-in team, got %+v / %d projects", cfg.Teams, len(cfg.Projects))
	}
}

func writeTemp(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ManifestFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
