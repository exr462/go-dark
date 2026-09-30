package docker

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/state"
)

func DockerAction(containerID, action string) tea.Cmd {
	return func() tea.Msg {
		_ = exec.Command("docker", action, containerID).Run()
		return state.StatusMsg(fmt.Sprintf("✅ Successfully executed: docker %s %s", action, containerID))
	}
}
