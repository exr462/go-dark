package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/renderer"
)

var (
	dockerRunningStyle = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	dockerStoppedStyle = lipgloss.NewStyle().Foreground(color.Red)
	dockerHintStyle    = lipgloss.NewStyle().Foreground(color.Overlay0)
	dockerKeyHint      = lipgloss.NewStyle().Foreground(color.Yellow)
)

func RenderDockerModal(m *model.UIState) string {
	var body strings.Builder
	body.WriteString(renderer.Title.Render("🐳 Docker Infrastructure Control Center (Ctrl+D)") + "\n")
	body.WriteString(renderer.Description.Render("Monitor and orchestrate local microservice containers across your daemon runtime layers.") + "\n")
	body.WriteString(strings.Repeat("─", max(m.WindowWidth-8, 20)) + "\n\n")

	// Print Table Grid Headers
	body.WriteString(fmt.Sprintf(
		"  %-12s %-25s %-20s %-20s\n",
		renderer.TableHeader.Render("CONTAINER ID"), renderer.TableHeader.Render("NAMES"), renderer.TableHeader.Render("IMAGE"), renderer.TableHeader.Render("STATUS"),
	))
	body.WriteString(strings.Repeat("╌", max(m.WindowWidth-8, 20)) + "\n")

	if len(m.DockerContainers) == 0 {
		body.WriteString(renderer.Meta.Render("  (No docker container contexts active or discovered on your machine daemon)") + "\n")
	} else {
		for i, c := range m.DockerContainers {
			statusFormatted := dockerStoppedStyle.Render(c.Status)
			if strings.HasPrefix(strings.ToLower(c.Status), "up") {
				statusFormatted = dockerRunningStyle.Render(c.Status)
			}

			rowText := fmt.Sprintf(
				"  %-12s %-25s %-20s %-20s",
				c.ID, c.Names, c.Image, statusFormatted,
			)

			if len(rowText) > m.WindowWidth-6 {
				rowText = rowText[:m.WindowWidth-9] + "..."
			}

			if i == m.SelectedDockerRow {
				body.WriteString(renderer.Selected.Render("> "+rowText) + "\n")
			} else {
				body.WriteString(renderer.Inactive.Render(rowText) + "\n")
			}
		}
	}

	body.WriteString("\n" + strings.Repeat("─", max(m.WindowWidth-8, 20)) + "\n")
	body.WriteString(dockerHintStyle.Render(fmt.Sprintf(
		"[%s] Start | [%s] Stop | [%s] Restart | [%s] Dashboard",
		dockerKeyHint.Render("s"),
		dockerKeyHint.Render("t"),
		dockerKeyHint.Render("r"),
		dockerKeyHint.Render("Esc"),
	)) + "\n")

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		renderer.ModalBox.Width(m.WindowWidth-4).Render(body.String()),
		renderer.WhiteSpace,
		lipgloss.WithWhitespaceForeground(color.Crust),
	)
}
