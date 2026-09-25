package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/renderer"
)

func RenderDockerModal(m *model.UIState) string {
	var body strings.Builder
	body.WriteString(renderer.Title.Render("🐳 Docker Infrastructure Control Center (Ctrl+D)") + "\n")
	body.WriteString("Monitor and orchestrate local microservice containers across your daemon runtime layers.\n")
	body.WriteString(strings.Repeat("─", max(m.WindowWidth-8, 20)) + "\n\n")

	// Print Table Grid Headers
	body.WriteString(fmt.Sprintf(
		"  %-12s %-25s %-20s %-20s\n",
		renderer.TableHeader.Render("CONTAINER ID"), renderer.TableHeader.Render("NAMES"), renderer.TableHeader.Render("IMAGE"), renderer.TableHeader.Render("STATUS"),
	))
	body.WriteString(strings.Repeat("╌", max(m.WindowWidth-8, 20)) + "\n")

	if len(m.DockerContainers) == 0 {
		body.WriteString("  \x1b[90m(No docker container contexts active or discovered on your machine daemon)\x1b[0m\n")
	} else {
		for i, c := range m.DockerContainers {
			statusColor := "\x1b[31m" // Default Red for stopped
			if strings.HasPrefix(strings.ToLower(c.Status), "up") {
				statusColor = "\x1b[32m" // Green for Running
			}

			rowText := fmt.Sprintf(
				"  %-12s %-25s %-20s %s%-20s\x1b[0m",
				c.ID, c.Names, c.Image, statusColor, c.Status,
			)

			// Restrict horizontal lengths to stay within bounds gracefully
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
	body.WriteString("\x1b[226;1m[s] Start Container  |  [t] Stop Container  |  [r] Restart  |  [Esc] Dashboard\x1b[0m\n")

	return lipgloss.Place(m.WindowWidth, m.WindowHeight, lipgloss.Center, lipgloss.Center, renderer.ModalBox.
		Width(m.WindowWidth-4).
		Render(body.String()))
}
