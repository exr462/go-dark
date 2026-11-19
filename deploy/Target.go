// Package deploy deploys the workspace to the local Rancher sandbox: it
// builds each deployable service's image, side-loads it into the cluster and
// installs its Helm chart configured like a real environment (dev by
// default) except for the database, wave by wave in dependency order.
package deploy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/env"
	"github.com/exr462/go-dark/kube"
	"github.com/exr462/go-dark/maven"
	"gopkg.in/yaml.v3"
)

// LocalValuesFile is an optional, developer-owned override layer in a chart
// (the team's untracked values.local-rancher.yaml): its main.image and
// main.version name the local image.
const LocalValuesFile = "values.local-rancher.yaml"

// OverlayFile is the values file go-dark writes into the staged chart.
const OverlayFile = "values.go-dark.yaml"

// EnvValuesFiles maps an environment to the chart values file its pipeline
// deploys with, when the team manifest doesn't say.
var EnvValuesFiles = config.DefaultValuesFiles

// Target is one service to deploy.
type Target struct {
	Project     config.Project
	ChartDir    string // helm/<chart> in the project
	ChartName   string
	LocalValues string // the chart's values.local-rancher.yaml, if any
	Image, Tag  string // the local image to build and deploy
	DockerDir   string // directory holding the Dockerfile (build context)
	Skip        string // why it can't be deployed (empty when it can)
}

// Ref is the image reference.
func (t Target) Ref() string { return t.Image + ":" + t.Tag }

// Plan turns dependency waves of deployable projects (model.ResolveDeployOrder)
// into waves of targets. Projects that can't be deployed locally come back
// in skipped, with why.
func Plan(projectWaves [][]config.Project) (waves [][]Target, skipped []Target) {
	for _, pw := range projectWaves {
		var wave []Target
		for _, p := range pw {
			t := discover(p)
			if t.Skip != "" {
				skipped = append(skipped, t)
				continue
			}
			wave = append(wave, t)
		}
		if len(wave) > 0 {
			waves = append(waves, wave)
		}
	}
	return waves, skipped
}

func discover(p config.Project) Target {
	t := Target{Project: p}
	t.ChartDir = findChart(p.Path)
	if t.ChartDir == "" {
		t.Skip = "no helm/<chart> with a main: section"
		return t
	}
	chart, err := readYAML(filepath.Join(t.ChartDir, "Chart.yaml"))
	if err != nil {
		t.Skip = "unreadable Chart.yaml: " + err.Error()
		return t
	}
	t.ChartName, _ = chart["name"].(string)
	if t.ChartName == "" {
		t.ChartName = filepath.Base(t.ChartDir)
	}

	if local := filepath.Join(t.ChartDir, LocalValuesFile); fileExists(local) {
		t.LocalValues = local
		values, err := readYAML(local)
		if err != nil {
			t.Skip = "unreadable " + LocalValuesFile + ": " + err.Error()
			return t
		}
		if main, _ := values["main"].(map[string]any); main != nil {
			if main["image"] != nil {
				t.Image = fmt.Sprint(main["image"])
			}
			if main["version"] != nil {
				t.Tag = fmt.Sprint(main["version"])
			}
		}
	}
	if t.Image == "" {
		t.Image = "local/" + t.ChartName
	}
	if t.Tag == "" {
		if t.Tag = projectVersion(p.Path); t.Tag == "" {
			t.Skip = "no version in pom.xml or " + LocalValuesFile
			return t
		}
	}

	t.DockerDir = findDockerfile(p.Path)
	if t.DockerDir == "" {
		t.Skip = "no Dockerfile"
	}
	return t
}

// findChart returns the first helm/<chart> whose values.yaml has the
// ypto layout's main: section.
func findChart(root string) string {
	matches, _ := filepath.Glob(filepath.Join(root, "helm", "*", "values.yaml"))
	for _, m := range matches {
		values, err := readYAML(m)
		if err != nil {
			continue
		}
		if _, ok := values["main"].(map[string]any); ok && fileExists(filepath.Join(filepath.Dir(m), "Chart.yaml")) {
			return filepath.Dir(m)
		}
	}
	return ""
}

func projectVersion(root string) string {
	pom, err := maven.ParsePOM(filepath.Join(root, "pom.xml"))
	if err != nil || strings.Contains(pom.Self.Version, "${") {
		return ""
	}
	return pom.Self.Version
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func readYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Charts carry Jenkins placeholders like {JENKINS_VERSION}, which YAML
	// would read as a flow mapping; quote them so the file parses.
	data = placeholder.ReplaceAll(data, []byte(`"$0"`))
	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

var placeholder = regexp.MustCompile(`\{[A-Z_]+\}`)

// findDockerfile returns the shallowest directory under root with a
// Dockerfile, ignoring build output.
func findDockerfile(root string) string {
	best, bestDepth := "", 1<<30
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == "target" || d.Name() == "node_modules" || d.Name() == ".git") {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == "Dockerfile" {
			if depth := strings.Count(path, string(os.PathSeparator)); depth < bestDepth {
				best, bestDepth = filepath.Dir(path), depth
			}
		}
		return nil
	})
	return best
}

// ImageArgv builds the target's image with the host's docker.
func ImageArgv(t Target) []string {
	return []string{"docker", "build", "-t", t.Ref(), t.DockerDir}
}

// LocalChartVersion is the chart version used when the service version
// isn't valid semver.
const LocalChartVersion = "0.0.0-local"

// ChartVersion is the version the staged chart gets: the service version,
// or LocalChartVersion when that isn't semver.
func ChartVersion(t Target) string {
	if semver.MatchString(t.Tag) {
		return t.Tag
	}
	return LocalChartVersion
}

var semver = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// Env is how the staged charts are configured.
type Env struct {
	Name       string // chart environment, e.g. lcl
	ConfigFrom string // environment whose values/config/secrets are copied, e.g. dev
	ValuesFile string // the ConfigFrom environment's values file (default: EnvValuesFiles)
	DBURL      string
	DBPassword string
	Arch       string // node architecture
}

// EnvFromConfig turns the legacy local cluster of config.json into an Env
// (used when no team manifest declares an environment).
func EnvFromConfig(c config.LocalCluster, arch string) Env {
	return Env{Name: c.Env, ConfigFrom: c.ConfigFrom, DBURL: c.Database.URL, DBPassword: c.Database.Password, Arch: arch}
}

// EnvFromLocal turns a team environment into an Env.
func EnvFromLocal(e env.Env, arch string) Env {
	return Env{Name: e.Name, ConfigFrom: e.ConfigFrom, ValuesFile: e.ValuesFile(), DBURL: e.DBURL(), DBPassword: e.DBPassword, Arch: arch}
}

// valuesFile is the values file of the environment the chart is copied from.
func (e Env) valuesFile() string {
	if e.ValuesFile != "" {
		return e.ValuesFile
	}
	return EnvValuesFiles[e.ConfigFrom]
}

// Staged is a chart copy ready to install.
type Staged struct {
	Dir     string
	Values  []string // -f files, in order
	HasDeps bool
}

// PrepareChart copies the chart to dir and makes it deployable as env:
//   - Chart.yaml gets a valid version instead of {JENKINS_VERSION};
//   - config/<from> and secrets/<from> are copied to config/<env> and
//     secrets/<env>, with the datasource URL pointed at the local database;
//   - an overlay sets the environment, the local image, the node's
//     architecture, one replica, no Dynatrace and the database passwords.
//
// Values are layered like the pipeline does: values.yaml (implicit), the
// ConfigFrom environment's values file, the developer's LocalValuesFile,
// then the overlay. The project's chart is untouched.
func PrepareChart(t Target, dir string, env Env) (Staged, error) {
	chartDir := filepath.Join(dir, t.ChartName)
	if err := os.CopyFS(chartDir, os.DirFS(t.ChartDir)); err != nil {
		return Staged{}, err
	}
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	data, err := os.ReadFile(chartFile)
	if err != nil {
		return Staged{}, err
	}
	data = versionLine.ReplaceAll(data, []byte("version: "+ChartVersion(t)))
	if err := os.WriteFile(chartFile, data, 0o644); err != nil {
		return Staged{}, err
	}
	st := Staged{Dir: chartDir, HasDeps: strings.Contains(string(data), "\ndependencies:")}

	if err := copyEnvDir(chartDir, "config", env, true); err != nil {
		return st, err
	}
	if err := copyEnvDir(chartDir, "secrets", env, false); err != nil {
		return st, err
	}

	var layers []map[string]any
	if f := env.valuesFile(); f != "" && fileExists(filepath.Join(chartDir, f)) {
		st.Values = append(st.Values, filepath.Join(chartDir, f))
	}
	for _, f := range append([]string{filepath.Join(chartDir, "values.yaml")}, st.Values...) {
		if v, err := readYAML(f); err == nil {
			layers = append(layers, v)
		}
	}
	if t.LocalValues != "" {
		st.Values = append(st.Values, t.LocalValues)
		if v, err := readYAML(t.LocalValues); err == nil {
			layers = append(layers, v)
		}
	}
	overlay, err := yaml.Marshal(Overlay(t, env, layers...))
	if err != nil {
		return st, err
	}
	overlayFile := filepath.Join(chartDir, OverlayFile)
	if err := os.WriteFile(overlayFile, overlay, 0o644); err != nil {
		return st, err
	}
	st.Values = append(st.Values, overlayFile)
	return st, nil
}

var versionLine = regexp.MustCompile(`(?m)^version:.*$`)

// copyEnvDir copies <chart>/<kind>/<from>/* to <chart>/<kind>/<env>/. For
// config, properties pointing at a database are repointed at env.DBURL.
func copyEnvDir(chartDir, kind string, env Env, rewrite bool) error {
	from := filepath.Join(chartDir, kind, env.ConfigFrom)
	if _, err := os.Stat(from); err != nil || env.Name == env.ConfigFrom {
		return nil
	}
	to := filepath.Join(chartDir, kind, env.Name)
	if err := os.RemoveAll(to); err != nil {
		return err
	}
	if err := os.CopyFS(to, os.DirFS(from)); err != nil {
		return err
	}
	if !rewrite || env.DBURL == "" {
		return nil
	}
	files, _ := filepath.Glob(filepath.Join(to, "*.properties"))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		data = datasourceURL.ReplaceAll(data, []byte("${1}="+env.DBURL))
		if err := os.WriteFile(f, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// datasourceURL matches the JDBC URL properties; ${...} indirections (e.g.
// spring.liquibase.url=${spring.datasource.url}) are left alone.
var datasourceURL = regexp.MustCompile(`(?m)^((?:spring\.datasource|spring\.liquibase|liquibase\.datasource)\.(?:jdbc-)?url)=(?:jdbc:[^\r\n]*|\$\{DB_URL\})`)

// Overlay is the values go-dark puts on top of the chart's own layers.
// Secrets under data: whose key mentions a database get the local database
// password; the rest keep their value from the layers, or a local dummy
// (the real ones are injected by the pipeline and aren't in the repo).
func Overlay(t Target, env Env, layers ...map[string]any) map[string]any {
	main := map[string]any{
		"env":             env.Name,
		"image":           t.Image,
		"version":         t.Tag,
		"imagePullPolicy": "Never",
		"dynatrace":       false,
		"replicas":        1,
		"minReplicas":     1,
		"maxReplicas":     1,
	}
	if env.Arch != "" {
		main["arch"] = env.Arch
	}
	out := map[string]any{"ypto": map[string]any{"env": env.Name}, "main": main}

	data := map[string]any{}
	for _, l := range layers {
		d, _ := l["data"].(map[string]any)
		for k, v := range d {
			// "???????" is the pipeline's placeholder for an injected secret.
			if v == nil || strings.Trim(fmt.Sprint(v), "?") == "" {
				if _, ok := data[k]; !ok {
					data[k] = nil
				}
				continue
			}
			data[k] = v
		}
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	secrets := map[string]any{}
	for _, k := range keys {
		switch {
		case databaseKey.MatchString(k):
			secrets[k] = env.DBPassword
		case data[k] == nil:
			secrets[k] = "changeit-local"
		default:
			secrets[k] = data[k]
		}
	}
	if len(secrets) > 0 {
		out["data"] = secrets
	}
	return out
}

var databaseKey = regexp.MustCompile(`(?i)db|database|datasource|liquibase`)

// HelmArgv installs or upgrades the staged chart with its value layers,
// waiting for the rollout. Spring services with a database can take
// minutes to become ready (their startupProbe allows 10).
func HelmArgv(c kube.Cluster, t Target, st Staged) []string {
	args := []string{"upgrade", "--install", t.ChartName, st.Dir}
	for _, f := range st.Values {
		args = append(args, "-f", f)
	}
	args = append(args, "--create-namespace", "--wait", "--timeout", "11m")
	return append([]string{"helm"}, c.Args(args...)...)
}

// DependencyBuildArgv fetches a staged chart's dependencies.
func DependencyBuildArgv(chartDir string) []string {
	return []string{"helm", "dependency", "build", chartDir}
}
