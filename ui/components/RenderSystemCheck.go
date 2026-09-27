package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/decorator"
)

var (
	currentPkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle           = lipgloss.NewStyle().Margin(1, 2)
)

func RenderSystemCheck(state *model.UIState) string {
	if state.Done {
		// In Bubble Tea v2, return views wrapped explicitly using tea.NewView
		return doneStyle.Render(fmt.Sprintf("Done! Installed %d packages.\n", len(state.Prechecks)))
	}

	// 1. Establish the fixed container modal width boundaries
	modalWidth := min(state.WindowWidth-4, 85)
	if modalWidth < 20 { // Edge safeguard for compressed screen layouts
		modalWidth = 20
	}

	// Usable text capacity layout width inside the box border lines
	innerWidth := modalWidth - 4

	var modalBody strings.Builder
	modalBody.WriteString(decorator.Title.Render("🛸 Precheck flight") + "\n\n")

	// Pass the restricted modal boundary down into our flex gap builder
	modalBody.WriteString(view(state, innerWidth) + "\n")

	// 2. Build the final screen space layout string
	renderedOutput := lipgloss.Place(
		state.WindowWidth, state.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(modalWidth).Render(modalBody.String()),
	)

	// Return structural v2 view type
	return renderedOutput
}

// Sub-components can still return a raw string to make string composition easy
func view(state *model.UIState, width int) string {
	totalSteps := len(state.Prechecks)
	w := lipgloss.Width(fmt.Sprintf("%d", totalSteps))

	// Match total steps accurately against your setup tracking index bounds
	pkgCount := fmt.Sprintf(" %*d/%*d", w, state.Index+1, w, totalSteps)

	spin := state.Spinner.View() + " "
	prog := state.Progress.View()

	// Calculate text space limits explicitly against the local modal container inner width
	cellsAvail := max(0, width-lipgloss.Width(spin+prog+pkgCount))

	pkgName := currentPkgNameStyle.Render(state.Prechecks[state.Index].Name)
	info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render(pkgName)

	// Build a precise elastic gap inside innerWidth bounds so elements do not break columns
	cellsRemaining := max(0, width-lipgloss.Width(spin+info+prog+pkgCount))
	gap := strings.Repeat(" ", cellsRemaining)

	return spin + info + gap + prog + pkgCount
}
