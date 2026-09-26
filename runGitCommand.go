package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection ALL
func (m *appModel) runGitCommand(proj config.Project, operation string) tea.Cmd {
	fullPath := proj.Path
	log.Printf("Running git command: %s for project: %s", operation, proj.Name)

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
				return config.GitStatusErrorMsg(fmt.Errorf("status failed: %s (%v)", strings.TrimSpace(string(out)), err))
			}
			return config.GitStatusLoadedMsg(string(out))

		case strings.Contains(op, "reset"):
			cmd = exec.Command("git", "reset", "--hard")
			cmd.Dir = fullPath

		case isCheckout:
			targetRef := "develop"
			isTag := false
			log.Print("IsCheckout")

			if len(m.state.AvailableBranches) > 0 {
				idx := m.state.SelectedGitBranch
				if idx >= 0 && idx < len(m.state.AvailableBranches) {
					rawRef := m.state.AvailableBranches[idx]

					// 🎯 STEP A: Instantly strip hidden line-end breaks (\r or \n) and trailing spaces
					cleanedRef := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(rawRef, "\r", ""), "\n", ""))

					if strings.HasSuffix(cleanedRef, " (tag)") {
						isTag = true
						targetRef = strings.TrimSuffix(cleanedRef, " (tag)")
					} else {
						targetRef = cleanedRef
					}
				}
			}

			// 🎯 STEP B: Sanitize the final variable one last time to prevent argument breaking
			targetRef = strings.TrimSpace(targetRef)

			cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

			var out []byte
			var err error

			if isTag {
				// 🎯 CORRECT SYNTAX: Direct checkout to go to a detached HEAD state on the tag.
				// We drop the combined branch flag formatting string entirely.
				cmd = exec.Command("git", "checkout", targetRef)
				log.Printf("Absolute path: %s [%s]", fullPath, cmd.String())
				cmd.Dir = fullPath
				out, err = cmd.CombinedOutput()

				// Fallback: If it's a completely unindexed remote reference marker, pull down tag references explicitly
				if err != nil {
					fetchCmd := exec.Command("git", "fetch", "origin", "tag", targetRef, "--no-tags")
					fetchCmd.Dir = fullPath
					fetchCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
					_, _ = fetchCmd.CombinedOutput()

					// Final checkout retry
					cmd = exec.Command("git", "checkout", targetRef)
					cmd.Dir = fullPath
					out, err = cmd.CombinedOutput()
				}

				// 📝 DEEP DEBUG LOGGING FOR FAULT TRACKING:
				if f, logErr := os.OpenFile("checkout-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); logErr == nil {
					logHeader := fmt.Sprintf("\n=== TAG CHECKOUT ATTEMPT [%s] ===\n", time.Now().Format("15:04:05"))
					_, _ = f.WriteString(logHeader)
					_, _ = f.WriteString(fmt.Sprintf("Directory: %s\n", fullPath))
					_, _ = f.WriteString(fmt.Sprintf("Target Tag: %s\n", targetRef))
					_, _ = f.WriteString(fmt.Sprintf("Exit Error: %v\n", err))
					_, _ = f.WriteString(fmt.Sprintf("Git Output:\n%s\n", string(out)))
					_, _ = f.WriteString("=====================================\n")
					_ = f.Close()
				}

				return config.GitCheckoutCompleteMsg{
					Output: string(out),
					Err:    err,
				}
			} else {
				// Standard branch checkout lane
				cmd = exec.Command("git", "checkout", targetRef)
				cmd.Dir = fullPath
				out, err = cmd.CombinedOutput()
				return config.GitCheckoutCompleteMsg{
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
			return config.GitCheckoutCompleteMsg{
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
