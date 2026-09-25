package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
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

type WorkspaceRefreshedMsg struct {
	Files     []string
	TreeNodes []model.FileNode
}
type GitStatusLoadedMsg string
type GitStatusErrorMsg error
type GitBranchesLoadedMsg []string

type GitBranchesErrorMsg error
type GitCheckoutCompleteMsg struct {
	Output string
	Err    error
}
type appModel struct {
	state *model.UIState
}

func (m *appModel) loadGitBranchesCmd() tea.Cmd {
	if len(m.state.Config.Projects) == 0 {
		return func() tea.Msg { return GitBranchesLoadedMsg{"main"} }
	}

	idx := m.state.SelectedGitProject
	if idx < 0 || idx >= len(m.state.Config.Projects) {
		return func() tea.Msg { return GitBranchesLoadedMsg{"main"} }
	}

	dir := filepath.Join(m.state.Config.BasePath, m.state.Config.Projects[idx].Path)
	if dir == "" {
		dir = "."
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return func() tea.Msg {
			// This fires instantly to break the loading screen loop
			return GitBranchesErrorMsg(fmt.Errorf("directory not found: %s", dir))
		}
	}

	return func() tea.Msg {
		// 🟢 FIX: Use 'git branch -a --format' which is more robust across platforms
		// than passing multi-pattern blocks down to plumbing engines.
		cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
		cmd.Dir = dir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return GitBranchesErrorMsg(fmt.Errorf("git failed: %s (%v)", strings.TrimSpace(string(output)), err))
		}

		var branches []string
		seen := make(map[string]bool)

		lines := strings.SplitSeq(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
		for line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			// 🧼 Clean up common structural symbolic pointers
			// "origin/HEAD" -> skip
			if strings.Contains(trimmed, "HEAD") {
				continue
			}

			// Clean up tracking branch decorators
			// "remotes/origin/feature-abc" -> "feature-abc"
			// "origin/feature-abc" -> "feature-abc"
			cleaned := trimmed
			if strings.HasPrefix(cleaned, "remotes/origin/") {
				cleaned = cleaned[len("remotes/origin/"):]
			} else if strings.HasPrefix(cleaned, "origin/") {
				cleaned = cleaned[len("origin/"):]
			}

			if cleaned != "" && !seen[cleaned] {
				seen[cleaned] = true
				branches = append(branches, cleaned)
			}
		}

		if len(branches) == 0 {
			branches = append(branches, "main")
		}

		return GitBranchesLoadedMsg(branches)
	}
}

func (m *appModel) Init() tea.Cmd {
	if m.state.ViewState == model.StateInstaller {
		return textinput.Blink
	}
	// Batch initial workspace discovery alongside the background docker ticker poll
	return tea.Batch(m.updateWorkspaceFiles(), m.pollDockerTelemetryCmd())
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
	proj := m.state.Config.Projects[m.state.SelectedProject]
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
			m.state.SelectedMenuIndex = 0
			return m, nil

		case 2:
			// Option 3: Reuse your Apache Maven Build Setup Manager screen (Ctrl+U)
			m.state.ViewState = model.StateMavenConfigModal
			m.state.MavenStep = model.StepSelectMvnAction
			m.state.SelectedMenuIndex = 0
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

	proj := m.state.Config.Projects[m.state.SelectedProject]
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
	// 1. Resolve indices and paths safely on the MAIN thread before spawning the thread
	idx := m.state.SelectedGitProject
	if idx < 0 || idx >= len(m.state.Config.Projects) {
		return nil
	}

	proj := m.state.Config.Projects[idx]
	fullPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)

	return func() tea.Msg {
		// 2. Perform the isolated file system read
		entries, err := os.ReadDir(fullPath) // ioutil.ReadDir is deprecated, os.ReadDir is preferred
		if err != nil {
			return model.StatusMsg(fmt.Sprintf("Directory missing at target: %s", fullPath))
		}

		var projectFiles []string
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				projectFiles = append(projectFiles, e.Name())
			}
		}

		// 3. Instead of mutating m.state directly, build the nodes into a local variable.
		// NOTE: If buildTreeNodes currently relies on mutating m.state, you should adapt it
		// to return a slice of nodes instead of directly setting m.state.TreeNodes.
		// For now, we will safely pass the calculated files list back.
		return WorkspaceRefreshedMsg{
			Files: projectFiles,
		}
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
	var d = targetStep
	switch d {
	case "without tests":
		d = "install  -DskipTests"
	case "full":
		d = "install"
	}
	cmdStr := fmt.Sprintf("%s clean %s", mvnBin, d)
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
	case GitStatusLoadedMsg:
		m.state.GitStatusOutput = string(msg)
		if m.state.GitStatusOutput == "" {
			m.state.GitStatusOutput = "✨ Working tree completely clean."
		}
		return m, nil

	case WorkspaceRefreshedMsg:
		// 🟢 SAFE MUTATION: Happening entirely on the coordinated main runtime thread
		m.state.Files = msg.Files

		// Run your tree builder calculations safely here on the main thread
		proj := m.state.Config.Projects[m.state.SelectedGitProject]
		fullPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)

		m.state.TreeNodes = []model.FileNode{}
		m.buildTreeNodes(fullPath, 0)

		// Trigger the viewer reloader to load the active file content
		return m, func() tea.Msg { return model.FileLoadMsg("sync") }

	case GitStatusErrorMsg:
		m.state.GitStatusOutput = fmt.Sprintf("❌ Error: %v", msg)
		return m, nil

	case GitBranchesLoadedMsg:
		m.state.AvailableBranches = msg
		m.state.SelectedGitBranch = 0
		return m, nil

	case GitBranchesErrorMsg:
		m.state.AvailableBranches = []string{} // Keep it empty
		m.state.StatusMsg = fmt.Sprintf("❌ Git Error: %v", msg)
		if len(m.state.AvailableBranches) == 0 {
			m.state.AvailableBranches = []string{"main"}
		}
		return m, nil

	case GitCheckoutCompleteMsg:
		if msg.Err != nil {
			m.state.StatusMsg = fmt.Sprintf("❌ Checkout Failed: %v", msg.Err)
		} else {
			m.state.StatusMsg = fmt.Sprintf("✅ Checked out successfully: %s", strings.TrimSpace(msg.Output))
		}
		// Reset state back to base dashboard layout
		m.state.ViewState = model.StateDashboard
		return m, m.updateWorkspaceFiles()

	case model.DockerTelemetryMsg:
		m.state.DockerTelemetry = model.DockerStats(msg)
		// RECURSIVE POLLING SAFETY HOOK: Continue background sweeps silently
		return m, m.pollDockerTelemetryCmd()

	case model.DockerContainersMsg:
		m.state.DockerContainers = []model.DockerContainer(msg)

	case tea.WindowSizeMsg:
		m.state.WindowWidth = msg.Width
		m.state.WindowHeight = msg.Height
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
		if strings.Contains(m.state.StatusMsg, "successfully completed") {
			if strings.Contains(m.state.StatusMsg, "pull") || strings.Contains(m.state.StatusMsg, "reset") {
				return m, m.updateWorkspaceFiles()
			}
		}
		return m, nil

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
				m.state.ViewState = model.StateGitOperationsModal
				m.state.GitOperationStep = model.StepSelectGitProject
				m.state.SelectedGitProject = m.state.SelectedProject
				m.state.SelectedGitCommand = 0
				return m, nil
			}
		}
		if msg.String() == "ctrl+d" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateDockerModal
			m.state.SelectedDockerRow = 0
			return m, m.fetchDockerContainersCmd() // Instantly populate rows table layout
		}

		if msg.String() == "ctrl+j" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateJDKConfigModal
			m.state.JDKStep = model.StepSelectJDKAction
			m.state.SelectedMenuIndex = 0
			m.state.SelectedJDKIndex = 0
			return m, nil
		}

		if msg.String() == "ctrl+y" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateConfigDeckModal
			m.state.SelectedConfigOption = 0
			return m, nil
		}

		if msg.String() == "ctrl+u" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateMavenConfigModal
			m.state.MavenStep = model.StepSelectMvnAction
			m.state.SelectedMenuIndex = 0
			m.state.SelectedMavenIndex = 0
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

		if msg.String() == "ctrl+n" && m.state.ViewState == model.StateDashboard {
			m.state.ViewState = model.StateAddProjectModal
			return m, nil
		}

		if msg.String() == "ctrl+b" && m.state.ViewState == model.StateDashboard {
			if len(m.state.Config.Projects) > 0 {
				m.state.ViewState = model.StateBuildModal
				m.state.SelectedBuildOption = 0
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
			m.state.ViewState = model.StateDashboard
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
		case model.StateGitOperationsModal:
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
		case model.StateDockerModal:
			return m.updateDockerModal(msg)
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

func (m *appModel) executeGitCheckoutCmd(proj config.Project, branch string) tea.Cmd {
	// 🟢 THREAD SAFETY FIX: Resolve the project path string on the MAIN thread first
	fullPath := config.ResolvePath(m.state.Config.BasePath, proj.Path)

	return func() tea.Msg {
		cmd := exec.Command("git", "checkout", branch)
		cmd.Dir = fullPath

		output, err := cmd.CombinedOutput()
		return GitCheckoutCompleteMsg{
			Output: string(output),
			Err:    err,
		}
	}
}

// Add this ticker cmd function to pull stats continuously in the background
func (m *appModel) pollDockerTelemetryCmd() tea.Cmd {
	return func() tea.Msg {
		// 1. Fetch count of total and running containers context models parameters
		cmdRun := exec.Command("docker", "ps", "-q")
		outRun, _ := cmdRun.Output()
		runningCount := len(strings.Split(strings.TrimSpace(string(outRun)), "\n"))
		if string(outRun) == "" {
			runningCount = 0
		}

		// 2. Fetch live global usage stats strings streams safely
		// Using unblocking formats passing limits arguments flags
		statsCmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.CPUPerc}},{{.MemUsage}}")
		outStats, err := statsCmd.Output()

		cpuStr := "0.0%"
		memStr := "0B / 0B"
		if err == nil && string(outStats) != "" {
			lines := strings.Split(strings.TrimSpace(string(outStats)), "\n")
			if len(lines) > 0 && strings.Contains(lines[0], ",") {
				parts := strings.Split(lines[0], ",")
				cpuStr = parts[0]
				memStr = parts[1]
			}
		}

		return model.DockerTelemetryMsg{
			CPU:     cpuStr,
			Memory:  memStr,
			Running: runningCount,
		}
	}
}

func (m *appModel) fetchDockerContainersCmd() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("docker", "ps", "-a", "--format", "{{.ID}},{{.Names}},{{.Image}},{{.Status}},{{.Ports}}")
		out, err := cmd.Output()
		if err != nil {
			return model.DockerContainersMsg{}
		}

		var list []model.DockerContainer
		lines := strings.SplitSeq(strings.TrimSpace(string(out)), "\n")
		for line := range lines {
			if line == "" {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) >= 4 {
				ports := ""
				if len(parts) == 5 {
					ports = parts[4]
				}
				list = append(list, model.DockerContainer{
					ID:     parts[0],
					Names:  parts[1],
					Image:  parts[2],
					Status: parts[3],
					Ports:  ports,
				})
			}
		}
		return model.DockerContainersMsg(list)
	}
}

func (m *appModel) runDockerActionCmd(containerID, action string) tea.Cmd {
	return func() tea.Msg {
		_ = exec.Command("docker", action, containerID).Run()
		return model.StatusMsg(fmt.Sprintf("✅ Successfully executed: docker %s %s", action, containerID))
	}
}

func (m *appModel) updateBuildModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var currentSessionID int
	if len(m.state.Config.Projects) > 0 {
		targetProj := m.state.Config.Projects[m.state.SelectedProject]
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
		proj := m.state.Config.Projects[m.state.SelectedProject]
		return m, m.spawnBackgroundSession(proj, chosenOpt)
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
			m.state.ViewState = model.StateDashboard
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
		case
			"up", "k":
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

func (m *appModel) updateGitOpsModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
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
			// Prefetch branches early so they are loaded by the time they hit checkout
			return m, m.loadGitBranchesCmd()

		case model.StepSelectGitCommand:
			chosenCmd := m.state.GitCommands[m.state.SelectedGitCommand]
			targetProj := m.state.Config.Projects[m.state.SelectedGitProject]
			resolvedDebugPath := filepath.Join(m.state.Config.BasePath, targetProj.Path)
			m.state.StatusMsg = fmt.Sprintf("DEBUG | Target Absolute Path: %s | Cmd: %s", resolvedDebugPath, chosenCmd)

			if strings.HasPrefix(chosenCmd, "checkout") {
				m.state.GitOperationStep = model.StepSelectGitBranch
				m.state.AvailableBranches = []string{} // Clear old ones safely
				m.state.SelectedGitBranch = 0
				return m, m.loadGitBranchesCmd()
			}

			if chosenCmd == "status" {
				m.state.GitOperationStep = 3 // Move to a new status preview display step layer!
				m.state.GitStatusOutput = "⏳ Querying workspace parameters..."
				return m, m.runGitCommand(targetProj, "status")
			}

			if chosenCmd == "reset" {

			}

			switch chosenCmd {
			case "reset":
				chosenCmd = "reset --hard"
			}

			// Clean exit for basic commands (pull, fetch, etc.)
			m.state.ViewState = model.StateDashboard
			proj := m.state.Config.Projects[m.state.SelectedGitProject]
			m.state.StatusMsg = fmt.Sprintf("DEBUG | Path: %s | Cmd: %s", proj.Path, chosenCmd)
			return m, m.runGitCommand(targetProj, chosenCmd) // 🟢 FIX: Do not attach loadGitBranchesCmd here

		case model.StepSelectGitBranch:
			if len(m.state.AvailableBranches) > 0 {
				targetProj := m.state.Config.Projects[m.state.SelectedGitProject]
				targetBranch := m.state.AvailableBranches[m.state.SelectedGitBranch]
				m.state.StatusMsg = fmt.Sprintf("🔄 Checking out %s...", targetBranch)
				return m, m.executeGitCheckoutCmd(targetProj, targetBranch)
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *appModel) runGitCommand(proj config.Project, operation string) tea.Cmd {
	// 🟢 CRITICAL SEPARATOR JOIN FIX: Combine BasePath and Project Path agnostically
	fullPath := filepath.Join(m.state.Config.BasePath, proj.Path)

	return func() tea.Msg {
		var cmd *exec.Cmd
		op := strings.ToLower(strings.TrimSpace(operation))

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
			// Specifying origin and --no-edit keeps background executions automated
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

			// Return the status string back to the coordinator thread
			return GitStatusLoadedMsg(string(out))

		case strings.Contains(op, "reset"):
			cmd = exec.Command("git", "reset", "--hard")
			cmd.Dir = fullPath

		case strings.Contains(op, "checkout"):
			cmd = exec.Command("git", "checkout", "main")
			cmd.Dir = fullPath

		default:
			return model.StatusMsg(fmt.Sprintf("❌ Unknown operation: %s", operation))
		}

		// Inherit host environmental credentials and block interactive hangs
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")

		out, err := cmd.CombinedOutput()
		if err != nil {
			gitLog := strings.ReplaceAll(strings.TrimSpace(string(out)), "\n", " | ")
			if gitLog == "" {
				gitLog = err.Error()
			}
			return model.StatusMsg(fmt.Sprintf("❌ Pull Failed: %s", gitLog))
		}

		return model.StatusMsg(fmt.Sprintf("✅ git %s successfully completed.", operation))
	}
}

func (m *appModel) updateDockerModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.state.SelectedDockerRow > 0 {
			m.state.SelectedDockerRow--
		}
	case "down", "j":
		if m.state.SelectedDockerRow < len(m.state.DockerContainers)-1 {
			m.state.SelectedDockerRow++
		}

	case "s", "t", "r": // [s]tart, s[t]op, [r]estart container commands keys triggers
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

		// Fire action and trigger table re-query refresh sequence batch inline safely
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
		if m.state.ActiveFocus == model.FocusProjects && m.state.SelectedProject > 0 {
			m.state.SelectedProject--
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.state.ActiveFocus == model.FocusTree && m.state.SelectedFile > 0 {
			m.state.SelectedFile--
			cmds = append(cmds, m.readFileContentCmd())
		}

	case "down", "j":
		if m.state.ActiveFocus == model.FocusProjects && m.state.SelectedProject < len(m.state.Config.Projects)-1 {
			m.state.SelectedProject++
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
	proj := m.state.Config.Projects[m.state.SelectedProject]
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
	case model.StateGitOperationsModal:
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
		return components.RenderFuzzyModal(m.state)
	case model.StateConfigDeckModal:
		return components.RenderConfigDeckModal(m.state)
	case model.StateDockerModal:
		return components.RenderDockerModal(m.state)
	default:
		return lipgloss.JoinVertical(lipgloss.Left, panels.RenderTopMenu(m.state), panels.RenderMainBody(m.state), panels.CreateFooter(m.state))
	}
}
func main() {
	f, err := initLogger()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize log file: %v\n", err)
		os.Exit(1)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize log file: %v\n", err)
			os.Exit(1)
		}
	}(f)
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
		state: &model.UIState{
			Config:          cfg,
			ViewState:       initialState,
			InstallerStep:   model.StepSetGlobalPrefs,
			ActiveFocus:     model.FocusProjects,
			FileViewer:      viewport.New(30, 20),
			Inputs:          inputs,
			GitMissing:      gitMissing,
			GitCommands:     []string{"fetch", "pull", "clone", "checkout", "reset", "status"},
			BuildOptions:    []string{"clean", "test", "compile", "package", "without tests", "full"},
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

func initLogger() (*os.File, error) {
	// Create or open a dedicated debug log file
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	// Route Go's standard log package output to this file
	log.SetOutput(f)
	log.Println("--- TUI Engine Session Started ---")
	return f, nil
}
