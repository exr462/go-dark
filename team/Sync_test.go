package team

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/exr462/go-dark/config"
)

// newManifestRepo creates a bare git repo holding a go-dark.json manifest and
// returns its URL plus a function that commits a new manifest version.
func newManifestRepo(t *testing.T, manifest string) (string, func(string)) {
	t.Helper()
	root := t.TempDir()
	bare := filepath.Join(root, "rail-team.git")
	work := filepath.Join(root, "work")
	git(t, root, "init", "--quiet", "--bare", "-b", "main", bare)
	git(t, root, "clone", "--quiet", bare, work)

	commit := func(body string) {
		if err := os.WriteFile(filepath.Join(work, config.ManifestFileName), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		git(t, work, "add", ".")
		git(t, work, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "--quiet", "-m", "manifest")
		git(t, work, "push", "--quiet", "origin", "HEAD:main")
	}
	commit(manifest)
	return bare, commit
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestSubscribe_ClonesAndRecordsManifestTeam(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	url, _ := newManifestRepo(t, `{"team":"rail","projects":[{"name":"api","type":"maven"}]}`)

	teams, err := Subscribe(nil, url)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].Name != "rail-team" || teams[0].Manifest != "rail" {
		t.Fatalf("unexpected subscription %+v", teams)
	}

	again, err := Subscribe(teams, url)
	if err != nil || len(again) != 1 {
		t.Fatalf("subscribing twice must be a no-op, got %+v, %v", again, err)
	}
}

func TestSync_PullsManifestUpdates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	url, commit := newManifestRepo(t, `{"team":"rail","projects":[{"name":"api"}]}`)
	teams, err := Subscribe(nil, url)
	if err != nil {
		t.Fatal(err)
	}

	commit(`{"team":"rail","projects":[{"name":"api"},{"name":"web","type":"npm"}]}`)
	teams, errs := Sync(teams)
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	manifests, _, _ := config.LoadCachedManifests(teams)
	if len(manifests) != 1 || len(manifests[0].Projects) != 2 {
		t.Fatalf("expected the pulled manifest with 2 projects, got %+v", manifests)
	}
}

func TestSync_FailureKeepsPreviousCheckout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	url, _ := newManifestRepo(t, `{"team":"rail","projects":[{"name":"api"}]}`)
	teams, err := Subscribe(nil, url)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(url); err != nil { // the remote disappears
		t.Fatal(err)
	}

	synced, errs := Sync(teams)
	if len(errs) != 1 {
		t.Fatalf("expected one sync error, got %v", errs)
	}
	manifests, _, _ := config.LoadCachedManifests(synced)
	if len(manifests) != 1 {
		t.Fatal("the cached manifest must still be usable after a failed pull")
	}
}

func TestSubscribe_InvalidURLFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := Subscribe(nil, filepath.Join(t.TempDir(), "missing.git")); err == nil {
		t.Fatal("expected an error for an unreachable repository")
	}
	if _, err := Subscribe(nil, "  "); err == nil {
		t.Fatal("expected an error for an empty URL")
	}
}

func TestUnsubscribe_RemovesCheckout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	url, _ := newManifestRepo(t, `{"team":"rail","projects":[]}`)
	teams, _ := Subscribe(nil, url)
	dir, _ := config.TeamDir(teams[0])

	if left := Unsubscribe(teams, "rail-team"); len(left) != 0 {
		t.Fatalf("expected no teams left, got %+v", left)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("the team checkout must be removed")
	}
}

func TestApplySynced_LeavesConcurrentChangesAlone(t *testing.T) {
	current := []config.Team{{Name: "a"}, {Name: "new"}}
	synced := []config.Team{{Name: "a", Manifest: "alpha"}, {Name: "removed", Manifest: "x"}}

	got := ApplySynced(current, synced)

	if len(got) != 2 || got[0].Manifest != "alpha" || got[1].Name != "new" {
		t.Fatalf("unexpected result %+v", got)
	}
}

func TestSync_SkipsTheBuiltinTeam(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	teams, errs := Sync([]config.Team{config.BuiltinTeam()})
	if len(errs) != 0 || len(teams) != 1 || !teams[0].Builtin {
		t.Fatalf("the built-in team has no repo to pull, got %+v, %v", teams, errs)
	}
}

func TestSubscribe_ReplacesASameNamedBuiltinTeam(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	url, _ := newManifestRepo(t, `{"team":"belgiantrain","projects":[{"name":"til-order","type":"maven"}]}`)
	// The built-in team's directory already exists with its written manifest.
	builtin := config.Team{Name: "rail-team", Manifest: "belgiantrain", Builtin: true}
	dir, _ := config.TeamDir(builtin)
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, config.ManifestFileName), []byte(`{"team":"belgiantrain","projects":[]}`), 0o644)

	teams, err := Subscribe([]config.Team{builtin}, url)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].Builtin || teams[0].GitURL != url || teams[0].Manifest != "belgiantrain" {
		t.Fatalf("the repo must replace the built-in team, got %+v", teams)
	}
}
