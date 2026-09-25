package renderer

import (
	"fmt"
	Strings "strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/ui/color"
)

var (
	Grey            = lipgloss.NewStyle().Foreground(color.Grey)
	DarkGrey        = lipgloss.NewStyle().Foreground(color.DarkGrey)
	LogWindow       = DarkGrey.Border(lipgloss.NormalBorder()).BorderForeground(color.DarkishGrey)
	Selected        = DarkGrey.Background(lipgloss.Color("46")).Padding(0, 1).Bold(true)
	Inactive        = Grey.Background(color.MidGrey).Padding(0, 1)
	Title           = lipgloss.NewStyle().Foreground(color.Orange).Bold(true)
	Error           = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	Section         = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	ModalBox        = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(color.ModalBorderColor).Background(color.ModalBackground).Padding(1, 2, 1, 2)
	Meta            = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	Key             = lipgloss.NewStyle().Foreground(color.Yellow).Width(14)
	Description     = Grey
	Author          = lipgloss.NewStyle().Foreground(lipgloss.Color("#dcdcdd")).Align(lipgloss.Center).Height(10)
	TableHeader     = lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
	Mode            = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	PanelTitle      = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	BorderPane      = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(color.DarkishGrey).Padding(1, 2)
	Bold            = lipgloss.NewStyle().Bold(true)
	WhiteSpace      = lipgloss.WithWhitespaceChars("░")
	ActiveLabel     = Bold.Foreground(color.Yellow)
	UnfocusedBorder = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(color.DarkishGrey)
	FocusedBorder   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(color.Pink)
	Modified        = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true) // Yellow
	Untracked       = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // Red
	Staged          = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)  // Green
	Clean           = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
)

// Row Helper wrapper function to draw unified lines
func Row(key, desc string) string {
	return Key.Render(" "+key) + Description.Render(desc) + "\n"
}

func DecorateProfiles(modalBody Strings.Builder, label string, index int, profiles []config.Profile) {
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
