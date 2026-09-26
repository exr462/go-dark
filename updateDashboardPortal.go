package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateDashboardPortal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "ctrl+g":
		if len(m.state.Config.Projects) > 0 {
			m.state.ViewState = model.StateGitOperationsModal
			m.state.GitOperationStep = model.StepSelectGitProject
			m.state.SelectedGitProject = m.state.SelectedProject
			m.state.SelectedGitCommand = 0
			return m, nil
		}
		return m, nil
	case "d": // 👈 Pressing 'd' over a highlighted row opens its dependency mapper
		if len(m.state.Config.Projects) > 0 && m.state.SelectedProject >= 0 {
			activeProj := m.state.Config.Projects[m.state.SelectedProject]

			// Initialize options excluding self to prevent cyclical dependencies
			m.state.DepScreen = model.NewDependencyScreen(activeProj.Name, m.state.Config.Projects)
			m.state.DepScreen.ActiveProjectIndex = m.state.SelectedProject

			// Flip view state boundary to render configuration modal
			m.state.ViewState = model.StateDependencyConfigModal
			return m, nil
		}
	case "B": // 👈 Capital 'B' triggers the full parallel build pipeline
		m.state.StatusMsg = "🏗️ Initializing parallel build graph..."
		m.state.ViewState = model.StateBuildModal // Optional: switch to a loading/progress view

		// Pass your max concurrency limit (e.g., 4 simultaneous builds)
		return m, m.TriggerPipelineCmd(4)
	case "ctrl+f":
		if len(m.state.Config.Projects) > 0 {
			m.state.ViewState = model.StateFuzzyModal
			m.state.FuzzyMode = model.FuzzyModeFiles
			m.state.FuzzyQueryInput.SetValue("")
			m.state.FuzzyResults = []model.FuzzyResult{}
			m.state.SelectedFuzzy = 0
			m.state.FuzzyQueryInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case "ctrl+d":
		m.state.ViewState = model.StateDockerModal
		m.state.SelectedDockerRow = 0
		return m, m.fetchDockerContainersCmd()

	case "ctrl+j":
		m.state.ViewState = model.StateJDKConfigModal
		m.state.JDKStep = model.StepSelectJDKAction
		m.state.SelectedMenuIndex = 0
		m.state.SelectedJDKIndex = 0
		return m, nil

	case "ctrl+u":
		m.state.ViewState = model.StateMavenConfigModal
		m.state.MavenStep = model.StepSelectMvnAction
		m.state.SelectedMenuIndex = 0
		m.state.SelectedMavenIndex = 0
		return m, nil

	case "ctrl+y":
		m.state.ViewState = model.StateConfigDeckModal
		m.state.SelectedConfigOption = 0
		return m, nil

	case "ctrl+n":
		m.state.ViewState = model.StateAddProjectModal
		m.state.FocusedInput = 3
		m.state.Inputs[3].SetValue("")
		m.state.Inputs[4].SetValue("")
		m.state.Inputs[5].SetValue("")
		m.state.Inputs[6].SetValue("")
		m.state.Inputs[3].Focus()
		return m, textinput.Blink

	case "ctrl+b":
		if len(m.state.Config.Projects) > 0 {
			m.state.ViewState = model.StateBuildModal
			m.state.SelectedBuildOption = 0
			m.state.BuildLogs = []string{"Console engine ready. Select command step to initialize stream..."}
			m.state.IsBuilding = false
			return m, nil
		}
		return m, nil

	case "ctrl+s":
		m.state.ViewState = model.StateSessionLogsModal
		m.state.ViewingSessionID = 0
		return m, nil

	case "?", "h":
		m.state.ViewState = model.StateHelpModal
		return m, nil

	case "tab":
		m.state.ActiveFocus = model.FocusArea((int(m.state.ActiveFocus) + 1) % 3)
		return m, nil

	case "up", "k":
		if m.state.ActiveFocus == model.FocusProjects && m.state.SelectedProject > 0 {
			m.state.SelectedProject--
			m.refreshRightPaneFromSelectedProject()
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.state.ActiveFocus == model.FocusTree && m.state.SelectedFile > 0 {
			m.state.SelectedFile--
			cmds = append(cmds, m.readFileContentCmd())
		}

	case "down", "j":
		if m.state.ActiveFocus == model.FocusProjects && m.state.SelectedProject < len(m.state.Config.Projects)-1 {
			m.state.SelectedProject++
			m.refreshRightPaneFromSelectedProject()
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.state.ActiveFocus == model.FocusTree && m.state.SelectedFile < len(m.state.TreeNodes)-1 {
			m.state.SelectedFile++
			cmds = append(cmds, m.readFileContentCmd())
		}

	case "enter", "right", "l":
		if m.state.ActiveFocus == model.FocusMenu {
			cmds = append(cmds, m.executeActiveMenuAction())
		} else if m.state.ActiveFocus == model.FocusTree && len(m.state.TreeNodes) > 0 {
			idx := m.state.SelectedFile
			if idx >= 0 && idx < len(m.state.TreeNodes) {
				if m.state.TreeNodes[idx].IsDir {
					m.state.TreeNodes[idx].IsExpanded = !m.state.TreeNodes[idx].IsExpanded
					m.rebuildActiveTree()
				} else {
					cmds = append(cmds, m.readFileContentCmd())
				}
			}
		}

	case "left", "backspace":
		if m.state.ActiveFocus == model.FocusTree && len(m.state.TreeNodes) > 0 {
			idx := m.state.SelectedFile
			if idx >= 0 && idx < len(m.state.TreeNodes) {
				if m.state.TreeNodes[idx].IsDir && m.state.TreeNodes[idx].IsExpanded {
					m.state.TreeNodes[idx].IsExpanded = false
					m.rebuildActiveTree()
				}
			}
		}

	case "ctrl+e":
		if len(m.state.TreeNodes) == 0 || m.state.SelectedFile < 0 || m.state.SelectedFile >= len(m.state.TreeNodes) {
			return m, nil
		}
		selectedNode := m.state.TreeNodes[m.state.SelectedFile]
		if selectedNode.IsDir {
			return m, nil
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nvim"
		}
		c := exec.Command(editor, selectedNode.FullPath)
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return model.FileLoadMsg("sync")
		})
	}

	return m, tea.Batch(cmds...)
}

func (m *appModel) updateConfigDeckModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil

	case "up", "k":
		if m.state.SelectedConfigOption > 0 {
			m.state.SelectedConfigOption--
		}
		return m, nil

	case "down", "j":
		if m.state.SelectedConfigOption < 3 {
			m.state.SelectedConfigOption++
		}
		return m, nil

	case "enter":
		switch m.state.SelectedConfigOption {
		case 0:
			m.state.ViewState = model.StateInstaller
			m.state.InstallerStep = model.StepSetGlobalPrefs
			m.state.FocusedInput = 0
			m.state.Inputs[0].SetValue(m.state.Config.BasePath)
			m.state.Inputs[1].SetValue(m.state.Config.GitUsername)
			m.state.Inputs[2].SetValue(m.state.Config.GitEmail)
			m.state.Inputs[0].Focus()
			return m, textinput.Blink

		case 1:
			m.state.ViewState = model.StateJDKConfigModal
			m.state.JDKStep = model.StepSelectJDKAction
			m.state.SelectedMenuIndex = 0
			return m, nil

		case 2:
			m.state.ViewState = model.StateMavenConfigModal
			m.state.MavenStep = model.StepSelectMvnAction
			m.state.SelectedMenuIndex = 0
			return m, nil

		case 3:
			m.state.ViewState = model.StateAddProjectModal
			m.state.FocusedInput = 3
			m.state.Inputs[3].SetValue("")
			m.state.Inputs[4].SetValue("")
			m.state.Inputs[5].SetValue("")
			m.state.Inputs[6].SetValue("")
			m.state.Inputs[3].Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m *appModel) updateFuzzyModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil

	case "ctrl+t":
		if m.state.FuzzyMode == model.FuzzyModeFiles {
			m.state.FuzzyMode = model.FuzzyModeContent
		} else {
			m.state.FuzzyMode = model.FuzzyModeFiles
		}
		m.state.SelectedFuzzy = 0
		m.runFuzzySearchEngine()
		m.syncFuzzyPreviewPane()
		return m, nil

	case "up", "k":
		if m.state.SelectedFuzzy > 0 {
			m.state.SelectedFuzzy--
			m.syncFuzzyPreviewPane()
		}
		return m, nil

	case "down", "j":
		if m.state.SelectedFuzzy < len(m.state.FuzzyResults)-1 {
			m.state.SelectedFuzzy++
			m.syncFuzzyPreviewPane()
		}
		return m, nil

	case "enter":
		if len(m.state.FuzzyResults) > 0 && m.state.SelectedFuzzy < len(m.state.FuzzyResults) {
			target := m.state.FuzzyResults[m.state.SelectedFuzzy]
			m.state.ViewState = model.StateDashboard

			for idx, node := range m.state.TreeNodes {
				if node.FullPath == target.FullPath {
					m.state.SelectedFile = idx
					m.state.ActiveFocus = model.FocusTree
					break
				}
			}
			return m, m.readFileContentCmd()
		}
		m.state.ViewState = model.StateDashboard
		return m, nil
	}

	var cmd tea.Cmd
	oldVal := m.state.FuzzyQueryInput.Value()
	var genericMsg tea.Msg = msg
	m.state.FuzzyQueryInput, cmd = m.state.FuzzyQueryInput.Update(genericMsg)

	if m.state.FuzzyQueryInput.Value() != oldVal {
		m.state.SelectedFuzzy = 0
		m.runFuzzySearchEngine()
		m.syncFuzzyPreviewPane()
	}

	return m, cmd
}

func (m *appModel) syncFuzzyPreviewPane() {
	if len(m.state.FuzzyResults) == 0 || m.state.SelectedFuzzy >= len(m.state.FuzzyResults) {
		m.state.FuzzyViewer.SetContent("No file selected for preview.")
		return
	}

	res := m.state.FuzzyResults[m.state.SelectedFuzzy]
	ext := strings.ToLower(filepath.Ext(res.FileName))
	isTextFile := ext == ".go" || ext == ".kt" || ext == ".java" || ext == ".xml" || ext == ".json" ||
		ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".jsx" || ext == ".tsx" || ext == ".ts" || ext == ".txt" ||
		ext == ".md" || ext == ".sh" || ext == ".sql" || ext == ".mod" || ext == ".sum" || res.FileName == "Dockerfile" || res.FileName == "pom.xml"

	if !isTextFile {
		notice := fmt.Sprintf("\n  📦 [BINARY ARTIFACT] Previews Blocked\n\n  File: %s\n\n  Fuzzy preview is disabled for compiled binary artifacts.", res.FileName)
		m.state.FuzzyViewer.SetContent(lipgloss.NewStyle().Foreground(color.Overlay0).Italic(true).Render(notice))
		return
	}

	data, err := os.ReadFile(res.FullPath)
	if err != nil {
		m.state.FuzzyViewer.SetContent(fmt.Sprintf("❌ Error opening preview: %v", err))
		return
	}

	m.state.FuzzyViewer.SetContent(string(data))

	if m.state.FuzzyMode == model.FuzzyModeContent && res.LineNum > 0 {
		m.state.FuzzyViewer.GotoTop()
		for i := 0; i < res.LineNum-3 && i < m.state.FuzzyViewer.Height; i++ {
			m.state.FuzzyViewer.LineDown(1)
		}
	} else {
		m.state.FuzzyViewer.GotoTop()
	}
}

func (m *appModel) runFuzzySearchEngine() {
	query := strings.ToLower(strings.TrimSpace(m.state.FuzzyQueryInput.Value()))
	m.state.FuzzyResults = []model.FuzzyResult{}
	if query == "" {
		return
	}

	if len(m.state.Config.Projects) == 0 || m.state.SelectedProject >= len(m.state.Config.Projects) {
		return
	}

	proj := m.state.Config.Projects[m.state.SelectedProject]
	rootPath := proj.Path

	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return
	}

	if m.state.FuzzyMode == model.FuzzyModeFiles {
		var traverse func(string)
		traverse = func(p string) {
			files, err := os.ReadDir(p)
			if err != nil {
				return
			}
			for _, f := range files {
				name := f.Name()
				if f.IsDir() && (name == "target" || name == "build" || name == "node_modules" || name == "bin" || name == ".git") {
					continue
				}
				if strings.HasPrefix(name, ".") && name != ".gitignore" {
					continue
				}

				fullP := filepath.Join(p, name)
				ext := strings.ToLower(filepath.Ext(name))

				if ext == ".jar" || ext == ".war" || ext == ".zip" || ext == ".class" || ext == ".exe" || ext == ".png" || ext == ".jpg" {
					continue
				}

				if strings.Contains(strings.ToLower(name), query) {
					m.state.FuzzyResults = append(m.state.FuzzyResults, model.FuzzyResult{
						FileName: name,
						FullPath: fullP,
					})
					if len(m.state.FuzzyResults) > 100 {
						return
					}
				}
				if f.IsDir() {
					traverse(fullP)
				}
			}
		}
		traverse(rootPath)
	} else {
		var deepScan func(string)
		deepScan = func(p string) {
			files, err := os.ReadDir(p)
			if err != nil {
				return
			}
			for _, f := range files {
				name := f.Name()
				if f.IsDir() && (name == "target" || name == "build" || name == "node_modules" || name == "bin" || name == ".git") {
					continue
				}
				if strings.HasPrefix(name, ".") && name != ".gitignore" {
					continue
				}

				fullP := filepath.Join(p, name)

				if f.IsDir() {
					deepScan(fullP)
				} else {
					ext := strings.ToLower(filepath.Ext(name))
					if ext == ".xml" || ext == ".go" || ext == ".json" || ext == ".java" || ext == ".txt" || ext == ".md" || ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".kt" || name == "Dockerfile" {
						file, err := os.Open(fullP)
						if err != nil {
							continue
						}

						scanner := bufio.NewScanner(file)
						lineCount := 0
						for scanner.Scan() {
							lineCount++
							txt := scanner.Text()
							if strings.Contains(strings.ToLower(txt), query) {
								m.state.FuzzyResults = append(m.state.FuzzyResults, model.FuzzyResult{
									FileName: name,
									FullPath: fullP,
									LineNum:  lineCount,
									Snippet:  strings.TrimSpace(txt),
								})
								if len(m.state.FuzzyResults) > 100 {
									file.Close()
									return
								}
							}
						}
						file.Close()
					}
				}
			}
		}
		deepScan(rootPath)
	}
}

func (m *appModel) updateBuildModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.state.Config.Projects) == 0 {
		m.state.ViewState = model.StateDashboard
		return m, nil
	}

	targetProj := m.state.Config.Projects[m.state.SelectedProject]
	var currentSessionID int
	for id, sess := range m.state.Sessions {
		if sess.ProjectName == targetProj.Name && sess.IsRunning {
			currentSessionID = id
			break
		}
	}

	if currentSessionID != 0 {
		if msg.String() == "esc" {
			m.state.ViewState = model.StateDashboard
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	case "left", "h":
		if m.state.SelectedBuildOption > 0 {
			m.state.SelectedBuildOption--
		}
	case "right", "l":
		if m.state.SelectedBuildOption < len(m.state.BuildOptions)-1 {
			m.state.SelectedBuildOption++
		}
	case "enter":
		m.state.IsBuilding = true
		chosenOpt := m.state.BuildOptions[m.state.SelectedBuildOption]
		return m, m.spawnBackgroundSession(targetProj, chosenOpt)
	}
	return m, nil
}

func (m *appModel) updateMvnModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state.MavenStep {
	case model.StepSelectMvnAction:
		switch msg.String() {
		case "esc":
			m.state.ViewState = model.StateConfigDeckModal
			return m, nil
		case "up", "k":
			if m.state.SelectedMenuIndex > 0 {
				m.state.SelectedMenuIndex--
			}
		case "down", "j":
			if m.state.SelectedMenuIndex < 1 {
				m.state.SelectedMenuIndex++
			}
		case "enter":
			if m.state.SelectedMenuIndex == 0 {
				m.state.MavenStep = model.StepAddNewMvnVersion
				m.state.FocusedInput = 9
				m.state.Inputs[9].SetValue("")
				m.state.Inputs[10].SetValue("")
				m.state.Inputs[9].Focus()
			} else {
				m.state.MavenStep = model.StepAssignMvnToProject
				m.state.SelectedMavenIndex = 0
			}
		}
	case model.StepAddNewMvnVersion:
		switch msg.String() {
		case "esc":
			m.state.MavenStep = model.StepSelectMvnAction
			return m, nil
		case "tab", "down":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 19 - m.state.FocusedInput
			if m.state.FocusedInput < 9 || m.state.FocusedInput > 10 {
				m.state.FocusedInput = 9
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "shift+tab", "up":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 19 - m.state.FocusedInput
			if m.state.FocusedInput < 9 || m.state.FocusedInput > 10 {
				m.state.FocusedInput = 10
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "enter":
			mVnName := m.state.Inputs[9].Value()
			mVnPath := m.state.Inputs[10].Value()
			if mVnName != "" && mVnPath != "" {
				m.state.Config.Mavens = append(m.state.Config.Mavens, config.Profile{Name: mVnName, Path: mVnPath})
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Added Maven Profile: %s", mVnName)
			}
			m.state.MavenStep = model.StepSelectMvnAction
			return m, nil
		}
		var cmd tea.Cmd
		m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
		return m, cmd
	case model.StepAssignMvnToProject:
		switch msg.String() {
		case "esc":
			m.state.MavenStep = model.StepSelectMvnAction
			return m, nil
		case "up", "k":
			if m.state.SelectedMavenIndex > 0 {
				m.state.SelectedMavenIndex--
			}
		case "down", "j":
			if m.state.SelectedMavenIndex < len(m.state.Config.Mavens)-1 {
				m.state.SelectedMavenIndex++
			}
		case "enter":
			if len(m.state.Config.Mavens) > 0 && len(m.state.Config.Projects) > 0 {
				chosenMvn := m.state.Config.Mavens[m.state.SelectedMavenIndex]
				m.state.Config.Projects[m.state.SelectedProject].MavenName = chosenMvn.Name
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Assigned Maven Profile: %s", chosenMvn.Name)
			}
			m.state.ViewState = model.StateDashboard
			return m, m.updateWorkspaceFiles()
		}
	}
	return m, nil
}

func (m *appModel) updateJDKModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state.JDKStep {
	case model.StepSelectJDKAction:
		switch msg.String() {
		case "esc":
			m.state.ViewState = model.StateConfigDeckModal
			return m, nil
		case "up", "k":
			if m.state.SelectedMenuIndex > 0 {
				m.state.SelectedMenuIndex--
			}
		case "down", "j":
			if m.state.SelectedMenuIndex < 1 {
				m.state.SelectedMenuIndex++
			}
		case "enter":
			if m.state.SelectedMenuIndex == 0 {
				m.state.JDKStep = model.StepAddNewJDKVersion
				m.state.FocusedInput = 7
				m.state.Inputs[7].SetValue("")
				m.state.Inputs[8].SetValue("")
				m.state.Inputs[7].Focus()
			} else {
				m.state.JDKStep = model.StepAssignJDKToProject
				m.state.SelectedJDKIndex = 0
			}
		}
	case model.StepAddNewJDKVersion:
		switch msg.String() {
		case "esc":
			m.state.JDKStep = model.StepSelectJDKAction
			return m, nil
		case "tab", "down":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 15 - m.state.FocusedInput
			if m.state.FocusedInput < 7 || m.state.FocusedInput > 8 {
				m.state.FocusedInput = 7
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "shift+tab", "up":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 15 - m.state.FocusedInput
			if m.state.FocusedInput < 7 || m.state.FocusedInput > 8 {
				m.state.FocusedInput = 8
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "enter":
			jName := m.state.Inputs[7].Value()
			jPath := m.state.Inputs[8].Value()
			if jName != "" && jPath != "" {
				m.state.Config.JDKs = append(m.state.Config.JDKs, config.Profile{Name: jName, Path: jPath})
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Added Java Profile: %s", jName)
			}
			m.state.JDKStep = model.StepSelectJDKAction
			return m, nil
		}
		var cmd tea.Cmd
		m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
		return m, cmd
	case model.StepAssignJDKToProject:
		switch msg.String() {
		case "esc":
			m.state.JDKStep = model.StepSelectJDKAction
			return m, nil
		case "up", "k":
			if m.state.SelectedJDKIndex > 0 {
				m.state.SelectedJDKIndex--
			}
		case "down", "j":
			if m.state.SelectedJDKIndex < len(m.state.Config.JDKs)-1 {
				m.state.SelectedJDKIndex++
			}
		case "enter":
			if len(m.state.Config.JDKs) > 0 && len(m.state.Config.Projects) > 0 {
				chosenJDK := m.state.Config.JDKs[m.state.SelectedJDKIndex]
				m.state.Config.Projects[m.state.SelectedProject].JDKName = chosenJDK.Name
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Assigned JDK Environment: %s", chosenJDK.Name)
			}
			m.state.ViewState = model.StateDashboard
			return m, m.updateWorkspaceFiles()
		}
	}
	return m, nil
}

func (m *appModel) updateDockerModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	case "up", "k":
		if m.state.SelectedDockerRow > 0 {
			m.state.SelectedDockerRow--
		}
	case "down", "j":
		if m.state.SelectedDockerRow < len(m.state.DockerContainers)-1 {
			m.state.SelectedDockerRow++
		}
	case "s", "t", "r":
		if len(m.state.DockerContainers) == 0 || m.state.SelectedDockerRow >= len(m.state.DockerContainers) {
			return m, nil
		}
		target := m.state.DockerContainers[m.state.SelectedDockerRow]
		action := "start"
		if msg.String() == "t" {
			action = "stop"
		}
		if msg.String() == "r" {
			action = "restart"
		}
		return m, tea.Batch(m.runDockerActionCmd(target.ID, action), m.fetchDockerContainersCmd())
	}
	return m, nil
}

func (m *appModel) updateInstaller(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state.GitMissing {
		return m, nil
	}

	if m.state.InstallerStep == model.StepSetGlobalPrefs {
		switch msg.String() {
		case "esc":
			m.state.ViewState = model.StateDashboard
			return m, nil
		case "tab", "down":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = (m.state.FocusedInput + 1) % 3
			m.state.Inputs[m.state.FocusedInput].Focus()
			return m, nil
		case "shift+tab", "up":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput--
			if m.state.FocusedInput < 0 {
				m.state.FocusedInput = 2
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
			return m, nil
		case "enter":
			m.state.Config.BasePath = m.state.Inputs[0].Value()
			m.state.Config.GitUsername = m.state.Inputs[1].Value()
			m.state.Config.GitEmail = m.state.Inputs[2].Value()
			if m.state.Config.BasePath == "" {
				return m, nil
			}
			_ = config.SaveConfig(m.state.Config)
			m.state.InstallerStep = model.StepAddFirstProject
			m.state.FocusedInput = 3
			m.state.Inputs[3].Focus()
			return m, nil
		}
		var cmd tea.Cmd
		m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	case "tab", "down":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput++
		if m.state.FocusedInput > 6 {
			m.state.FocusedInput = 3
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
	case "shift+tab", "up":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput--
		if m.state.FocusedInput < 3 {
			m.state.FocusedInput = 6
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
	case "enter":
		name := m.state.Inputs[3].Value()
		path := m.state.Inputs[4].Value()
		pType := m.state.Inputs[5].Value()
		gitURL := m.state.Inputs[6].Value()
		if name != "" && path != "" {
			m.state.Config.Projects = append(m.state.Config.Projects, config.Project{
				Name:   name,
				Path:   config.ResolvePath(m.state.Config.BasePath, path),
				Type:   strings.ToLower(pType),
				GitURL: gitURL,
			})
		}
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, m.updateWorkspaceFiles()
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}

func (m *appModel) updateModalForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.state.SelectedConfigOption == 3 {
			m.state.ViewState = model.StateConfigDeckModal
		} else {
			m.state.ViewState = model.StateDashboard
		}
		return m, nil
	case "tab", "down":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput++
		if m.state.FocusedInput > 6 {
			m.state.FocusedInput = 3
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
	case "shift+tab", "up":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput--
		if m.state.FocusedInput < 3 {
			m.state.FocusedInput = 6
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
	case "enter":
		name := m.state.Inputs[3].Value()
		path := m.state.Inputs[4].Value()
		pType := m.state.Inputs[5].Value()
		gitURL := m.state.Inputs[6].Value()
		if name == "" || path == "" {
			return m, nil
		}
		newProj := config.Project{
			Name:   name,
			Path:   config.ResolvePath(m.state.Config.BasePath, path),
			Type:   strings.ToLower(pType),
			GitURL: gitURL,
		}
		m.state.Config.Projects = append(m.state.Config.Projects, newProj)
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, func() tea.Msg { return model.ConfigRefreshedMsg(m.state.Config) }
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}

func (m *appModel) updateSessionLogsModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	default:
		if msg.String() >= "1" && msg.String() <= "9" {
			runes := []rune(msg.String())
			if len(runes) > 0 {
				targetID := int(runes[0] - '0')
				m.state.ViewingSessionID = targetID
				return m, nil
			}
		}
	}
	return m, nil
}

func (m *appModel) updateHelpModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "q", "enter":
		m.state.ViewState = model.StateDashboard
		return m, nil
	}
	return m, nil
}

func (m *appModel) executeActiveMenuAction() tea.Cmd {
	if len(m.state.Config.Projects) == 0 {
		return nil
	}
	return func() tea.Msg {
		return model.StatusMsg("Manual workspace isolation completed.")
	}
}
