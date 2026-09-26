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
func (m *appModel) executeGitCheckoutCmd(proj config.Project, branch string) tea.Cmd {
	return func() tea.Msg {
		gitDir := filepath.Join(proj.Path, ".git")
		isNotCloned := false
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			isNotCloned = true
		}

		if isNotCloned {
			if proj.GitURL == "" {
				return config.GitCheckoutCompleteMsg{
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
					return config.GitCheckoutCompleteMsg{
						Output: string(out2),
						Err:    fmt.Errorf("clone failed: %s (%v)", strings.TrimSpace(string(out2)), err2),
					}
				}
				coCmd := exec.Command("git", "checkout", branch)
				coCmd.Dir = proj.Path
				coCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
				outCo, _ := coCmd.CombinedOutput()
				return config.GitCheckoutCompleteMsg{
					Output: fmt.Sprintf("Cloned & checked out: %s\n%s", branch, string(outCo)),
					Err:    nil,
				}
			}

			return config.GitCheckoutCompleteMsg{
				Output: fmt.Sprintf("Cloned and checked out %s (%s)", proj.Name, branch),
				Err:    nil,
			}
		}

		// Local repo exists: normal checkout
		cmd := exec.Command("git", "checkout", branch)
		cmd.Dir = proj.Path
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		output, err := cmd.CombinedOutput()
		return config.GitCheckoutCompleteMsg{
			Output: string(output),
			Err:    err,
		}
	}
}
