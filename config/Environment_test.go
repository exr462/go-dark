package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest_EnvironmentGetsDefaultsAndIsValidated(t *testing.T) {
	path := filepath.Join(t.TempDir(), ManifestFileName)
	_ = os.WriteFile(path, []byte(`{"team":"rail","projects":[],
		"environment":{"application":"til","domain":"lcl-intern-belgiantrain.be","ingress":{"strip_prefix":true}}}`), 0o644)

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	e := m.Environment
	if e == nil {
		t.Fatal("the environment must be loaded")
	}
	if e.Name != "lcl" || e.ConfigFrom != "dev" || e.Namespace != "til-lcl" || e.ValuesFile() != "values.Development.yaml" ||
		e.Ingress.Controller != IngressNginxInc || !e.Ingress.StripPrefix || e.Database.Type != DatabaseOracleFree || e.Database.UsersFrom != UsersFromConfig {
		t.Fatalf("defaults not applied: %+v", *e)
	}
	if e.Host("api") != "api.til.lcl-intern-belgiantrain.be" || e.Host("rancher") != "rancher.lcl-intern-belgiantrain.be" || e.Host("nope") != "" {
		t.Fatalf("hosts must expand the placeholders: %v", e.Hosts)
	}
	if got := strings.Join(e.HostNames(), ","); got != "api.til.lcl-intern-belgiantrain.be,rancher.lcl-intern-belgiantrain.be" {
		t.Fatalf("HostNames = %q", got)
	}
	if envs := Environments([]Manifest{m, {Team: "other"}}); len(envs) != 1 || envs[0].Team != "rail" {
		t.Fatalf("Environments lists the teams that declare one, got %+v", envs)
	}
}

func TestLoadManifest_RejectsBrokenEnvironments(t *testing.T) {
	cases := map[string]string{
		"no application":    `{"domain":"x.local"}`,
		"no domain":         `{"application":"til"}`,
		"bad name":          `{"application":"til","domain":"x.local","name":"Lcl Env"}`,
		"bad namespace":     `{"application":"til","domain":"x.local","namespace":"til/lcl"}`,
		"bad host":          `{"application":"til","domain":"x.local","hosts":{"api":"api_til.{domain}"}}`,
		"missing host":      `{"application":"til","domain":"x.local","hosts":{"api":""}}`,
		"unknown ingress":   `{"application":"til","domain":"x.local","ingress":{"controller":"traefik"}}`,
		"unknown database":  `{"application":"til","domain":"x.local","database":{"type":"postgres"}}`,
		"no values file":    `{"application":"til","domain":"x.local","config_from":"qa"}`,
		"bad database user": `{"application":"til","domain":"x.local","database":{"users":[{"app":"a b"}]}}`,
	}
	for name, env := range cases {
		path := filepath.Join(t.TempDir(), ManifestFileName)
		_ = os.WriteFile(path, []byte(`{"team":"rail","projects":[],"environment":`+env+`}`), 0o644)
		if _, err := LoadManifest(path); err == nil || !strings.Contains(err.Error(), "environment") {
			t.Errorf("%s: expected an environment error, got %v", name, err)
		}
	}
}

func TestLoadManifest_ExampleManifestHasAValidEnvironment(t *testing.T) {
	m, err := LoadManifest(filepath.Join("..", "examples", "belgiantrain", ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if m.Environment == nil || m.Environment.Application != "til" || m.Environment.Host("api") != "api.til.lcl-intern-belgiantrain.be" {
		t.Fatalf("unexpected example environment %+v", m.Environment)
	}
}

func TestRefreshProjects_CollectsTeamEnvironments(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	team := Team{Name: "rail-manifests", GitURL: "git@host:rail/rail-manifests.git", Manifest: "rail"}
	writeManifest(t, team, `{"team":"rail","projects":[{"name":"api","type":"maven"}],
		"environment":{"application":"til","domain":"lcl.local"}}`)
	cfg := Config{Teams: []Team{team}, BasePath: t.TempDir()}

	cfg.RefreshProjects()

	if len(cfg.Environments) != 1 || cfg.Environments[0].Team != "rail" || cfg.Environments[0].Namespace != "til-lcl" {
		t.Fatalf("expected the team's environment, got %+v", cfg.Environments)
	}
}
