package decorator

import (
	"fmt"
	Strings "strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/ui/color"
)

var (
	Text            = lipgloss.NewStyle().Foreground(color.Text)
	Subtext         = lipgloss.NewStyle().Foreground(color.Subtext0)
	Muted           = lipgloss.NewStyle().Foreground(color.Overlay0)
	Grey            = Text
	DarkGrey        = lipgloss.NewStyle().Foreground(color.Mantle)
	LogWindow       = lipgloss.NewStyle().Background(color.Mantle).Border(lipgloss.RoundedBorder()).BorderForeground(color.Surface1)
	Selected        = lipgloss.NewStyle().Foreground(color.Crust).Background(color.Mauve).Padding(0, 1).Bold(true)
	SelectedSubtle  = lipgloss.NewStyle().Foreground(color.Mauve).Background(color.Surface1).Padding(0, 1).Bold(true)
	Inactive        = lipgloss.NewStyle().Foreground(color.Subtext0).Background(color.Surface0).Padding(0, 1)
	InactivePlain   = lipgloss.NewStyle().Foreground(color.Text).Padding(0, 1)
	Title           = lipgloss.NewStyle().Foreground(color.Mauve).Bold(true)
	Error           = lipgloss.NewStyle().Foreground(color.Red).Bold(true)
	Success         = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	Warning         = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	Info            = lipgloss.NewStyle().Foreground(color.Sky)
	Section         = lipgloss.NewStyle().Foreground(color.Lavender).Bold(true)
	ModalBox        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color.Mauve).Background(color.Base).Padding(1, 2, 1, 2)
	Meta            = lipgloss.NewStyle().Foreground(color.Overlay0).Italic(true)
	Key             = lipgloss.NewStyle().Foreground(color.Yellow).Width(14)
	Description     = lipgloss.NewStyle().Foreground(color.Subtext0)
	Author          = lipgloss.NewStyle().Foreground(color.Subtext1).Align(lipgloss.Center).Height(10)
	TableHeader     = lipgloss.NewStyle().Foreground(color.Blue).Bold(true)
	Mode            = lipgloss.NewStyle().Foreground(color.Peach).Bold(true)
	PanelTitle      = lipgloss.NewStyle().Foreground(color.Lavender).Bold(true)
	BorderPane      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color.Surface1).Padding(1, 2)
	Bold            = lipgloss.NewStyle().Bold(true)
	WhiteSpace      = lipgloss.WithWhitespaceChars(" ")
	ActiveLabel     = Bold.Foreground(color.Yellow)
	UnfocusedBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color.Surface1)
	FocusedBorder   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color.Mauve)
	Modified        = lipgloss.NewStyle().Foreground(color.Yellow).Bold(true)
	Untracked       = lipgloss.NewStyle().Foreground(color.Red).Bold(true)
	Staged          = lipgloss.NewStyle().Foreground(color.Green).Bold(true)
	Clean           = lipgloss.NewStyle().Foreground(color.Teal)
)

// Row Helper wrapper function to draw unified lines
func Row(key, desc string) string {
	return Key.Render(" "+key) + Description.Render(desc) + "\n"
}

func DecorateProfiles(modalBody *Strings.Builder, label string, index int, profiles []config.Profile) {
	if len(profiles) == 0 {
		modalBody.WriteString(fmt.Sprintf("  ❌ No configured %s profiles found in database.\n  Go back and add one first.", label))
	} else {
		for i, profile := range profiles {
			if i == index {
				modalBody.WriteString(Selected.Render(fmt.Sprintf("> %s (%s)", profile.Name, profile.Path)) + "\n")
			} else {
				modalBody.WriteString(Inactive.Render(fmt.Sprintf("  %s", profile.Name)) + "\n")
			}
		}
	}
}
