package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/task"
	"github.com/exr462/go-dark/ui/components"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case task.PipelineTaskStartedMsg:
		m.state.StatusMsg = fmt.Sprintf("🏗️  Building: %s...", msg)
		return m, nil

	case task.PipelineTaskFinishedMsg:
		if msg.Err != nil {
			m.state.StatusMsg = fmt.Sprintf("❌ Build Error on component: %s", msg.ProjectName)
		} else {
			m.state.StatusMsg = fmt.Sprintf("✅ Component complete: %s", msg.ProjectName)
		}
		return m, nil

	case task.PipelineCompleteMsg:
		if msg.Success {
			m.state.StatusMsg = "🎉 All workspace modules built successfully!"
		} else {
			m.state.StatusMsg = "❌ Pipeline compilation aborted due to build errors."
		}
		m.state.ViewState = model.StateDashboard
		return m, m.updateWorkspaceFiles()

	case config.GitStatusLoadedMsg:
		m.state.GitStatusOutput = string(msg)
		if m.state.GitStatusOutput == "" {
			m.state.GitStatusOutput = "✨ Working tree clean."
		}
		return m, nil

	case config.WorkspaceRefreshedMsg:
		m.state.Files = msg.Files
		if m.state.SelectedProject >= 0 && m.state.SelectedProject < len(m.state.Config.Projects) {
			proj := m.state.Config.Projects[m.state.SelectedProject]
			m.state.TreeNodes = []model.FileNode{}
			if _, err := os.Stat(proj.Path); err == nil {
				m.buildTreeNodes(proj.Path, 0)
			}
		}
		return m, func() tea.Msg { return model.FileLoadMsg("sync") }

	case config.GitStatusErrorMsg:
		m.state.GitStatusOutput = fmt.Sprintf("❌ Error: %v", msg)
		return m, nil

	case config.GitBranchesLoadedMsg:
		m.state.AvailableBranches = msg
		m.state.SelectedGitBranch = 0
		return m, nil

	case config.GitBranchesErrorMsg:
		m.state.AvailableBranches = []string{"main"}
		m.state.SelectedGitBranch = 0
		m.state.StatusMsg = fmt.Sprintf("❌ Git: %v", msg)
		return m, nil

	case config.GitCheckoutCompleteMsg:
		if msg.Err != nil {
			m.state.StatusMsg = fmt.Sprintf("❌ %v", msg.Err)
		} else {
			m.state.StatusMsg = fmt.Sprintf("✅ %s", strings.TrimSpace(msg.Output))
			// Refresh fetched status across projects
			for i := range m.state.Config.Projects {
				gitDir := filepath.Join(m.state.Config.Projects[i].Path, ".git")
				if _, err := os.Stat(gitDir); err == nil {
					m.state.Config.Projects[i].Fetched = true
				}
			}
			_ = config.SaveConfig(m.state.Config)
		}
		m.state.ViewState = model.StateDashboard
		return m, m.updateWorkspaceFiles()

	case model.DockerTelemetryMsg:
		m.state.DockerTelemetry = model.DockerStats(msg)
		return m, m.pollDockerTelemetryCmd()

	case model.DockerContainersMsg:
		m.state.DockerContainers = []model.DockerContainer(msg)
		return m, nil

	case tea.WindowSizeMsg:
		m.state.WindowWidth = msg.Width
		m.state.WindowHeight = msg.Height
		m.state.FileViewer.Width = (msg.Width / 2) - 4
		m.state.FileViewer.Height = max(msg.Height-8, 5)
		m.state.FuzzyViewer.Width = (msg.Width / 2) - 4
		m.state.FuzzyViewer.Height = max(msg.Height-12, 5)
		return m, nil

	case model.FileLoadMsg:
		if len(m.state.TreeNodes) > 0 {
			if m.state.SelectedFile >= len(m.state.TreeNodes) {
				m.state.SelectedFile = 0
			}
			cmds = append(cmds, m.readFileContentCmd())
		} else {
			m.state.FileViewer.SetContent("Empty or uncloned project repository.")
		}

	case model.ConfigRefreshedMsg:
		m.state.Config = config.Config(msg)
		cmds = append(cmds, m.updateWorkspaceFiles())

	case model.StatusMsg:
		m.state.StatusMsg = string(msg)
		if strings.Contains(m.state.StatusMsg, "successfully completed") {
			for i := range m.state.Config.Projects {
				gitDir := filepath.Join(m.state.Config.Projects[i].Path, ".git")
				if _, err := os.Stat(gitDir); err == nil {
					m.state.Config.Projects[i].Fetched = true
				}
			}
			_ = config.SaveConfig(m.state.Config)
			return m, m.updateWorkspaceFiles()
		}
		return m, nil

	case model.BuildLogLineMsg:
		if sess, exists := m.state.Sessions[msg.SessionID]; exists {
			if msg.Line != "" {
				line := msg.Line
				if regexp.MustCompile(`(?i)\[error\]|fail`).MatchString(line) {
					line = "\x1b[31;1m" + line + "\x1b[0m"
				} else if regexp.MustCompile(`(?i)\[warn`).MatchString(line) {
					line = "\x1b[33;1m" + line + "\x1b[0m"
				} else if regexp.MustCompile(`(?i)\[info\]|success`).MatchString(line) {
					line = "\x1b[32m" + line + "\x1b[0m"
				}

				sess.Logs = append(sess.Logs, line)
				if m.state.ViewState == model.StateBuildModal && m.state.ActiveSessionID == msg.SessionID {
					m.state.BuildLogs = sess.Logs
				}
			}
		}
		return m, func() tea.Msg {
			activeCh, exists := sessionChannels[msg.SessionID]
			if !exists {
				return nil
			}
			line, ok := <-activeCh
			if !ok {
				return model.BuildCompleteMsg{SessionID: msg.SessionID, Err: nil}
			}
			return model.BuildLogLineMsg{SessionID: msg.SessionID, Line: line}
		}

	case model.BuildCompleteMsg:
		if sess, exists := m.state.Sessions[msg.SessionID]; exists {
			sess.IsRunning = false
			sess.Logs = append(sess.Logs, "────────────────────────────────────────────────────────")
			if msg.Err != nil {
				sess.Logs = append(sess.Logs, fmt.Sprintf("❌ PROCESS TERMINATED WITH ERROR: %v", msg.Err))
			} else {
				sess.Logs = append(sess.Logs, "✅ PROCESS LOOP SUCCESSFULLY TERMINATED IN BACKGROUND.")
			}

			if m.state.ViewState == model.StateBuildModal && m.state.ActiveSessionID == msg.SessionID {
				m.state.BuildLogs = sess.Logs
				m.state.IsBuilding = false
			}
		}
		delete(sessionChannels, msg.SessionID)
		return m, nil

	case components.EditFileMsg:
		m.state.ViewState = model.StateDashboard
		if msg.Err != nil {
			m.state.LastError = msg.Err
			m.state.StatusMsg = fmt.Sprintf("❌ Editor: %v", msg.Err)
			return m, nil
		}
		m.state.ActiveCodeBuffer = msg.Content
		return m, nil

	case tea.KeyMsg:
		// Global quit shortcuts
		if msg.String() == "ctrl+c" || msg.String() == "ctrl+q" {
			return m, tea.Quit
		}

		// Route explicitly to active modal handler
		switch m.state.ViewState {
		case model.StateHelpModal:
			return m.updateHelpModal(msg)
		case model.StateInstaller:
			return m.updateInstaller(msg)
		case model.StateAddProjectModal:
			return m.updateModalForm(msg)
		case model.StateGitOperationsModal:
			return m.updateGitOpsModal(msg)
		case model.StateJDKConfigModal:
			return m.updateJDKModal(msg)
		case model.StateMavenConfigModal:
			return m.updateMvnModal(msg)
		case model.StateBuildModal:
			return m.updateBuildModal(msg)
		case model.StateSessionLogsModal:
			return m.updateSessionLogsModal(msg)
		case model.StateFuzzyModal:
			return m.updateFuzzyModal(msg)
		case model.StateConfigDeckModal:
			return m.updateConfigDeckModal(msg)
		case model.StateDockerModal:
			return m.updateDockerModal(msg)
		case model.StateEditorModal:
			return m, nil
		case model.StateDependencyConfigModal:
			// 🔍 SAFE VALIDATION INSIDE TARGET CONTEXT
			projIdx := m.state.DepScreen.ActiveProjectIndex
			if projIdx < 0 || projIdx >= len(m.state.Config.Projects) {
				m.state.ViewState = model.StateDashboard
				return m, nil
			}
			return m.updateDependencyScreen(msg)
		case model.StateDashboard:
			fallthrough
		default:
			return m.updateDashboardPortal(msg)
		}
	}

	// Viewport event dispatching
	if m.state.ViewState == model.StateDashboard {
		var viewCmd tea.Cmd
		m.state.FileViewer, viewCmd = m.state.FileViewer.Update(msg)
		cmds = append(cmds, viewCmd)
	}

	if m.state.ViewState == model.StateFuzzyModal {
		var fuzzyCmd tea.Cmd
		m.state.FuzzyViewer, fuzzyCmd = m.state.FuzzyViewer.Update(msg)
		cmds = append(cmds, fuzzyCmd)
	}

	// Cleaned duplicate fallback block at the bottom
	return m, tea.Batch(cmds...)
}
