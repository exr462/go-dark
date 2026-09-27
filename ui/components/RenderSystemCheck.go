package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/decorator"
)

const (
	MochaBase    = "#1e1e2e" // Dark background
	MochaSurface = "#313244" // Lighter background for tags/boxes
	MochaOverlay = "#6c7086" // Dim grey for borders and brackets
	MochaSubtext = "#a6adc8" // Elegant silver/grey text
	MochaGreen   = "#a6e3a1" // Soft matrix green for success
	MochaTeal    = "#94e2d5" // Bright teal for titles/tags
)

var (
	// Tag matches Catppuccin's dark surface with a bright Teal accent label
	tagStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(MochaSurface)).
			Foreground(lipgloss.Color(MochaTeal)).
			Bold(true).
			Padding(0, 1)

	// Meta text uses a soft overlay grey
	metaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(MochaOverlay))

	// Package names use Mocha's premium silver/white subtext color
	pkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(MochaSubtext)).Bold(true)

	// Brackets and dividers use the muted overlay shade
	bracketsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(MochaOverlay))

	// Done success messages shine in Catppuccin's signature pastel Green
	doneStyle = lipgloss.NewStyle().Margin(1, 2).Foreground(lipgloss.Color(MochaGreen)).Bold(true)
)

func RenderSystemCheck(ui *model.UI) string {
	if ui.Done {
		// In Bubble Tea v2, return views wrapped explicitly using tea.NewView
		return doneStyle.Render(fmt.Sprintf("Done! Installed %d packages.\n", len(ui.Prechecks)))
	}

	// 1. Establish the fixed container modal width boundaries
	modalWidth := max(min(ui.WindowWidth-4, 85),
		// Edge safeguard for compressed screen layouts
		20)

	// Usable text capacity layout width inside the box border lines
	innerWidth := modalWidth - 4

	var modalBody strings.Builder

	modalBody.WriteString(tagStyle.Render("SYSTEM") + metaStyle.Render(" go-dark // build-v0.0.3") + "\n")
	modalBody.WriteString(metaStyle.Render(" STATUS : PREFLIGHT_TASKS_STAGE") + "\n")
	modalBody.WriteString(metaStyle.Render(" ───────────────────────────────────────") + "\n\n")

	// Pass the restricted modal boundary down into our flex gap builder
	modalBody.WriteString(view(ui, innerWidth) + "\n")

	// 2. Build the final screen space layout string
	renderedOutput := lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(modalWidth).Render(modalBody.String()),
	)

	// Return structural v2 view type
	return renderedOutput
}

// Sub-components can still return a raw string to make string composition easy
func view(ui *model.UI, width int) string {
	totalSteps := len(ui.Prechecks)
	w := lipgloss.Width(fmt.Sprintf("%d", totalSteps))

	pkgCount := bracketsStyle.Render("[ ") +
		fmt.Sprintf("%*d/%*d", w, ui.Index+1, w, totalSteps) +
		bracketsStyle.Render(" ]")

	spin := ui.Spinner.View() + " "

	percentVal := (float64(ui.Index) / float64(totalSteps)) * 100
	percentStr := fmt.Sprintf(" %5.1f%%", percentVal)

	// 1. STABILIZE THE PROGRESS BAR WIDTH
	//    Instead of calculating based on the text, give the progress bar a fixed wide allocation
	//    (e.g., 30 or 35 characters wide).
	ui.Progress.Width = 35
	prog := ui.Progress.View()

	// 2. STABILIZE THE TEXT BOUNDARY
	//    Calculate how many remaining cells are left over for your "init: package_name" string
	rightSideWidth := lipgloss.Width(prog + percentStr + " " + pkgCount)
	leftSideOverhead := lipgloss.Width(spin + "init: ")

	cellsAvailForText := max(0, width-leftSideOverhead-rightSideWidth-2)

	// Clamp the package name style to that exact size so it never pushes the progress bar
	pkgName := pkgNameStyle.Render(ui.Prechecks[ui.Index].Name)
	info := lipgloss.NewStyle().MaxWidth(cellsAvailForText).Render("init: " + pkgName)

	// 3. ELASTIC GAP CORES
	//    The middle spacer now dynamically absorbs any size changes from the package text!
	cellsRemaining := max(0, width-lipgloss.Width(spin+info+prog+percentStr+" "+pkgCount))
	gap := strings.Repeat(" ", cellsRemaining)

	// Output order: spinner -> file name -> elastic spacer -> stable wide progress indicators
	return spin + info + gap + prog + percentStr + " " + pkgCount
}
