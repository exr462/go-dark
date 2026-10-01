// Package deploy simulates deploying the workspace's dependency-ordered
// components to a Rancher/Kubernetes-style environment. It performs no real
// network or cluster I/O; it produces a deterministic, dependency-aware
// transcript of the steps a real rollout would take (namespace, manifests,
// scheduling, rollout status, service exposure), driven by the same
// dependency engine used for local builds (model.ResolveDeployOrder).
package deploy

import (
	"fmt"
	"strings"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

// DefaultNamespace is used when the caller doesn't supply one.
const DefaultNamespace = "go-dark-workspace"

// LogEntry is a single, pre-computed line of the simulated rollout
// transcript. Wave/Project are left zero-valued for banner lines that span
// the whole rollout (namespace creation, completion banner, wave headers).
type LogEntry struct {
	Wave    int
	Project string
	Text    string
}

// Plan resolves the Deployable dependency graph for the given active
// projects/registry and builds the full simulated rollout transcript, wave
// by wave. It is a pure function - safe and fast to unit test - the actual
// "streaming" effect is applied by the UI layer afterwards.
func Plan(activeProjects []config.Project, registry []config.AvailableProject, namespace string) ([]LogEntry, error) {
	if namespace == "" {
		namespace = DefaultNamespace
	}

	_, waves, err := model.ResolveDeployOrder(activeProjects, registry)
	if err != nil {
		return nil, err
	}
	if len(waves) == 0 {
		return nil, fmt.Errorf("no deployable projects found - mark projects as Deployable in config to include them")
	}

	transcript := []LogEntry{
		{Text: fmt.Sprintf("📛 Namespace %q ready (simulated Rancher project / Kubernetes namespace)", namespace)},
	}

	for waveIdx, wave := range waves {
		names := make([]string, 0, len(wave))
		for _, p := range wave {
			names = append(names, p.Name)
		}
		transcript = append(transcript, LogEntry{
			Wave: waveIdx,
			Text: fmt.Sprintf("🌊 Wave %d/%d: rolling out %s in parallel", waveIdx+1, len(waves), strings.Join(names, ", ")),
		})
		for _, p := range wave {
			transcript = append(transcript, rolloutSteps(waveIdx, namespace, p)...)
		}
	}

	transcript = append(transcript, LogEntry{Text: "🎉 Workspace rollout complete - every deployable service is healthy."})
	return transcript, nil
}

// rolloutSteps produces the deterministic, ordered set of simulated steps
// Rancher/Kubernetes would perform to bring one deployable project online.
func rolloutSteps(wave int, namespace string, p config.AvailableProject) []LogEntry {
	slug := kebabCase(p.Name)
	steps := []string{
		fmt.Sprintf("🔧 [%s] Rendering Deployment/Service manifests", p.Name),
		fmt.Sprintf("📦 [%s] kubectl apply -n %s -f %s-deployment.yaml", p.Name, namespace, slug),
		fmt.Sprintf("🚀 [%s] Rancher scheduler placed workload onto the cluster", p.Name),
		fmt.Sprintf("⏳ [%s] Rollout status: 1/1 pods ready", p.Name),
		fmt.Sprintf("🌐 [%s] Service %s-svc exposed (ClusterIP)", p.Name, slug),
		fmt.Sprintf("✅ [%s] Deployment healthy", p.Name),
	}

	entries := make([]LogEntry, len(steps))
	for i, s := range steps {
		entries[i] = LogEntry{Wave: wave, Project: p.Name, Text: s}
	}
	return entries
}

// kebabCase is a small, dependency-free normalizer good enough for turning
// project names into plausible simulated Kubernetes resource names.
func kebabCase(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r == ' ' || r == '_':
			out = append(out, '-')
		case r >= 'A' && r <= 'Z':
			out = append(out, r-'A'+'a')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

// Lines flattens a transcript down to plain text, ready to be streamed into
// the UI one line at a time.
func Lines(transcript []LogEntry) []string {
	lines := make([]string, len(transcript))
	for i, e := range transcript {
		lines[i] = e.Text
	}
	return lines
}
