package model

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/maven"
)

func writePom(t *testing.T, dir, content string) {
	t.Helper()
	_ = os.MkdirAll(dir, 0o755)
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildGraph_MarksSuggestionsUnconfirmedAndDrift(t *testing.T) {
	root := t.TempDir()
	writePom(t, filepath.Join(root, "parent"), `<project><groupId>g</groupId><artifactId>parent</artifactId><version>3.5.8</version></project>`)
	writePom(t, filepath.Join(root, "auth"), `<project><parent><groupId>g</groupId><artifactId>parent</artifactId><version>3.5.8</version></parent><artifactId>auth</artifactId><version>1.0</version></project>`)
	writePom(t, filepath.Join(root, "svc"), `<project><parent><groupId>g</groupId><artifactId>parent</artifactId><version>3.4.9</version></parent><artifactId>svc</artifactId><version>2.0</version>
	  <dependencies><dependency><groupId>g</groupId><artifactId>auth</artifactId><version>1.0</version></dependency></dependencies></project>`)
	projects := []config.Project{
		{ID: "t/parent", Path: filepath.Join(root, "parent"), Fetched: true},
		{ID: "t/auth", Path: filepath.Join(root, "auth"), Fetched: true},
		// The manifest says svc depends on parent and docs; the poms say parent and auth.
		{ID: "t/svc", Path: filepath.Join(root, "svc"), Fetched: true, DependsOn: []string{"t/parent", "t/docs"}},
		{ID: "t/docs", Path: filepath.Join(root, "docs"), Fetched: true},
	}
	w, errs := maven.Scan(projects)
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	rows, err := BuildGraph(projects, w)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]GraphRow{}
	for _, r := range rows {
		byID[r.Project.ID] = r
	}

	svc := byID["t/svc"]
	if !slices.Equal(svc.Suggested, []string{"t/auth"}) {
		t.Errorf("auth is referenced by the poms but missing from the manifest, got %v", svc.Suggested)
	}
	if len(svc.Unconfirmed) != 0 {
		t.Errorf("docs isn't a Maven project, so it can't be judged; got %v", svc.Unconfirmed)
	}
	if len(svc.Drift) != 1 || svc.Drift[0].GA.ArtifactID != "parent" || svc.Drift[0].Version != "3.4.9" {
		t.Errorf("svc uses parent 3.4.9 while parent is 3.5.8, got %+v", svc.Drift)
	}
	if svc.Version != "2.0" || !svc.Scanned || byID["t/docs"].Scanned {
		t.Errorf("unexpected scan info: %+v / %+v", svc, byID["t/docs"])
	}
	if byID["t/svc"].Wave <= byID["t/parent"].Wave {
		t.Error("svc must be in a later wave than parent")
	}
	if len(byID["t/auth"].Drift) != 0 {
		t.Error("auth uses the current parent version")
	}

	// A manifest dependency the poms don't confirm is flagged.
	projects[2].DependsOn = []string{"t/parent", "t/auth"}
	projects[1].DependsOn = []string{"t/svc"} // wrong direction: auth's poms don't use svc
	projects[2].DependsOn = []string{"t/parent"}
	rows, _ = BuildGraph(projects, w)
	for _, r := range rows {
		if r.Project.ID == "t/auth" && !slices.Equal(r.Unconfirmed, []string{"t/svc"}) {
			t.Errorf("expected auth -> svc to be unconfirmed, got %v", r.Unconfirmed)
		}
	}
}

func TestBuildGraph_WithoutScanStillLaysOutWaves(t *testing.T) {
	projects := []config.Project{{ID: "a", Fetched: true}, {ID: "b", Fetched: true, DependsOn: []string{"a"}}, {ID: "c"}}
	rows, err := BuildGraph(projects, nil)
	if err != nil || len(rows) != 2 || rows[0].Project.ID != "a" || rows[1].Wave != 1 {
		t.Fatalf("unexpected rows %+v, %v", rows, err)
	}
}
