// Package deploy deploys the workspace to the local cluster: it builds each
// deployable service's image into minikube's docker daemon and installs its
// Helm chart with the service's values.local-rancher.yaml, wave by wave in
// dependency order.
package deploy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kube"
	"gopkg.in/yaml.v3"
)

// LocalValuesFile marks a chart that can be deployed locally.
const LocalValuesFile = "values.local-rancher.yaml"

// Target is one service to deploy.
type Target struct {
	Project     config.Project
	ChartDir    string // helm/<chart> in the project
	ChartName   string
	LocalValues string // the chart's values.local-rancher.yaml
	Image, Tag  string // main.image / main.version from LocalValues
	DockerDir   string // directory holding the Dockerfile (build context)
	Skip        string // why it can't be deployed (empty when it can)
}

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
	matches, _ := filepath.Glob(filepath.Join(p.Path, "helm", "*", LocalValuesFile))
	if len(matches) == 0 {
		t.Skip = "no helm/<chart>/" + LocalValuesFile
		return t
	}
	t.LocalValues = matches[0]
	t.ChartDir = filepath.Dir(matches[0])

	chart, err := readYAML(filepath.Join(t.ChartDir, "Chart.yaml"))
	if err != nil {
		t.Skip = "unreadable Chart.yaml: " + err.Error()
		return t
	}
	t.ChartName, _ = chart["name"].(string)
	if t.ChartName == "" {
		t.ChartName = filepath.Base(t.ChartDir)
	}

	values, err := readYAML(t.LocalValues)
	if err != nil {
		t.Skip = "unreadable " + LocalValuesFile + ": " + err.Error()
		return t
	}
	main, _ := values["main"].(map[string]any)
	if main == nil || main["image"] == nil || main["version"] == nil {
		t.Skip = LocalValuesFile + " has no main.image / main.version"
		return t
	}
	t.Image = fmt.Sprint(main["image"])
	t.Tag = fmt.Sprint(main["version"])

	t.DockerDir = findDockerfile(p.Path)
	if t.DockerDir == "" {
		t.Skip = "no Dockerfile"
	}
	return t
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

// ImageArgv builds the target's image (run with minikube's docker env).
func ImageArgv(t Target) []string {
	return []string{"docker", "build", "-t", t.Image + ":" + t.Tag, t.DockerDir}
}

// LocalChartVersion replaces the charts' {JENKINS_VERSION} placeholder,
// which Helm rejects, in the temporary copy that gets installed.
const LocalChartVersion = "0.0.0-local"

// PrepareChart copies the chart to dir with a valid version and reports
// whether it has dependencies to build. The project's chart is untouched.
func PrepareChart(t Target, dir string) (chartDir string, hasDeps bool, err error) {
	chartDir = filepath.Join(dir, t.ChartName)
	if err := os.CopyFS(chartDir, os.DirFS(t.ChartDir)); err != nil {
		return "", false, err
	}
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	data, err := os.ReadFile(chartFile)
	if err != nil {
		return "", false, err
	}
	data = versionLine.ReplaceAll(data, []byte("version: "+LocalChartVersion))
	if err := os.WriteFile(chartFile, data, 0o644); err != nil {
		return "", false, err
	}
	return chartDir, strings.Contains(string(data), "\ndependencies:"), nil
}

var versionLine = regexp.MustCompile(`(?m)^version:.*$`)

// HelmArgv installs or upgrades the prepared chart with the local overrides
// (Helm applies the chart's own values.yaml first), waiting for rollout.
func HelmArgv(c kube.Cluster, t Target, chartDir string) []string {
	args := []string{"upgrade", "--install", t.ChartName, chartDir, "-f", t.LocalValues, "--create-namespace", "--wait", "--timeout", "5m"}
	return append([]string{"helm"}, c.Args(args...)...)
}

// DependencyBuildArgv fetches a prepared chart's dependencies.
func DependencyBuildArgv(chartDir string) []string {
	return []string{"helm", "dependency", "build", chartDir}
}
