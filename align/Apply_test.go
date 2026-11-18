package align

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/maven"
	"github.com/exr462/go-dark/model"
)

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// workspace creates parent + svc repos, each its own git repo with one commit.
func workspace(t *testing.T) (map[string]string, []config.Project) {
	t.Helper()
	// Isolate from the developer's git config (identity, hooks, signing).
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	root := t.TempDir()
	poms := map[string]string{
		"parent": `<project><groupId>g</groupId><artifactId>parent</artifactId><version>3.5.8</version></project>`,
		"svc": `<project>
  <parent><groupId>g</groupId><artifactId>parent</artifactId><version>3.4.9</version></parent>
  <artifactId>svc</artifactId><version>1.0</version>
  <properties><parent.lib>3.5.0</parent.lib></properties>
  <dependencyManagement><dependencies>
    <dependency><groupId>g</groupId><artifactId>parent</artifactId><version>${parent.lib}</version></dependency>
  </dependencies></dependencyManagement>
</project>
`,
	}
	dirs := map[string]string{}
	var projects []config.Project
	for _, name := range []string{"parent", "svc"} {
		dir := filepath.Join(root, name)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(poms[name]), 0o644)
		_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644)
		run(t, dir, "init", "-q", "-b", "main")
		run(t, dir, "add", ".")
		run(t, dir, "-c", "user.name=t", "-c", "user.email=t@e", "commit", "-q", "-m", "init")
		dirs["t/"+name] = dir
		projects = append(projects, config.Project{ID: "t/" + name, Path: dir, Fetched: true})
	}
	return dirs, projects
}

func plan(t *testing.T, projects []config.Project) map[string][]maven.Change {
	t.Helper()
	w, errs := maven.Scan(projects)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	rows := model.AlignRows(w, nil)
	if len(rows) != 1 || rows[0].ProjectID != "t/parent" {
		t.Fatalf("expected one drifting project (parent), got %+v", rows)
	}
	return model.AlignChanges(rows, map[string]model.AlignTarget{})
}

func TestApply_CommitsOnBranchOnlyTheEditedPom(t *testing.T) {
	dirs, projects := workspace(t)
	svc := dirs["t/svc"]
	// An unrelated staged change must not end up in the commit.
	_ = os.WriteFile(filepath.Join(svc, "README.md"), []byte("staged\n"), 0o644)
	run(t, svc, "add", "README.md")

	results := Apply(plan(t, projects), dirs, Identity{Name: "Go Dark", Email: "go@dark"})

	if len(results) != 1 || !results[0].OK() {
		t.Fatalf("expected one committed repo, got %+v", results)
	}
	if branch := run(t, svc, "rev-parse", "--abbrev-ref", "HEAD"); branch != Branch {
		t.Fatalf("expected to be on %s, got %s", Branch, branch)
	}
	if files := run(t, svc, "show", "--name-only", "--format=", "HEAD"); files != "pom.xml" {
		t.Fatalf("the commit must only contain pom.xml, got %q", files)
	}
	if msg := run(t, svc, "log", "-1", "--format=%s|%an"); msg != CommitMessage+"|Go Dark" {
		t.Fatalf("unexpected commit %q", msg)
	}
	if staged := run(t, svc, "diff", "--cached", "--name-only"); staged != "README.md" {
		t.Fatalf("the unrelated change must stay staged, got %q", staged)
	}
	pom, _ := os.ReadFile(filepath.Join(svc, "pom.xml"))
	if !strings.Contains(string(pom), "<version>3.5.8</version></parent>") || !strings.Contains(string(pom), "<parent.lib>3.5.8</parent.lib>") {
		t.Fatalf("both the parent and the property must be aligned:\n%s", pom)
	}
	if run(t, dirs["t/parent"], "rev-parse", "--abbrev-ref", "HEAD") != "main" {
		t.Fatal("repos without changes must not be touched")
	}

	// Aligned now: nothing more to do.
	w, _ := maven.Scan(projects)
	if rows := model.AlignRows(w, nil); len(rows) != 0 {
		t.Fatalf("after alignment there must be no drift, got %+v", rows)
	}
}

func TestApply_SkipsDirtyPomAndExistingBranch(t *testing.T) {
	dirs, projects := workspace(t)
	svc := dirs["t/svc"]
	changes := plan(t, projects)

	pomPath := filepath.Join(svc, "pom.xml")
	orig, _ := os.ReadFile(pomPath)
	_ = os.WriteFile(pomPath, append(orig, []byte("<!-- local edit -->\n")...), 0o644)
	res := Apply(changes, dirs, Identity{})
	if res[0].OK() || !strings.Contains(res[0].Skipped, "uncommitted changes in pom.xml") {
		t.Fatalf("a dirty pom must skip the repo, got %+v", res[0])
	}
	_ = os.WriteFile(pomPath, orig, 0o644)

	run(t, svc, "branch", Branch)
	res = Apply(changes, dirs, Identity{})
	if res[0].OK() || !strings.Contains(res[0].Skipped, "already exists") {
		t.Fatalf("an existing branch that isn't checked out must skip the repo, got %+v", res[0])
	}
	if branch := run(t, svc, "rev-parse", "--abbrev-ref", "HEAD"); branch != "main" {
		t.Fatal("a skipped repo must stay on its branch")
	}

	// Already on the branch: commits there.
	run(t, svc, "switch", "-q", Branch)
	res = Apply(changes, dirs, Identity{Name: "x", Email: "x@y"})
	if !res[0].OK() {
		t.Fatalf("on the alignment branch it must commit, got %+v", res[0])
	}
}

func TestAlignRows_ReleaseTargetAndKeep(t *testing.T) {
	_, projects := workspace(t)
	w, _ := maven.Scan(projects)
	rows := model.AlignRows(w, map[string]string{"t/parent": "3.5.7"})

	if rows[0].Versions["3.4.9"] != 1 || rows[0].Versions["3.5.0"] != 1 || model.VersionSummary(rows[0].Versions) != "3.4.9 ×1, 3.5.0 ×1" {
		t.Fatalf("unexpected histogram %v", rows[0].Versions)
	}
	release := model.AlignChanges(rows, map[string]model.AlignTarget{"t/parent": model.AlignToRelease})
	if len(release["t/svc"]) != 2 || release["t/svc"][0].Value != "3.5.7" {
		t.Fatalf("release target must use the tag version, got %+v", release)
	}
	if keep := model.AlignChanges(rows, map[string]model.AlignTarget{"t/parent": model.AlignKeep}); len(keep) != 0 {
		t.Fatalf("keep must produce no changes, got %+v", keep)
	}
	preview := model.PreviewChanges(release["t/svc"])
	if len(preview) != 2 || !strings.Contains(preview[0].Old, "3.4.9") || !strings.Contains(preview[0].New, "3.5.7") || preview[0].Line != 2 {
		t.Fatalf("unexpected preview %+v", preview)
	}
}
