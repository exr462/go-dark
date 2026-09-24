package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-build/config"
	"github.com/exr462/go-build/model"
	"github.com/exr462/go-build/ui/components"
	"github.com/exr462/go-build/ui/panels"
)

type appModel struct {
	state model.UIState
}

func (m appModel) Init() tea.Cmd {
	if m.state.ViewState == model.StateInstaller {
		return textinput.Blink
	}
	return m.updateWorkspaceFiles()
}

func (m appModel) updateWorkspaceFiles() tea.Cmd {
	return func() tea.Msg {
		if len(m.state.Config.Projects) == 0 {
			return model.FileLoadMsg("")
		}
		if m.state.SelectedProj >= len(m.state.Config.Projects) {
			m.state.SelectedProj = 0
		}
		proj := m.state.Config.Projects[m.state.SelectedProj]
		fullPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)

		entries, err := ioutil.ReadDir(fullPath)
		if err != nil {
			return model.StatusMsg(fmt.Sprintf("Directory missing at target: %s", fullPath))
		}

		var projectFiles []string
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				projectFiles = append(projectFiles, e.Name())
			}
		}

		m.state.Files = projectFiles
		return model.FileLoadMsg("sync")
	}
}

func (m appModel) readFileContentCmd() tea.Cmd {
	return func() tea.Msg {
		if len(m.state.Files) == 0 {
			return model.StatusMsg("No workspace files available.")
		}
		proj := m.state.Config.Projects[m.state.SelectedProj]
		fullProjPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)

		targetFilePath := filepath.Join(fullProjPath, m.state.Files[m.state.SelectedFile])
		data, err := ioutil.ReadFile(targetFilePath)
		if err != nil {
			return model.StatusMsg(fmt.Sprintf("Failed file stream: %v", err))
		}
		m.state.FileViewer.SetContent(string(data))
		return model.StatusMsg(fmt.Sprintf("Inspecting file: %s", m.state.Files[m.state.SelectedFile]))
	}
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.TerminalW = msg.Width
		m.state.TerminalH = msg.Height
		m.state.FileViewer.Width = (msg.Width / 2) - 4
		m.state.FileViewer.Height = msg.Height - 8
		if m.state.FileViewer.Height < 5 {
			m.state.FileViewer.Height = 5
		}

	case model.FileLoadMsg:
		if len(m.state.Files) > 0 {
			if m.state.SelectedFile >= len(m.state.Files) {
				m.state.SelectedFile = 0
			}
			cmds = append(cmds, m.readFileContentCmd())
		} else {
			m.state.FileViewer.SetContent("No files found.")
		}

	case model.ConfigRefreshedMsg:
		m.state.Config = config.Config(msg)
		cmds = append(cmds, m.updateWorkspaceFiles())

	case model.StatusMsg:
		m.state.StatusMsg = string(msg)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Intercept Git menu hotkey global triggers
		if msg.String() == "ctrl+g" && m.state.ViewState == model.StateDashboard {
			if len(m.state.Config.Projects) > 0 {
				m.state.ViewState = model.StateGitOpsModal
				m.state.GitOpsStep = model.StepSelectGitProject
				m.state.SelectedGitProj = m.state.SelectedProj
				m.state.SelectedGitCmd = 0
				return m, nil
			}
		}

		switch m.state.ViewState {
		case model.StateInstaller:
			return m.updateInstaller(msg)
		case model.StateAddProjectModal:
			return m.updateModalForm(msg)
		case model.StateGitOpsModal:
			return m.updateGitOpsModal(msg)
		default:
			return m.updateDashboardPortal(msg)
		}
	}

	var viewCmd tea.Cmd
	m.state.FileViewer, viewCmd = m.state.FileViewer.Update(msg)
	cmds = append(cmds, viewCmd)

	return m, tea.Batch(cmds...)
}

func (m appModel) updateGitOpsModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.state.GitOpsStep == model.StepSelectGitCommand {
			m.state.GitOpsStep = model.StepSelectGitProject
		} else {
			m.state.ViewState = model.StateDashboard
		}
		return m, nil

	case "up", "k":
		if m.state.GitOpsStep == model.StepSelectGitProject {
			if m.state.SelectedGitProj > 0 {
				m.state.SelectedGitProj--
			}
		} else {
			if m.state.SelectedGitCmd > 0 {
				m.state.SelectedGitCmd--
			}
		}

	case "down", "j":
		if m.state.GitOpsStep == model.StepSelectGitProject {
			if m.state.SelectedGitProj < len(m.state.Config.Projects)-1 {
				m.state.SelectedGitProj++
			}
		} else {
			if m.state.SelectedGitCmd < len(m.state.GitCommands)-1 {
				m.state.SelectedGitCmd++
			}
		}

	case "enter":
		if m.state.GitOpsStep == model.StepSelectGitProject {
			m.state.GitOpsStep = model.StepSelectGitCommand
			return m, nil
		}

		cmdToRun := m.state.GitCommands[m.state.SelectedGitCmd]
		proj := m.state.Config.Projects[m.state.SelectedGitProj]

		m.state.ViewState = model.StateDashboard
		m.state.StatusMsg = fmt.Sprintf("Executing task: git %s on %s...", cmdToRun, proj.Name)

		return m, m.runGitCommandCmd(proj, cmdToRun)
	}

	return m, nil
}

func (m appModel) runGitCommandCmd(proj config.Project, operation string) tea.Cmd {
	return func() tea.Msg {
		fullPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)
		var cmd *exec.Cmd

		switch operation {
		case "clone":
			if proj.GitURL == "" {
				return model.StatusMsg("❌ Git Operation Aborted: No remote clone URL found.")
			}
			_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
			cmd = exec.Command("git", "clone", proj.GitURL, fullPath)
		case "fetch":
			cmd = exec.Command("git", "fetch", "--all")
			cmd.Dir = fullPath
		case "pull":
			cmd = exec.Command("git", "pull")
			cmd.Dir = fullPath
		case "checkout (main)":
			cmd = exec.Command("git", "checkout", "main")
			cmd.Dir = fullPath
		}

		if cmd == nil {
			return model.StatusMsg("Unknown automation action variant mapping.")
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			return model.StatusMsg(fmt.Sprintf("❌ Error: %v | Log: %s", err, string(out)))
		}
		return model.StatusMsg(fmt.Sprintf("✅ Successful execution: git %s completed.", operation))
	}
}

func (m appModel) updateInstaller(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state.GitMissing {
		return m, nil
	}

	if m.state.InstallerStep == model.StepSetGlobalPrefs {
		switch msg.String() {
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
			m.state.Config.GitConfig = config.GitConfig{
				GitUsername: m.state.Inputs[1].Value(),
				GitEmail:    m.state.Inputs[2].Value(),
			}

			if m.state.Config.BasePath == "" {
				return m, nil
			}

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
				Path:   path,
				Type:   strings.ToLower(pType),
				GitURL: gitURL,
			})
		}

		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		m.state.StatusMsg = "Setup complete."
		return m, m.updateWorkspaceFiles()
	}

	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}

func (m appModel) updateDashboardPortal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab":
		m.state.ActiveFocus = model.FocusArea((int(m.state.ActiveFocus) + 1) % 3)
	case "ctrl+n":
		m.state.ViewState = model.StateAddProjectModal
		m.state.FocusedInput = 3
		m.state.Inputs[3].SetValue("")
		m.state.Inputs[4].SetValue("")
		m.state.Inputs[5].SetValue("")
		m.state.Inputs[6].SetValue("")
		m.state.Inputs[3].Focus()
		return m, nil
	case "up", "k":
		if m.state.ActiveFocus == model.FocusProjects && m.state.SelectedProj > 0 {
			m.state.SelectedProj--
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.state.ActiveFocus == model.FocusTree && m.state.SelectedFile > 0 {
			m.state.SelectedFile--
			cmds = append(cmds, m.readFileContentCmd())
		}
	case "down", "j":
		if m.state.ActiveFocus == model.FocusProjects && m.state.SelectedProj < len(m.state.Config.Projects)-1 {
			m.state.SelectedProj++
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.state.ActiveFocus == model.FocusTree && m.state.SelectedFile < len(m.state.Files)-1 {
			m.state.SelectedFile++
			cmds = append(cmds, m.readFileContentCmd())
		}
	case "enter":
		if m.state.ActiveFocus == model.FocusMenu {
			cmds = append(cmds, m.executeActiveMenuAction())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m appModel) updateModalForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if name == "" || path == "" {
			m.state.StatusMsg = "❌ Error: Project settings missing fields!"
			return m, nil
		}
		newProj := config.Project{Name: name, Path: path, Type: strings.ToLower(pType), GitURL: gitURL}
		m.state.Config.Projects = append(m.state.Config.Projects, newProj)
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, func() tea.Msg {
			return model.ConfigRefreshedMsg(m.state.Config)
		}
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}
func (m appModel) executeActiveMenuAction() tea.Cmd {
	if len(m.state.Config.Projects) == 0 {
		return nil
	}
	proj := m.state.Config.Projects[m.state.SelectedProj]
	fullProjPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)
	return func() tea.Msg {
		backupDir := fullProjPath + "_backup_target"
		_ = os.RemoveAll(backupDir)
		_ = exec.Command("cp", "-r", fullProjPath, backupDir).Run()
		_ = exec.Command("git", "clean", "-xdf").Run()
		return model.StatusMsg("Workspace isolation operation completed.")
	}
}
func (m appModel) View() string {
	switch m.state.ViewState {
	case model.StateInstaller:
		return components.RenderInstaller(m.state)
	case model.StateAddProjectModal:
		return components.RenderModal(m.state)
	case model.StateGitOpsModal:
		return components.RenderGitOpsModal(m.state)
	default:
		topBar := panels.RenderTopMenu(m.state)
		body := panels.RenderMainBody(m.state)
		footerText := fmt.Sprintf(" [Ctrl+G] Git Control | BasePath: %s | User: %s | Logs: %s", m.state.Config.BasePath, m.state.Config.GitConfig.GitUsername, m.state.StatusMsg)
		footer := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("250")).Width(m.state.TerminalW).Render(footerText)
		return lipgloss.JoinVertical(lipgloss.Left, topBar, body, footer)
	}
}
func main() {
	cfg, isFirstRun := config.LoadConfig()
	_, gitErr := exec.LookPath("git")
	gitMissing := gitErr != nil
	home, _ := os.UserHomeDir()
	inputs := make([]textinput.Model, 7)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Global Workspace Base Path"
	inputs[0].SetValue(filepath.Join(home, "Developer"))
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "e.g. John Doe"
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "e.g. john@example.com"
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "My Application Service"
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "my-service-folder"
	inputs[5] = textinput.New()
	inputs[5].Placeholder = "java"
	inputs[6] = textinput.New()
	inputs[6].Placeholder = "git@github.com:user/repo.git"
	initialState := model.StateDashboard
	if isFirstRun || gitMissing {
		initialState = model.StateInstaller
		if !gitMissing {
			inputs[0].Focus()
		}
	}
	m := appModel{state: model.UIState{Config: cfg, ViewState: initialState, InstallerStep: model.StepSetGlobalPrefs, ActiveFocus: model.FocusProjects, FileViewer: viewport.New(30, 20), Inputs: inputs, GitMissing: gitMissing, GitCommands: []string{"fetch", "pull", "clone", "checkout (main)"}}}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		os.Exit(1)
	}
}
