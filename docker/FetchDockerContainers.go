package docker

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func FetchDockerContainers() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("docker", "ps", "-a", "--format", "{{.ID}},{{.Names}},{{.Image}},{{.Status}},{{.Ports}}")
		out, err := cmd.Output()
		if err != nil {
			return DockerContainersMsg{}
		}

		var list []DockerContainer
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
				list = append(list, DockerContainer{
					ID:     parts[0],
					Names:  parts[1],
					Image:  parts[2],
					Status: parts[3],
					Ports:  ports,
				})
			}
		}
		return DockerContainersMsg(list)
	}
}
