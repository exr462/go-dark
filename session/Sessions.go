package session

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
	componentmessage "github.com/exr462/go-dark/ui/component/message"
)

var sessionChannels = make(map[int]chan string)

func SpawnBackgroundSession(ui *model.UI, proj config.Project, targetStep string) tea.Cmd {
	ui.NextSessionID++
	sID := ui.NextSessionID

	var mvnBin = "mvn"
	if proj.Type != "java" {
		mvnBin = "docker"
	}
	// BuildOptions:    []string{"clean", "test", "compile", "package", "without tests", "full"},
	switch targetStep {
	case "test":
		targetStep = "clean test"
	case "compile":
		targetStep = "clean compile"
	case "package":
		targetStep = "clean package"
	case "without tests":
		targetStep = "clean install -DskipTests"
	case "full":
		targetStep = "clean install"
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
	ui.Sessions[sID] = session
	ui.ActiveSessionID = sID

	localCh := make(chan string, 500)

	go func() {
		fullProjPath := proj.Path

		var targetMvnPath string
		for _, mvn := range ui.Config.Mavens {
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

			cmdArgs = strings.Split(targetStep, " ")
		} else {
			binPath = "docker"
			cmdArgs = []string{"build", "-t", strings.ToLower(proj.Name) + ":latest", "."}
		}

		cmd := exec.Command(binPath, cmdArgs...)
		cmd.Dir = fullProjPath

		var targetJDKPath string
		for _, jdk := range ui.Config.JDKs {
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
			return componentmessage.BuildCompleteMsg{SessionID: sID, Err: fmt.Errorf("channel untracked")}
		}

		line, ok := <-activeCh
		if !ok {
			return componentmessage.BuildCompleteMsg{SessionID: sID, Err: nil}
		}
		return componentmessage.BuildLogLineMsg{SessionID: sID, Line: line}
	}
}
