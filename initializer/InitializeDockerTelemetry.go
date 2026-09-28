package initializer

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

type initializerDockerTelemetry struct {
	ui *model.UI
}

func (m *initializerDockerTelemetry) OnAction() tea.Cmd {
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

func InitializeDockerTelemetry(ui *model.UI) Initializer {
	return &initializerDockerTelemetry{ui}
}
