package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) loadGitBranchesCmd() tea.Cmd {
	if len(m.state.Config.Projects) == 0 {
		return func() tea.Msg { return config.GitBranchesLoadedMsg{"develop"} }
	}

	idx := m.state.SelectedGitProject
	if idx < 0 || idx >= len(m.state.Config.Projects) {
		return func() tea.Msg { return config.GitBranchesLoadedMsg{"develop"} }
	}

	proj := m.state.Config.Projects[idx]
	dir := proj.Path
	if dir == "" {
		dir = "."
	}

	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Project is not yet cloned locally.
		// Query remote repository branches AND tags if GitURL is available.
		if proj.GitURL != "" {
			return func() tea.Msg {
				// Query both heads (branches) and tags from remote
				cmd := exec.Command("git", "ls-remote", "--refs", proj.GitURL)
				cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return config.GitBranchesLoadedMsg([]string{"develop"})
				}

				var items []string
				seen := make(map[string]bool)
				lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")

				for _, line := range lines {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						ref := fields[1]
						var name string

						if strings.HasPrefix(ref, "refs/heads/") {
							name = strings.TrimPrefix(ref, "refs/heads/")
						} else if strings.HasPrefix(ref, "refs/tags/") {
							// Strip the tag prefix and optionally add a visual indicator
							tagName := strings.TrimPrefix(ref, "refs/tags/")
							// Skip dereferenced peeled tags (e.g., v1.0.0^{})
							if strings.HasSuffix(tagName, "^{}") {
								continue
							}
							name = tagName
						}

						if name != "" && !seen[name] {
							seen[name] = true
							items = append(items, name)
						}
					}
				}
				if len(items) == 0 {
					items = []string{"develop"}
				}
				return config.GitBranchesLoadedMsg(items)
			}
		}
		return func() tea.Msg { return config.GitBranchesLoadedMsg([]string{"develop"}) }
	}

	// Local git directory exists: query branches AND tags from local repo
	return func() tea.Msg {
		// 1. Get Branches
		branchCmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
		branchCmd.Dir = dir
		branchCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		branchOutput, err := branchCmd.CombinedOutput()
		if err != nil {
			return config.GitBranchesErrorMsg(fmt.Errorf("git branch failed: %s (%v)", strings.TrimSpace(string(branchOutput)), err))
		}

		var items []string
		seen := make(map[string]bool)

		lines := strings.SplitSeq(strings.ReplaceAll(string(branchOutput), "\r\n", "\n"), "\n")
		for line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.Contains(trimmed, "HEAD") {
				continue
			}

			cleaned := trimmed
			if strings.HasPrefix(cleaned, "remotes/origin/") {
				cleaned = cleaned[len("remotes/origin/"):]
			} else if strings.HasPrefix(cleaned, "origin/") {
				cleaned = cleaned[len("origin/"):]
			}

			if cleaned != "" && !seen[cleaned] {
				seen[cleaned] = true
				items = append(items, cleaned)
			}
		}

		// 2. Get Local Tags
		tagCmd := exec.Command("git", "tag", "--sort=-v:refname")
		tagCmd.Dir = dir
		tagCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		if tagOutput, err := tagCmd.CombinedOutput(); err == nil {
			tagLines := strings.Split(strings.ReplaceAll(string(tagOutput), "\r\n", "\n"), "\n")
			tagCount := 0

			for _, tagLine := range tagLines {
				// Stop parsing once we hit our cap limit to keep the UI clean
				if tagCount >= m.state.Config.MaxListTag {
					break
				}

				trimmedTag := strings.TrimSpace(tagLine)
				if trimmedTag != "" {
					displayTag := trimmedTag + " (tag)"
					if !seen[displayTag] {
						seen[displayTag] = true
						items = append(items, displayTag)
						tagCount++
					}
				}
			}
		}

		if len(items) == 0 {
			items = append(items, "develop")
		}

		return config.GitBranchesLoadedMsg(items)
	}
}
