package maven

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/exr462/go-dark/config"
)

// fixture writes a small multi-repo workspace shaped like the real one:
// a parent repo with a module, a library repo, and a service whose library
// version lives in a property used by dependencyManagement.
func fixture(t *testing.T) (string, []config.Project) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"parent/pom.xml": `<?xml version="1.0"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <groupId>be.bt</groupId>
  <artifactId>parent-pom</artifactId>
  <version>4.0.0</version>
  <packaging>pom</packaging>
</project>`,
		"parent/starter/pom.xml": `<project>
  <parent><groupId>be.bt</groupId><artifactId>parent-pom</artifactId><version>4.0.0</version></parent>
  <artifactId>rest-starter</artifactId>
</project>`,
		"lib/pom.xml": `<project>
  <parent>
    <groupId>be.bt</groupId>
    <artifactId>parent-pom</artifactId>
    <version>3.5.7</version>
  </parent>
  <groupId>be.bt.lib</groupId>
  <artifactId>lib-parent</artifactId>
  <version>1.10.0-SNAPSHOT</version>
</project>`,
		"lib/api/pom.xml": `<project>
  <parent><groupId>be.bt.lib</groupId><artifactId>lib-parent</artifactId><version>1.10.0-SNAPSHOT</version></parent>
  <artifactId>lib-api</artifactId>
</project>`,
		"lib/target/pom.xml": `<project><artifactId>ignored-build-output</artifactId></project>`,
		"svc/pom.xml": `<project>
    <parent>
        <groupId>be.bt</groupId>
        <artifactId>parent-pom</artifactId>
        <version>3.4.9</version> <!--to change when released-->
    </parent>
    <groupId>be.bt.svc</groupId>
    <artifactId>svc</artifactId>
    <version>2.0.0-SNAPSHOT</version>
    <properties>
        <lib.version>1.9.0</lib.version>
        <java.version>21</java.version>
    </properties>
    <dependencyManagement>
        <dependencies>
            <dependency>
                <groupId>be.bt.lib</groupId>
                <artifactId>lib-api</artifactId>
                <version>${lib.version}</version>
            </dependency>
            <dependency>
                <groupId>be.bt.lib</groupId>
                <artifactId>lib-parent</artifactId>
                <version>${lib.version}</version>
            </dependency>
        </dependencies>
    </dependencyManagement>
</project>`,
		"svc/service/pom.xml": `<project>
    <parent><groupId>be.bt.svc</groupId><artifactId>svc</artifactId><version>2.0.0-SNAPSHOT</version></parent>
    <artifactId>svc-service</artifactId>
    <dependencies>
        <dependency><groupId>be.bt</groupId><artifactId>rest-starter</artifactId></dependency>
        <dependency><groupId>be.bt.lib</groupId><artifactId>lib-api</artifactId></dependency>
        <dependency><groupId>be.bt.svc</groupId><artifactId>svc</artifactId><version>${project.version}</version></dependency>
        <dependency><groupId>org.junit</groupId><artifactId>junit</artifactId><version>5.0</version></dependency>
    </dependencies>
    <build><plugins><plugin>
        <groupId>be.bt</groupId><artifactId>rest-starter</artifactId><version><!-- pinned -->3.5.7</version>
    </plugin></plugins></build>
</project>`,
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var projects []config.Project
	for _, n := range []string{"parent", "lib", "svc"} {
		projects = append(projects, config.Project{ID: "bt/" + n, Path: filepath.Join(root, n), Fetched: true})
	}
	return root, projects
}

func TestScan_IndexesArtifactsWithInheritedCoordinates(t *testing.T) {
	_, projects := fixture(t)
	w, errs := Scan(projects)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	want := map[string]string{
		"be.bt:parent-pom":      "bt/parent@4.0.0",
		"be.bt:rest-starter":    "bt/parent@4.0.0",
		"be.bt.lib:lib-parent":  "bt/lib@1.10.0-SNAPSHOT",
		"be.bt.lib:lib-api":     "bt/lib@1.10.0-SNAPSHOT",
		"be.bt.svc:svc-service": "bt/svc@2.0.0-SNAPSHOT",
	}
	for ga, where := range want {
		found := false
		for k, a := range w.Artifacts {
			if k.String() == ga {
				found = a.ProjectID+"@"+a.Version == where
			}
		}
		if !found {
			t.Errorf("artifact %s should be %s", ga, where)
		}
	}
	if len(w.Artifacts) != 6 {
		t.Errorf("expected 6 artifacts (build output skipped), got %d", len(w.Artifacts))
	}
	if w.Projects["bt/lib"].Version != "1.10.0-SNAPSHOT" {
		t.Errorf("project version comes from the root pom, got %q", w.Projects["bt/lib"].Version)
	}
}

func TestScan_ReferencesAndEditLocations(t *testing.T) {
	_, projects := fixture(t)
	w, _ := Scan(projects)

	refs := w.RefsFrom("bt/svc")
	byKey := map[string]Ref{}
	for _, r := range refs {
		byKey[r.GA.ArtifactID+"/"+string(r.Kind)] = r
	}

	parent := byKey["parent-pom/parent"]
	if parent.Version != "3.4.9" || parent.Edit == nil || parent.Edit.Property != "" || parent.Edit.Current != "3.4.9" {
		t.Fatalf("parent ref must be a literal edit, got %+v", parent)
	}
	managed := byKey["lib-api/managed dependency"]
	if managed.Version != "1.9.0" || managed.Edit == nil || managed.Edit.Property != "lib.version" ||
		filepath.Base(filepath.Dir(managed.Edit.File)) != "svc" {
		t.Fatalf("managed ref must resolve through the property in the root pom, got %+v (%+v)", managed, managed.Edit)
	}
	if dep := byKey["lib-api/dependency"]; dep.Edit != nil || dep.Version != "" {
		t.Fatalf("a dependency without version (managed) is not editable, got %+v", dep)
	}
	if plugin := byKey["rest-starter/plugin"]; plugin.Edit != nil {
		t.Fatal("a version element with a comment inside must not be editable")
	}
	for _, r := range refs {
		if r.Target == "bt/svc" || r.GA.ArtifactID == "junit" {
			t.Fatalf("same-repo and third-party references must be excluded, got %+v", r)
		}
	}

	if got := w.SuggestedDeps("bt/svc"); !slices.Equal(got, []string{"bt/lib", "bt/parent"}) {
		t.Fatalf("SuggestedDeps = %v", got)
	}
	if got := w.SuggestedDeps("bt/parent"); len(got) != 0 {
		t.Fatalf("the parent depends on nothing, got %v", got)
	}
	if n := len(w.RefsTo("bt/lib")); n != 3 {
		t.Fatalf("expected 3 refs to lib (2 managed, 1 plain dependency), got %d", n)
	}
}

func TestApply_ChangesOnlyTheVersionText(t *testing.T) {
	_, projects := fixture(t)
	w, _ := Scan(projects)
	svcPom := filepath.Join(projects[2].Path, "pom.xml")
	before, _ := os.ReadFile(svcPom)

	var changes []Change
	for _, r := range w.RefsFrom("bt/svc") {
		if r.Edit == nil {
			continue
		}
		v := "3.5.8"
		if r.Target == "bt/lib" {
			v = "1.10.0-SNAPSHOT"
		}
		changes = append(changes, Change{Edit: *r.Edit, Value: v})
	}
	changed, err := Apply(changes)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] != svcPom {
		t.Fatalf("only the svc root pom should change, got %v", changed)
	}

	after, _ := os.ReadFile(svcPom)
	expected := strings.Replace(string(before), "<version>3.4.9</version>", "<version>3.5.8</version>", 1)
	expected = strings.Replace(expected, "<lib.version>1.9.0</lib.version>", "<lib.version>1.10.0-SNAPSHOT</lib.version>", 1)
	if string(after) != expected {
		t.Fatalf("unexpected file content:\n%s", after)
	}
	if !strings.Contains(string(after), "<!--to change when released-->") {
		t.Fatal("comments must be preserved")
	}

	// The scanned offsets are now stale: applying again must refuse.
	if _, err := Apply(changes); err == nil || !strings.Contains(err.Error(), "changed since") {
		t.Fatalf("stale edits must be refused, got %v", err)
	}
}

func TestApply_ConflictingValuesForOnePropertyFail(t *testing.T) {
	_, projects := fixture(t)
	w, _ := Scan(projects)
	var changes []Change
	for i, r := range w.RefsTo("bt/lib") {
		if r.Edit != nil {
			changes = append(changes, Change{Edit: *r.Edit, Value: []string{"1.0", "2.0"}[i%2]})
		}
	}
	if _, err := Apply(changes); err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("expected a conflict error, got %v", err)
	}
}

func TestScan_DuplicateProducerIsReported(t *testing.T) {
	root, projects := fixture(t)
	dup := filepath.Join(root, "parent-copy")
	_ = os.MkdirAll(dup, 0o755)
	data, _ := os.ReadFile(filepath.Join(root, "parent", "pom.xml"))
	_ = os.WriteFile(filepath.Join(dup, "pom.xml"), data, 0o644)
	projects = append(projects, config.Project{ID: "bt/parent-copy", Path: dup, Fetched: true})

	w, errs := Scan(projects)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "produced by both") {
		t.Fatalf("expected one duplicate-producer error, got %v", errs)
	}
	for k, a := range w.Artifacts {
		if k.ArtifactID == "parent-pom" && a.ProjectID != "bt/parent" {
			t.Fatal("the first configured producer must win")
		}
	}
}

func TestScan_SkipsUnclonedAndNonMavenProjects(t *testing.T) {
	_, projects := fixture(t)
	projects[0].Fetched = false
	projects = append(projects, config.Project{ID: "bt/web", Path: t.TempDir(), Fetched: true})
	w, _ := Scan(projects)
	if _, ok := w.Projects["bt/parent"]; ok {
		t.Fatal("uncloned projects are skipped")
	}
	if _, ok := w.Projects["bt/web"]; ok {
		t.Fatal("projects without a root pom are skipped")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"3.10.0", "3.9.1", 1}, {"3.5.7", "3.5.7", 0}, {"3.6.0-SNAPSHOT", "3.6.0", -1},
		{"4.0.0.ALPHA", "3.5.8", 1}, {"1.2", "1.2.1", -1},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestLatestReleaseTag(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	run("-c", "user.name=t", "-c", "user.email=t@e", "commit", "-q", "--allow-empty", "-m", "x")
	for _, tag := range []string{"parent-pom-3.5.7", "parent-pom-3.5.10", "parent-pom-3.6.0-SNAPSHOT", "v3.5.9", "other-9.9.9"} {
		run("tag", tag)
	}
	if got := LatestReleaseTag(dir, "parent-pom"); got != "3.5.10" {
		t.Fatalf("LatestReleaseTag = %q, want 3.5.10", got)
	}
	if got := LatestReleaseTag(t.TempDir(), "x"); got != "" {
		t.Fatalf("no repo means no tag, got %q", got)
	}
}
