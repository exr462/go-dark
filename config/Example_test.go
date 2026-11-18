package config

import "testing"

// The example manifest is what teams copy; it must stay loadable and in
// sync with the registry it replaced.
func TestExampleBelgiantrainManifestMatchesLegacyRegistry(t *testing.T) {
	m, err := LoadManifest("../examples/belgiantrain/" + ManifestFileName)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Projects) != len(legacyRegistry) {
		t.Fatalf("expected %d projects, got %d", len(legacyRegistry), len(m.Projects))
	}
	for i, lp := range legacyRegistry {
		mp := m.Projects[i]
		if mp.Name != lp.Name || mp.Buildable != lp.Buildable || mp.Deployable != lp.Deployable || len(mp.DependsOn) != len(lp.Dependencies) {
			t.Errorf("project %d differs: manifest %+v, legacy %+v", i, mp, lp)
		}
	}
	if m.Projects[0].GitURL != "git@bitbucket.org:belgiantrain/ypto-maven-parent.git" {
		t.Errorf("unexpected default git URL %q", m.Projects[0].GitURL)
	}
}
