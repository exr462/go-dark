package docker

import (
	"fmt"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/state"
)

func DockerAction(containerID, action string) tea.Cmd {
	return func() tea.Msg {
		_ = exec.Command("docker", action, containerID).Run()
		return state.StatusMsg(fmt.Sprintf("✅ Successfully executed: docker %s %s", action, containerID))
	}
}
