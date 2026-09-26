package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

type GitStatusLoadedMsg string
type GitStatusErrorMsg error
type GitBranchesLoadedMsg []string
type GitBranchesErrorMsg error
type GitCheckoutCompleteMsg struct {
	Output string
	Err    error
}

func (m *appModel) loadGitBranchesCmd() tea.Cmd {
	if len(m.state.Config.Projects) == 0 {
		type GitStatusLoadedMsg string
		type GitStatusErrorMsg error
		type GitBranchesLoadedMsg []string
		type GitBranchesErrorMsg error
		type GitCheckoutCompleteMsg struct {
			Output string
			Err    error
		}
		return func() tea.Msg { return GitBranchesLoadedMsg{"develop"} }
	}

	idx := m.state.SelectedGitProject
	if idx < 0 || idx >= len(m.state.Config.Projects) {
		return func() tea.Msg { return GitBranchesLoadedMsg{"develop"} }
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
					return GitBranchesLoadedMsg([]string{"develop"})
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
							name = tagName + " (tag)"
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
				return GitBranchesLoadedMsg(items)
			}
		}
		return func() tea.Msg { return GitBranchesLoadedMsg([]string{"develop"}) }
	}

	// Local git directory exists: query branches AND tags from local repo
	return func() tea.Msg {
		// 1. Get Branches
		branchCmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
		branchCmd.Dir = dir
		branchCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		branchOutput, err := branchCmd.CombinedOutput()
		if err != nil {
			return GitBranchesErrorMsg(fmt.Errorf("git branch failed: %s (%v)", strings.TrimSpace(string(branchOutput)), err))
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
				if tagCount >= 10 {
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

		return GitBranchesLoadedMsg(items)
	}
}

func (m *appModel) executeGitCheckoutCmd(proj config.Project, branch string) tea.Cmd {
	return func() tea.Msg {
		gitDir := filepath.Join(proj.Path, ".git")
		isNotCloned := false
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			isNotCloned = true
		}

		if isNotCloned {
			if proj.GitURL == "" {
				return GitCheckoutCompleteMsg{
					Err: fmt.Errorf("project not found locally and no git_url specified"),
				}
			}

			// Ensure parent directory exists
			_ = os.MkdirAll(filepath.Dir(proj.Path), 0755)

			// Clone repo and checkout target branch
			cloneCmd := exec.Command("git", "clone", "--branch", branch, proj.GitURL, proj.Path)
			cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
			_, err := cloneCmd.CombinedOutput()
			if err != nil {
				// Fallback: standard clone then checkout
				cloneCmd2 := exec.Command("git", "clone", proj.GitURL, proj.Path)
				cloneCmd2.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
				out2, err2 := cloneCmd2.CombinedOutput()
				if err2 != nil {
					return GitCheckoutCompleteMsg{
						Output: string(out2),
						Err:    fmt.Errorf("clone failed: %s (%v)", strings.TrimSpace(string(out2)), err2),
					}
				}
				coCmd := exec.Command("git", "checkout", branch)
				coCmd.Dir = proj.Path
				coCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
				outCo, _ := coCmd.CombinedOutput()
				return GitCheckoutCompleteMsg{
					Output: fmt.Sprintf("Cloned & checked out: %s\n%s", branch, string(outCo)),
					Err:    nil,
				}
			}

			return GitCheckoutCompleteMsg{
				Output: fmt.Sprintf("Cloned and checked out %s (%s)", proj.Name, branch),
				Err:    nil,
			}
		}

		// Local repo exists: normal checkout
		cmd := exec.Command("git", "checkout", branch)
		cmd.Dir = proj.Path
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		output, err := cmd.CombinedOutput()
		return GitCheckoutCompleteMsg{
			Output: string(output),
			Err:    err,
		}
	}
}

func (m *appModel) runGitCommand(proj config.Project, operation string) tea.Cmd {
	fullPath := proj.Path

	return func() tea.Msg {
		var cmd *exec.Cmd
		op := strings.ToLower(strings.TrimSpace(operation))
		isCheckout := strings.Contains(op, "checkout")

		switch {
		case strings.Contains(op, "clone"):
			if proj.GitURL == "" {
				return model.StatusMsg("❌ Git Operation Aborted: No URL found.")
			}
			_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
			cmd = exec.Command("git", "clone", proj.GitURL, fullPath)

		case strings.Contains(op, "fetch"):
			cmd = exec.Command("git", "fetch", "--all")
			cmd.Dir = fullPath

		case strings.Contains(op, "pull"):
			cmd = exec.Command("git", "pull", "origin", "--no-edit")
			cmd.Dir = fullPath

		case strings.Contains(op, "status"):
			cmd = exec.Command("git", "status", "-s")
			cmd.Dir = fullPath
			cmd.Env = os.Environ()
			out, err := cmd.CombinedOutput()
			if err != nil {
				return GitStatusErrorMsg(fmt.Errorf("status failed: %s (%v)", strings.TrimSpace(string(out)), err))
			}
			return GitStatusLoadedMsg(string(out))

		case strings.Contains(op, "reset"):
			cmd = exec.Command("git", "reset", "--hard")
			cmd.Dir = fullPath

		case isCheckout:
			targetRef := "develop"
			isTag := false

			if len(m.state.AvailableBranches) > 0 {
				idx := m.state.SelectedGitBranch
				if idx >= 0 && idx < len(m.state.AvailableBranches) {
					rawRef := m.state.AvailableBranches[idx]
					if strings.HasSuffix(rawRef, " (tag)") {
						isTag = true
						targetRef = strings.TrimSuffix(rawRef, " (tag)")
					} else {
						targetRef = rawRef
					}
				}
			}

			cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

			if isTag {
				// 1. Try checking out the tag name directly (Standard approach)
				cmd = exec.Command("git", "checkout", targetRef)
				cmd.Dir = fullPath
				out, err := cmd.CombinedOutput()

				// 2. If it fails (Pathspec error / missing tag locally), force a deep fetch of all tags
				if err != nil {
					// --tags pulls all references down, overriding restrictive local refspecs
					fetchCmd := exec.Command("git", "fetch", "origin", "--tags", "--force")
					fetchCmd.Dir = fullPath
					fetchCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
					_, _ = fetchCmd.CombinedOutput()

					// Retry plain checkout without the "tags/" prefix string
					cmd = exec.Command("git", "checkout", targetRef)
					cmd.Dir = fullPath
					out, err = cmd.CombinedOutput()
				}

				return GitCheckoutCompleteMsg{
					Output: string(out),
					Err:    err,
				}
			} else {
				// Standard branch checkout path
				cmd = exec.Command("git", "checkout", targetRef)
				cmd.Dir = fullPath
				out, err := cmd.CombinedOutput()
				return GitCheckoutCompleteMsg{
					Output: string(out),
					Err:    err,
				}
			}

		default:
			return model.StatusMsg(fmt.Sprintf("❌ Unknown operation: %s", operation))
		}

		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.CombinedOutput()

		// 🎯 Handle Checkout Message Routing
		if isCheckout {
			// Assuming GitCheckoutCompleteMsg is a struct like: struct { Output string; Err error }
			// Adjust the fields below if your struct definition uses different field names.
			return GitCheckoutCompleteMsg{
				Output: string(out),
				Err:    err,
			}
		}

		// Fallback for all other standard commands (clone, pull, fetch, reset)
		if err != nil {
			gitLog := strings.ReplaceAll(strings.TrimSpace(string(out)), "\n", " | ")
			if gitLog == "" {
				gitLog = err.Error()
			}
			return model.StatusMsg(fmt.Sprintf("❌ git %s failed: %s", operation, gitLog))
		}

		return model.StatusMsg(fmt.Sprintf("✅ git %s successfully completed for %s.", operation, proj.Name))
	}
}

func (m *appModel) updateGitOpsModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.state.GitOperationStep == model.StepSelectGitBranch {
			m.state.GitOperationStep = model.StepSelectGitCommand
		} else if m.state.GitOperationStep == model.StepSelectGitCommand {
			m.state.GitOperationStep = model.StepSelectGitProject
		} else if m.state.GitOperationStep == 3 {
			m.state.GitOperationStep = model.StepSelectGitCommand
		} else {
			m.state.ViewState = model.StateDashboard
		}
		return m, nil

	case "up", "k":
		switch m.state.GitOperationStep {
		case model.StepSelectGitProject:
			if m.state.SelectedGitProject > 0 {
				m.state.SelectedGitProject--
			}
		case model.StepSelectGitCommand:
			if m.state.SelectedGitCommand > 0 {
				m.state.SelectedGitCommand--
			}
		case model.StepSelectGitBranch:
			if m.state.SelectedGitBranch > 0 {
				m.state.SelectedGitBranch--
			}
		}
		return m, nil

	case "down", "j":
		switch m.state.GitOperationStep {
		case model.StepSelectGitProject:
			if m.state.SelectedGitProject < len(m.state.Config.Projects)-1 {
				m.state.SelectedGitProject++
			}
		case model.StepSelectGitCommand:
			if m.state.SelectedGitCommand < len(m.state.GitCommands)-1 {
				m.state.SelectedGitCommand++
			}
		case model.StepSelectGitBranch:
			if m.state.SelectedGitBranch < len(m.state.AvailableBranches)-1 {
				m.state.SelectedGitBranch++
			}
		}
		return m, nil

	case "enter":
		switch m.state.GitOperationStep {
		case model.StepSelectGitProject:
			m.state.GitOperationStep = model.StepSelectGitCommand
			m.state.SelectedGitCommand = 0
			return m, m.loadGitBranchesCmd()

		case model.StepSelectGitCommand:
			chosenCmd := m.state.GitCommands[m.state.SelectedGitCommand]
			targetProj := m.state.Config.Projects[m.state.SelectedGitProject]

			if strings.HasPrefix(chosenCmd, "checkout") {
				m.state.GitOperationStep = model.StepSelectGitBranch
				m.state.AvailableBranches = []string{}
				m.state.SelectedGitBranch = 0
				return m, m.loadGitBranchesCmd()
			}

			if chosenCmd == "status" {
				if !targetProj.Fetched {
					m.state.StatusMsg = fmt.Sprintf("⚠️ %s is not cloned yet.", targetProj.Name)
					return m, nil
				}
				m.state.GitOperationStep = 3
				m.state.GitStatusOutput = "⏳ Querying workspace parameters..."
				return m, m.runGitCommand(targetProj, "status")
			}

			if chosenCmd == "reset" {
				chosenCmd = "reset --hard"
			}

			if !targetProj.Fetched && (chosenCmd == "pull" || chosenCmd == "fetch" || strings.HasPrefix(chosenCmd, "reset")) {
				m.state.StatusMsg = fmt.Sprintf("⚠️ %s is not cloned yet. Use checkout or clone first.", targetProj.Name)
				return m, nil
			}

			m.state.ViewState = model.StateDashboard
			m.state.StatusMsg = fmt.Sprintf("🔄 Executing git %s on %s...", chosenCmd, targetProj.Name)
			return m, m.runGitCommand(targetProj, chosenCmd)

		case model.StepSelectGitBranch:
			if len(m.state.AvailableBranches) > 0 && m.state.SelectedGitBranch < len(m.state.AvailableBranches) {
				targetProj := m.state.Config.Projects[m.state.SelectedGitProject]
				targetBranch := m.state.AvailableBranches[m.state.SelectedGitBranch]
				m.state.StatusMsg = fmt.Sprintf("🔄 Checking out %s on %s...", targetBranch, targetProj.Name)
				return m, m.executeGitCheckoutCmd(targetProj, targetBranch)
			}
			return m, nil
		}
	}
	return m, nil
}
