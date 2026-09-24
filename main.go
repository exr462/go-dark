package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/components"
	"github.com/exr462/go-dark/ui/panels"
)

var sessionChannels = make(map[int]chan string)

type appModel struct {
	state model.UIState
}

func (m *appModel) Init() tea.Cmd {
	if m.state.ViewState == model.StateInstaller {
		return textinput.Blink
	}
	return m.updateWorkspaceFiles()
}

func (m *appModel) buildTreeNodes(currentPath string, depth int) {
	entries, err := ioutil.ReadDir(currentPath)
	if err != nil {
		return
	}

	for _, e := range entries {
		name := e.Name()
		// 1. FILTER: Ignore hidden system dotfiles
		if strings.HasPrefix(name, ".") && name != ".gitignore" {
			continue
		}

		// 2. FILTER: Block standard compiled asset artifact output folders completely
		// This keeps target/ compilation assets from blowing up tree depths!
		if e.IsDir() && (name == "target" || name == "build" || name == "node_modules" || name == "bin") {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		// 3. FILTER: Block binary archives and compiled bytecode elements from entering list arrays
		if ext == ".jar" || ext == ".war" || ext == ".zip" || ext == ".class" || ext == ".exe" || ext == ".png" || ext == ".jpg" {
			continue
		}

		fullNodePath := filepath.Join(currentPath, e.Name())

		node := model.FileNode{
			Name:     e.Name(),
			FullPath: fullNodePath,
			IsDir:    e.IsDir(),
			Depth:    depth,
		}
		m.state.TreeNodes = append(m.state.TreeNodes, node)

		// If it's a directory and previously set to expanded, recursively append its children inline
		// For the first setup layout run, directories initialize as collapsed
	}
}

// Helper to rebuild tree state on expansion or collapse actions
func (m *appModel) rebuildActiveTree() {
	if len(m.state.Config.Projects) == 0 {
		return
	}
	proj := m.state.Config.Projects[m.state.SelectedProj]
	rootPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)

	// Capture which directories are expanded before wiping state
	expandedPaths := make(map[string]bool)
	for _, n := range m.state.TreeNodes {
		if n.IsDir && n.IsExpanded {
			expandedPaths[n.FullPath] = true
		}
	}

	var freshTree []model.FileNode

	var walkDir func(string, int)
	walkDir = func(currentPath string, depth int) {
		entries, err := ioutil.ReadDir(currentPath)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			// 1. FILTER: Ignore hidden system dotfiles
			if strings.HasPrefix(name, ".") && name != ".gitignore" {
				continue
			}

			// 2. FILTER: Block standard compiled asset artifact output folders completely
			// This keeps target/ compilation assets from blowing up tree depths!
			if e.IsDir() && (name == "target" || name == "build" || name == "node_modules" || name == "bin") {
				continue
			}

			ext := strings.ToLower(filepath.Ext(name))
			// 3. FILTER: Block binary archives and compiled bytecode elements from entering list arrays
			if ext == ".jar" || ext == ".war" || ext == ".zip" || ext == ".class" || ext == ".exe" || ext == ".png" || ext == ".jpg" {
				continue
			}
			fullP := filepath.Join(currentPath, e.Name())
			isExp := expandedPaths[fullP]

			node := model.FileNode{
				Name:       e.Name(),
				FullPath:   fullP,
				IsDir:      e.IsDir(),
				IsExpanded: isExp,
				Depth:      depth,
			}
			freshTree = append(freshTree, node)

			// Descend if directory is set to expanded mode status
			if node.IsDir && isExp {
				walkDir(fullP, depth+1)
			}
		}
	}

	walkDir(rootPath, 0)
	m.state.TreeNodes = freshTree
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
		if m.state.SelectedConfigOption < 3 { // Total of 4 structural options categories (0 to 3)
			m.state.SelectedConfigOption++
		}
		return m, nil

	case "enter":
		// !!! REUSE PRE-EXISTING WIZARD MODAL STATE TRIGGERS CONTEXT CHANNELS !!!
		switch m.state.SelectedConfigOption {
		case 0:
			// Option 1: Reuse the global configuration setup fields from your Installer sequence
			m.state.ViewState = model.StateInstaller
			m.state.InstallerStep = model.StepSetGlobalPrefs
			m.state.FocusedInput = 0
			m.state.Inputs[0].SetValue(m.state.Config.BasePath)
			m.state.Inputs[1].SetValue(m.state.Config.GitUsername)
			m.state.Inputs[2].SetValue(m.state.Config.GitEmail)
			m.state.Inputs[0].Focus()
			return m, textinput.Blink

		case 1:
			// Option 2: Reuse your Java JDK Environment Setup Manager screen (Ctrl+J)
			m.state.ViewState = model.StateJDKConfigModal
			m.state.JDKStep = model.StepSelectJDKAction
			m.state.SelectedMenuIdx = 0
			return m, nil

		case 2:
			// Option 3: Reuse your Apache Maven Build Setup Manager screen (Ctrl+U)
			m.state.ViewState = model.StateMavenConfigModal
			m.state.MvnStep = model.StepSelectMvnAction
			m.state.SelectedMenuIdx = 0
			return m, nil

		case 3:
			// Option 4: Reuse your 'Add New Workspace Project' data form overlay modal (Ctrl+N)
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

	case "ctrl+t": // Toggle search mode between Titles and Code Content text strings
		if m.state.FuzzyMode == model.FuzzyModeFiles {
			m.state.FuzzyMode = model.FuzzyModeContent
		} else {
			m.state.FuzzyMode = model.FuzzyModeFiles
		}
		m.state.SelectedFuzzy = 0
		m.runFuzzySearchEngine()
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

			// Find the index in our main TreeNodes slice matching the target path
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

	// Route alphanumeric character keystrokes into the editor query buffer

	var cmd tea.Cmd
	oldVal := m.state.FuzzyQueryInput.Value()

	// FIXED: Explicitly cast your KeyMsg back up into a general tea.Msg
	// This ensures the underlying textinput state loop processes character bytes correctly!
	var genericMsg tea.Msg = msg
	m.state.FuzzyQueryInput, cmd = m.state.FuzzyQueryInput.Update(genericMsg)

	// If query changed, fire the real-time lookup index scoring recalculation matches
	if m.state.FuzzyQueryInput.Value() != oldVal {
		m.state.SelectedFuzzy = 0
		m.runFuzzySearchEngine()

		// Secure fix: Instantly rebuild and load text content straight into the right pane
		m.syncFuzzyPreviewPane()
	}

	// Create an overarching batch list to route internal model events cleanly
	var cmdList []tea.Cmd
	if cmd != nil {
		cmdList = append(cmdList, cmd)
	}

	return m, tea.Batch(cmdList...)
}

// Add this method to main.go right under your updateFuzzyModal methods:
func (m *appModel) syncFuzzyPreviewPane() {
	if len(m.state.FuzzyResults) == 0 || m.state.SelectedFuzzy >= len(m.state.FuzzyResults) {
		m.state.FuzzyViewer.SetContent("No file selected for preview context.")
		return
	}

	res := m.state.FuzzyResults[m.state.SelectedFuzzy]

	// !!! FIX 1: EXTENSION WHTELIST GUARD FOR THE FUZZY PREVIEW PANE !!!
	ext := strings.ToLower(filepath.Ext(res.FileName))
	isTextFile := ext == ".go" || ext == ".kt" || ext == ".java" || ext == ".xml" || ext == ".json" ||
		ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".jsx" || ext == ".tsx" || ext == ".ts" || ext == ".txt" ||
		ext == ".md" || ext == ".sh" || ext == ".sql" || res.FileName == "Dockerfile" || res.FileName == "pom.xml"

	if !isTextFile {
		// Intercept and print a safe notice instead of reading raw binary data
		notice := fmt.Sprintf("\n  📦 [BINARY ARTIFACT] Previews Blocked\n\n  File: %s\n\n  Fuzzy preview is disabled for compiled binary artifacts to protect terminal layout encoding.", res.FileName)
		m.state.FuzzyViewer.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true).Render(notice))
		return
	}

	// Verified safe text file. Load normally
	data, err := ioutil.ReadFile(res.FullPath)
	if err != nil {
		m.state.FuzzyViewer.SetContent(fmt.Sprintf("❌ Error opening preview track: %v", err))
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

	proj := m.state.Config.Projects[m.state.SelectedProj]
	rootPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)

	if m.state.FuzzyMode == model.FuzzyModeFiles {
		// Mode 1: File Names Finder
		var traverse func(string)
		traverse = func(p string) {
			files, err := ioutil.ReadDir(p)
			if err != nil {
				return
			}
			for _, f := range files {
				name := f.Name()

				// !!! FIX 2: IGNORE BUILD DIRECTORIES IN SEARCH TRAVERSAL !!!
				if f.IsDir() && (name == "target" || name == "build" || name == "node_modules" || name == "bin" || name == ".git") {
					continue
				}
				if strings.HasPrefix(name, ".") && name != ".gitignore" {
					continue
				}

				fullP := filepath.Join(p, name)
				ext := strings.ToLower(filepath.Ext(name))

				// !!! FIX 3: IGNORE COMPRESSED ARCHIVES & SYSTEM IMAGES !!!
				if ext == ".jar" || ext == ".war" || ext == ".zip" || ext == ".class" || ext == ".exe" || ext == ".png" || ext == ".jpg" {
					continue
				}

				if strings.Contains(strings.ToLower(name), query) {
					m.state.FuzzyResults = append(m.state.FuzzyResults, model.FuzzyResult{
						FileName: name,
						FullPath: fullP,
					})
				}
				if f.IsDir() {
					traverse(fullP)
				}
			}
		}
		traverse(rootPath)
	} else {
		// Mode 2: Deep Content Text Scanning
		var deepScan func(string)
		deepScan = func(p string) {
			files, err := ioutil.ReadDir(p)
			if err != nil {
				return
			}
			for _, f := range files {
				name := f.Name()

				// !!! FIX 4: IGNORE BUILD DIRECTORIES IN DEEP CONTENT SCANNING !!!
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
					// Strict whitelist of human-readable text code files
					if ext == ".xml" || ext == ".go" || ext == ".json" || ext == ".java" || ext == ".txt" || ext == ".md" || ext == ".properties" || ext == ".yml" || ext == ".yaml" || name == "Dockerfile" {
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
								if len(m.state.FuzzyResults) > 150 {
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

func (m *appModel) updateWorkspaceFiles() tea.Cmd {
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

		m.state.TreeNodes = []model.FileNode{}
		m.buildTreeNodes(fullPath, 0)

		m.state.Files = projectFiles
		return model.FileLoadMsg("sync")
	}
}

func (m *appModel) readFileContentCmd() tea.Cmd {
	return func() tea.Msg {
		if len(m.state.TreeNodes) == 0 || m.state.SelectedFile >= len(m.state.TreeNodes) {
			return model.StatusMsg("No nodes currently highlighted inside workspace hierarchy.")
		}

		node := m.state.TreeNodes[m.state.SelectedFile]
		if node.IsDir {
			return model.StatusMsg(fmt.Sprintf("Directory selected: %s", node.Name))
		}

		// EXTENSION POLICING: Explicitly maintain an allowed whitelist of editable developer source extensions
		ext := strings.ToLower(filepath.Ext(node.Name))
		isTextFile := ext == ".go" || ext == ".java" || ext == ".xml" || ext == ".json" ||
			ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".txt" ||
			ext == ".md" || ext == ".sh" || ext == ".sql" || node.Name == "Dockerfile" || node.Name == "pom.xml"

		if !isTextFile {
			// FIXED: Block reading. Render a clean text notification box instead of raw binary bytes noise
			notice := fmt.Sprintf("\n  📦 [BINARY ARTIFACT] Previews Blocked\n\n  File: %s\n  Type: Compiled Binary Asset Context\n\n  Go-Dark blocks loading binary formats to prevent terminal encoding distortion.", node.Name)
			m.state.FileViewer.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true).Render(notice))
			return model.StatusMsg(fmt.Sprintf("⚠️ Blocked unreadable binary asset format: %s", node.Name))
		}

		// Safe route: File is verified plain-text. Stream contents normally
		data, err := ioutil.ReadFile(node.FullPath)
		if err != nil {
			return model.StatusMsg(fmt.Sprintf("Failed to stream layout text: %v", err))
		}

		m.state.FileViewer.SetContent(string(data))
		return model.StatusMsg(fmt.Sprintf("Inspecting file relative path structure: %s", node.Name))
	}
}

func (m *appModel) spawnBackgroundSession(proj config.Project, targetStep string) tea.Cmd {
	m.state.NextSessionID++
	sID := m.state.NextSessionID

	var mvnBin = "mvn"
	if proj.Type != "java" {
		mvnBin = "docker"
	}
	cmdStr := fmt.Sprintf("%s clean %s", mvnBin, targetStep)
	if proj.Type != "java" {
		cmdStr = fmt.Sprintf("docker build -t %s:latest .", strings.ToLower(proj.Name))
	}

	session := &model.BuildSession{
		ID:          sID,
		ProjectName: proj.Name,
		Command:     cmdStr,
		IsRunning:   true,
		Logs:        []string{fmt.Sprintf("🚀 [Session %d] Initializing background execution...", sID)},
	}
	m.state.Sessions[sID] = session
	m.state.ActiveSessionID = sID

	localCh := make(chan string, 500)

	go func() {
		fullProjPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)

		var targetMvnPath string
		for _, mvn := range m.state.Config.Mavens {
			if mvn.Name == proj.MavenName {
				targetMvnPath = mvn.Path
				break
			}
		}

		binPath := "mvn"
		var cmdArgs []string
		if proj.Type == "java" {
			if targetMvnPath != "" {
				binPath = filepath.Join(targetMvnPath, "bin", "mvn")
			}
			cmdArgs = []string{"clean", targetStep}
		} else {
			binPath = "docker"
			cmdArgs = []string{"build", "-t", strings.ToLower(proj.Name) + ":latest", "."}
		}

		cmd := exec.Command(binPath, cmdArgs...)
		cmd.Dir = fullProjPath

		var targetJDKPath string
		for _, jdk := range m.state.Config.JDKs {
			if jdk.Name == proj.JDKName {
				targetJDKPath = jdk.Path
				break
			}
		}

		env := os.Environ()
		if targetJDKPath != "" {
			env = append(env, fmt.Sprintf("JAVA_HOME=%s", targetJDKPath))
		}
		if targetMvnPath != "" {
			env = append(env, fmt.Sprintf("MAVEN_HOME=%s", targetMvnPath), fmt.Sprintf("M2_HOME=%s", targetMvnPath))
		}
		var prefixes []string
		if targetJDKPath != "" {
			prefixes = append(prefixes, filepath.Join(targetJDKPath, "bin"))
		}
		if targetMvnPath != "" {
			prefixes = append(prefixes, filepath.Join(targetMvnPath, "bin"))
		}
		if len(prefixes) > 0 {
			env = append(env, fmt.Sprintf("PATH=%s:%s", strings.Join(prefixes, ":"), os.Getenv("PATH")))
		}
		cmd.Env = env

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			localCh <- fmt.Sprintf("❌ Setup Pipe Exception: %v", err)
			close(localCh)
			return
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			localCh <- fmt.Sprintf("❌ Start Process Exception: %v", err)
			close(localCh)
			return
		}

		localCh <- fmt.Sprintf("$ %s %s", binPath, strings.Join(cmdArgs, " "))
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			localCh <- scanner.Text()
		}

		waitErr := cmd.Wait()
		if waitErr != nil {
			localCh <- fmt.Sprintf("❌ Exec Terminated with fault code: %v", waitErr)
		}
		close(localCh)
	}()

	return listenToSessionChannel(sID, localCh)
}

func listenToSessionChannel(sID int, ch chan string) tea.Cmd {
	if ch != nil {
		sessionChannels[sID] = ch
	}
	return func() tea.Msg {
		activeCh, exists := sessionChannels[sID]
		if !exists {
			return model.BuildCompleteMsg{SessionID: sID, Err: fmt.Errorf("channel untracked")}
		}

		line, ok := <-activeCh
		if !ok {
			return model.BuildCompleteMsg{SessionID: sID, Err: nil}
		}
		return model.BuildLogLineMsg{SessionID: sID, Line: line}
	}
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.TerminalW = msg.Width
		m.state.TerminalH = msg.Height
		m.state.FileViewer.Width = (msg.Width / 2) - 4
		m.state.FileViewer.Height = max(msg.Height-8, 5)
		m.state.FuzzyViewer.Width = (msg.Width / 2) - 4
		m.state.FuzzyViewer.Height = max(msg.Height-12, 5)

	case model.FileLoadMsg:
		if len(m.state.TreeNodes) > 0 {
			if m.state.SelectedFile >= len(m.state.TreeNodes) {
				m.state.SelectedFile = 0
			}
			cmds = append(cmds, m.readFileContentCmd())
		} else {
			m.state.FileViewer.SetContent("Empty project directory root.")
		}

	case model.ConfigRefreshedMsg:
		m.state.Config = config.Config(msg)
		cmds = append(cmds, m.updateWorkspaceFiles())

	case model.StatusMsg:
		m.state.StatusMsg = string(msg)

	case model.BuildLogLineMsg:
		if sess, exists := m.state.Sessions[msg.SessionID]; exists {
			if msg.Line != "" {
				// Apply real-time regex color highlighting filters
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
			sess.Logs = append(sess.Logs, "--------------------------------------------------------")
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

	case tea.KeyMsg:
		if msg.String() == "ctrl+q" {
			return m, tea.Quit
		}

		if (msg.String() == "?" || msg.String() == "h") && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateHelpModal
			return m, nil
		}

		if msg.String() == "ctrl+g" && m.state.ViewState == model.StateDashboard {
			if len(m.state.Config.Projects) > 0 {
				m.state.ViewState = model.StateGitOpsModal
				m.state.GitOpsStep = model.StepSelectGitProject
				m.state.SelectedGitProj = m.state.SelectedProj
				m.state.SelectedGitCmd = 0
				return m, nil
			}
		}

		if msg.String() == "ctrl+j" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateJDKConfigModal
			m.state.JDKStep = model.StepSelectJDKAction
			m.state.SelectedMenuIdx = 0
			m.state.SelectedJDKIdx = 0
			return m, nil
		}

		if msg.String() == "ctrl+y" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateConfigDeckModal
			m.state.SelectedConfigOption = 0
			return m, nil
		}

		if msg.String() == "ctrl+u" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateMavenConfigModal
			m.state.MvnStep = model.StepSelectMvnAction
			m.state.SelectedMenuIdx = 0
			m.state.SelectedMvnIdx = 0
			return m, nil
		}

		if msg.String() == "ctr+t" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateDashboard
			return m, nil
		}

		if msg.String() == "ctrl+f" && m.state.ViewState == model.StateDashboard {
			if len(m.state.Config.Projects) > 0 {
				m.state.ViewState = model.StateFuzzyModal
				m.state.FuzzyMode = model.FuzzyModeFiles
				m.state.FuzzyQueryInput.SetValue("")
				m.state.FuzzyResults = []model.FuzzyResult{}
				m.state.SelectedFuzzy = 0
				m.state.FuzzyQueryInput.Focus()
				return m, textinput.Blink
			}
		}

		if msg.String() == "ctrl+b" && m.state.ViewState == model.StateDashboard {
			if len(m.state.Config.Projects) > 0 {
				m.state.ViewState = model.StateBuildModal
				m.state.SelectedBuildOpt = 0
				m.state.BuildLogs = []string{"Console engine ready. Select command step to initialize stream..."}
				m.state.IsBuilding = false
				return m, nil
			}
		}

		if msg.String() == "ctrl+s" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateSessionLogsModal
			m.state.ViewingSessionID = 0
			return m, nil
		}

		if msg.String() == "esc" {
			if m.state.ViewState == model.StateBuildModal {
				if m.state.ActiveSessionID != 0 {
					delete(m.state.Sessions, m.state.ActiveSessionID)
					m.state.ActiveSessionID = 0
				}
				m.state.ViewState = model.StateDashboard
				return m, nil
			}
			if m.state.ViewState == model.StateSessionLogsModal || m.state.ViewState == model.StateHelpModal {
				m.state.ViewState = model.StateDashboard
				return m, nil
			}
		}

		if m.state.ViewState == model.StateSessionLogsModal {
			if msg.String() >= "1" && msg.String() <= "9" {
				runes := []rune(msg.String())
				if len(runes) > 0 {
					targetID := int(runes[0] - '0')
					m.state.ViewingSessionID = targetID
					return m, nil
				}
			}
		}

		switch m.state.ViewState {
		case model.StateHelpModal:
			m.state.ViewState = model.StateDashboard
			return m, nil
		case model.StateInstaller:
			return m.updateInstaller(msg)
		case model.StateAddProjectModal:
			return m.updateModalForm(msg)
		case model.StateGitOpsModal:
			return m.updateGitOpsModal(msg)
		case model.StateJDKConfigModal:
			return m.updateJDKModal(msg)
		case model.StateMavenConfigModal:
			return m.updateMvnModal(msg)
		case model.StateBuildModal:
			return m.updateBuildModal(msg)
		case model.StateFuzzyModal:
			return m.updateFuzzyModal(msg)
		case model.StateConfigDeckModal:
			return m.updateConfigDeckModal(msg)
		default:
			return m.updateDashboardPortal(msg)
		}
	}

	// FIXED GLOBAL VIEWPORT DISPATCH ROUTERS
	// This ensures only the active, visible viewport receives scroll events
	if m.state.ViewState == model.StateDashboard {
		var viewCmd tea.Cmd
		m.state.FileViewer, viewCmd = m.state.FileViewer.Update(msg)
		cmds = append(cmds, viewCmd)
	}

	if m.state.ViewState == model.StateFuzzyModal {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			// Forward scrolling keys explicitly down to the live viewport instance
			case "pgup", "pgdown", "up", "down":
				var cmd tea.Cmd
				m.state.FuzzyViewer, cmd = m.state.FuzzyViewer.Update(msg)
				return m, cmd

			case "ctrl+j", "tab":
				if len(m.state.FuzzyResults) > 0 {
					m.state.SelectedFuzzy = (m.state.SelectedFuzzy + 1) % len(m.state.FuzzyResults)
					m.syncFuzzyPreviewPane()
					// Pro-Tip: Call your preview-reloader function here to sync the right pane content on change!
				}
				return m, nil

			case "ctrl+k", "shift+tab":
				if len(m.state.FuzzyResults) > 0 {
					m.state.SelectedFuzzy = (m.state.SelectedFuzzy - 1 + len(m.state.FuzzyResults)) % len(m.state.FuzzyResults)
					// Pro-Tip: Call your preview-reloader function here to sync the right pane content on change!
					m.syncFuzzyPreviewPane()
				}
				return m, nil
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *appModel) updateBuildModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var currentSessionID int
	if len(m.state.Config.Projects) > 0 {
		targetProj := m.state.Config.Projects[m.state.SelectedProj]
		for id, sess := range m.state.Sessions {
			if sess.ProjectName == targetProj.Name && sess.IsRunning {
				currentSessionID = id
				break
			}
		}
	}
	if currentSessionID != 0 {
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	case "left", "h":
		if m.state.SelectedBuildOpt > 0 {
			m.state.SelectedBuildOpt--
		}
	case "right", "l":
		if m.state.SelectedBuildOpt < len(m.state.BuildOptions)-1 {
			m.state.SelectedBuildOpt++
		}
	case "enter":
		m.state.IsBuilding = true
		chosenOpt := m.state.BuildOptions[m.state.SelectedBuildOpt]
		proj := m.state.Config.Projects[m.state.SelectedProj]
		fullProjPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)
		backupDir := fullProjPath + "_backup_target"
		_ = os.RemoveAll(backupDir)
		_ = exec.Command("cp", "-r", fullProjPath, backupDir).Run()
		_ = exec.Command("git", "clean", "-xdf").Run()
		return m, m.spawnBackgroundSession(proj, chosenOpt)
	}
	return m, nil
}

func (m *appModel) updateMvnModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state.MvnStep {
	case model.StepSelectMvnAction:
		switch msg.String() {
		case "esc":
			m.state.ViewState = model.StateConfigDeckModal
			return m, nil
		case "up", "k":
			if m.state.SelectedMenuIdx > 0 {
				m.state.SelectedMenuIdx--
			}
		case "down", "j":
			if m.state.SelectedMenuIdx < 1 {
				m.state.SelectedMenuIdx++
			}
		case "enter":
			if m.state.SelectedMenuIdx == 0 {
				m.state.MvnStep = model.StepAddNewMvnVersion
				m.state.FocusedInput = 9
				m.state.Inputs[9].SetValue("")
				m.state.Inputs[10].SetValue("")
				m.state.Inputs[9].Focus()
			} else {
				m.state.MvnStep = model.StepAssignMvnToProject
				m.state.SelectedMvnIdx = 0
			}
		}
	case model.StepAddNewMvnVersion:
		switch msg.String() {
		case "esc":
			m.state.MvnStep = model.StepSelectMvnAction
			return m, nil
		case "tab", "down":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 19 - m.state.FocusedInput
			if m.state.FocusedInput < 9 || m.state.FocusedInput > 10 {
				m.state.FocusedInput = 9
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "enter":
			mVnName := m.state.Inputs[9].Value()
			mVnPath := m.state.Inputs[10].Value()
			if mVnName != "" && mVnPath != "" {
				m.state.Config.Mavens = append(m.state.Config.Mavens, config.MavenProfile{Name: mVnName, Path: mVnPath})
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Added Maven Profile: %s", mVnName)
			}
			m.state.MvnStep = model.StepSelectMvnAction
			return m, nil
		}
		var cmd tea.Cmd
		m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
		return m, cmd
	case model.StepAssignMvnToProject:
		switch msg.String() {
		case "esc":
			m.state.MvnStep = model.StepSelectMvnAction
			return m, nil
		case "up", "k":
			if m.state.SelectedMvnIdx > 0 {
				m.state.SelectedMvnIdx--
			}
		case "down", "j":
			if m.state.SelectedMvnIdx < len(m.state.Config.Mavens)-1 {
				m.state.SelectedMvnIdx++
			}
		case "enter":
			if len(m.state.Config.Mavens) > 0 && len(m.state.Config.Projects) > 0 {
				chosenMvn := m.state.Config.Mavens[m.state.SelectedMvnIdx]
				m.state.Config.Projects[m.state.SelectedProj].MavenName = chosenMvn.Name
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
			m.state.ViewState = model.StateDashboard
			return m, nil
		case "up", "k":
			if m.state.SelectedMenuIdx > 0 {
				m.state.SelectedMenuIdx--
			}
		case "down", "j":
			if m.state.SelectedMenuIdx < 1 {
				m.state.SelectedMenuIdx++
			}
		case "enter":
			if m.state.SelectedMenuIdx == 0 {
				m.state.JDKStep = model.StepAddNewJDKVersion
				m.state.FocusedInput = 7
				m.state.Inputs[7].SetValue("")
				m.state.Inputs[8].SetValue("")
				m.state.Inputs[7].Focus()
			} else {
				m.state.JDKStep = model.StepAssignJDKToProject
				m.state.SelectedJDKIdx = 0
			}
		}
	case model.StepAddNewJDKVersion:
		switch msg.String() {
		case "esc":
			m.state.JDKStep = model.StepSelectJDKAction
			return m, nil
		case
			"tab", "down":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 15 - m.state.FocusedInput
			if m.state.FocusedInput < 7 || m.state.FocusedInput > 8 {
				m.state.FocusedInput = 7
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "enter":
			jName := m.state.Inputs[7].Value()
			jPath := m.state.Inputs[8].Value()
			if jName != "" && jPath != "" {
				m.state.Config.JDKs = append(m.state.Config.JDKs, config.JDKProfile{Name: jName, Path: jPath})
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
		case
			"up", "k":
			if m.state.SelectedJDKIdx > 0 {
				m.state.SelectedJDKIdx--
			}
		case "down", "j":
			if m.state.SelectedJDKIdx < len(m.state.Config.JDKs)-1 {
				m.state.SelectedJDKIdx++
			}
		case "enter":
			if len(m.state.Config.JDKs) > 0 && len(m.state.Config.Projects) > 0 {
				chosenJDK := m.state.Config.JDKs[m.state.SelectedJDKIdx]
				m.state.Config.Projects[m.state.SelectedProj].JDKName = chosenJDK.Name
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Assigned JDK Environment: %s", chosenJDK.Name)
			}
			m.state.ViewState = model.StateDashboard
			return m, m.updateWorkspaceFiles()
		}
	}
	return m, nil
}
func (m *appModel) updateGitOpsModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
func (m *appModel) runGitCommandCmd(proj config.Project, operation string) tea.Cmd {
	return func() tea.Msg {
		fullPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)
		var cmd *exec.Cmd
		switch operation {
		case "clone":
			if proj.GitURL == "" {
				return model.StatusMsg("❌ Git Operation Aborted: No URL found.")
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
		out, err := cmd.CombinedOutput()
		if err != nil {
			return model.StatusMsg(fmt.Sprintf("❌ Error: %v | Log: %s", err, string(out)))
		}
		return model.StatusMsg(fmt.Sprintf("✅ git %s successfully completed.", operation))
	}
}
func (m *appModel) updateInstaller(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		case "esc":
			m.state.ViewState = model.StateConfigDeckModal
			return m, nil
		case "enter":
			m.state.Config.BasePath = m.state.Inputs[0].Value()
			m.state.Config.GitUsername = m.state.Inputs[1].Value()
			m.state.Config.GitEmail = m.state.Inputs[2].Value()
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
			m.state.Config.Projects = append(m.state.Config.Projects, config.Project{Name: name, Path: path, Type: strings.ToLower(pType), GitURL: gitURL})
		}
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, m.updateWorkspaceFiles()
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}
func (m *appModel) updateDashboardPortal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab":
		m.state.ActiveFocus = model.FocusArea((int(m.state.ActiveFocus) + 1) % 3)

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
		} else if m.state.ActiveFocus == model.FocusTree && m.state.SelectedFile < len(m.state.TreeNodes)-1 {
			m.state.SelectedFile++
			cmds = append(cmds, m.readFileContentCmd())
		}

	case "enter", "right", "l":
		if m.state.ActiveFocus == model.FocusMenu {
			cmds = append(cmds, m.executeActiveMenuAction())
		} else if m.state.ActiveFocus == model.FocusTree && len(m.state.TreeNodes) > 0 {
			idx := m.state.SelectedFile
			if m.state.TreeNodes[idx].IsDir {
				// Toggle folder state expansion parameters
				m.state.TreeNodes[idx].IsExpanded = !m.state.TreeNodes[idx].IsExpanded
				m.rebuildActiveTree()
			} else {
				cmds = append(cmds, m.readFileContentCmd())
			}
		}

	case "ctrl+e":
		if len(m.state.TreeNodes) == 0 || m.state.SelectedFile < 0 || m.state.SelectedFile >= len(m.state.TreeNodes) {
			return m, nil
		}

		// 2. Target the highlighted node
		selectedNode := m.state.TreeNodes[m.state.SelectedFile]

		// 3. Prevent trying to open folders in NeoVim
		if selectedNode.IsDir {
			return m, nil
		}

		// 4. Instantiate the system application execution argument context
		// (Change selectedNode.Path to whatever field holds your full system filepath)
		c := exec.Command("nvim", selectedNode.FullPath)

		// 5. Hand raw terminal process control directly over to NeoVim
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			// This callback loop executes automatically the moment you type ':q' or ':wq' in nvim
			return nil // Returning nil forces bubbletea to safely clear the terminal and redraw your dashboard
		})

	case "left", "h":
		if m.state.ActiveFocus == model.FocusTree && len(m.state.TreeNodes) > 0 {
			idx := m.state.SelectedFile
			if m.state.TreeNodes[idx].IsDir && m.state.TreeNodes[idx].IsExpanded {
				m.state.TreeNodes[idx].IsExpanded = false
				m.rebuildActiveTree()
			}
		}
	}
	return m, tea.Batch(cmds...)
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
		newProj := config.Project{Name: name, Path: path, Type: strings.ToLower(pType), GitURL: gitURL}
		m.state.Config.Projects = append(m.state.Config.Projects, newProj)
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, func() tea.Msg { return model.ConfigRefreshedMsg(m.state.Config) }
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}
func (m *appModel) executeActiveMenuAction() tea.Cmd {
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
		return model.StatusMsg("Manual workspace isolation completed.")
	}
}
func (m *appModel) View() string {
	switch m.state.ViewState {
	case model.StateHelpModal:
		return components.RenderHelpModal(m.state)
	case model.StateInstaller:
		return components.RenderInstaller(m.state)
	case model.StateAddProjectModal:
		return components.RenderModal(m.state)
	case model.StateGitOpsModal:
		return components.RenderGitOpsModal(m.state)
	case model.StateJDKConfigModal:
		return components.RenderJDKConfigModal(m.state)
	case model.StateMavenConfigModal:
		return components.RenderMvnConfigModal(m.state)
	case model.StateBuildModal:
		return components.RenderBuildModal(m.state)
	case model.StateSessionLogsModal:
		return components.RenderSessionLogsModal(m.state)
	case model.StateFuzzyModal:
		return components.RenderFuzzyModal(&m.state)
	case model.StateConfigDeckModal:
		return components.RenderConfigDeckModal(m.state)
	default:
		topBar := panels.RenderTopMenu(m.state)
		body := panels.RenderMainBody(m.state)
		var boundJDK string
		if len(m.state.Config.Projects) > 0 && m.state.SelectedProj < len(m.state.Config.Projects) {
			boundJDK = m.state.Config.Projects[m.state.SelectedProj].JDKName
		}
		if boundJDK == "" {
			boundJDK = "System Default"
		}
		var activeCount int
		for _, s := range m.state.Sessions {
			if s.IsRunning {
				activeCount++
			}
		}
		var sessionStatusStr = "\x1b[90mNo active background sessions\x1b[0m"
		if activeCount > 0 {
			sessionStatusStr = fmt.Sprintf("⚡ \x1b[33;1mBackground Active: %d Running\x1b[0m", activeCount)
		}
		footerText := fmt.Sprintf(" Press [?] for Help | [Ctrl+F] Fuzzy Find | Bound: %s | %s | Status: %s", boundJDK, sessionStatusStr, m.state.StatusMsg)
		footer := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("250")).Width(m.state.TerminalW).Render(footerText)
		return lipgloss.JoinVertical(lipgloss.Left, topBar, body, footer)
	}
}
func main() {
	cfg, isFirstRun := config.LoadConfig()
	_, gitErr := exec.LookPath("git")
	gitMissing :=
		gitErr != nil
	home, _ := os.UserHomeDir()
	inputs := make([]textinput.Model, 11)
	for i := range inputs {
		inputs[i] = textinput.New()
	}
	inputs[0].Placeholder = "Global Workspace Base Path"
	inputs[0].SetValue(filepath.Join(home, "Developer"))
	inputs[1].Placeholder = "e.g. John Doe"
	inputs[2].Placeholder = "e.g. john@example.com"
	inputs[3].Placeholder = "My Application Service"
	inputs[4].Placeholder = "my-service-folder"
	inputs[5].Placeholder = "java"
	inputs[6].Placeholder = "git@github.com:user/repo.git"
	inputs[7].Placeholder = "Profile Name (e.g. Java-17)"
	inputs[8].Placeholder = "JAVA_HOME path (e.g. /usr/lib/jvm/...)"
	inputs[9].Placeholder = "Maven Profile Name (e.g. Maven-3.9)"
	inputs[10].Placeholder = "MAVEN_HOME directory path"
	initialState := model.StateDashboard
	if isFirstRun || gitMissing {
		initialState = model.StateInstaller
		if !gitMissing {
			inputs[0].Focus()
		}
	}
	m := &appModel{
		state: model.UIState{
			Config:          cfg,
			ViewState:       initialState,
			InstallerStep:   model.StepSetGlobalPrefs,
			ActiveFocus:     model.FocusProjects,
			FileViewer:      viewport.New(30, 20),
			Inputs:          inputs,
			GitMissing:      gitMissing,
			GitCommands:     []string{"fetch", "pull", "clone", "checkout (main)"},
			BuildOptions:    []string{"clean", "test", "compile", "package", "install"},
			BuildLogs:       []string{"Console ready. Select option step to launch..."},
			Sessions:        make(map[int]*model.BuildSession),
			FuzzyQueryInput: textinput.New(),
			FuzzyViewer:     viewport.New(30, 20),
		},
	}
	m.state.FuzzyQueryInput.Placeholder = "Type lookup phrase attributes (e.g. controller)..."
	m.state.FuzzyQueryInput.CharLimit = 50
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		os.Exit(1)
	}
}
