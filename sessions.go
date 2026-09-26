package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

var sessionChannels = make(map[int]chan string)

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
		d = "install -DskipTests"
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
		fullProjPath := proj.Path

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

func (m *appModel) pollDockerTelemetryCmd() tea.Cmd {
	return func() tea.Msg {
		cmdRun := exec.Command("docker", "ps", "-q")
		outRun, _ := cmdRun.Output()
		runningCount := len(strings.Split(strings.TrimSpace(string(outRun)), "\n"))
		if string(outRun) == "" {
			runningCount = 0
		}

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
