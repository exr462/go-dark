package components

import (
	"fmt"
	Strings "strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/config"
)

// TODO fix colors for every screen
const (
	ModalBackground  = lipgloss.Color("#1e1e2e")
	ModalBorderColor = lipgloss.Color("#1d1d2c")
	Orange           = lipgloss.Color("208")
	DarkGrey         = lipgloss.Color("0")
	Grey             = lipgloss.Color("250")
	DarkerGrey       = lipgloss.Color("233")
	DarkishGrey      = lipgloss.Color("240")
	MidGrey          = lipgloss.Color("237")
	Yellow           = lipgloss.Color("229")
	Pink             = lipgloss.Color("205")
)

var (
	greyStyle        = lipgloss.NewStyle().Foreground(Grey)
	darkGreyStyle    = lipgloss.NewStyle().Foreground(DarkGrey)
	logWindowStyle   = darkGreyStyle.Border(lipgloss.NormalBorder()).BorderForeground(DarkishGrey)
	selectedStyle    = darkGreyStyle.Background(lipgloss.Color("46")).Padding(0, 1).Bold(true)
	inactiveStyle    = greyStyle.Background(MidGrey).Padding(0, 1)
	TitleStyle       = lipgloss.NewStyle().Foreground(Orange).Bold(true)
	sectionStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	ModalBox         = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(ModalBorderColor).Background(ModalBackground).Padding(1, 2, 1, 2)
	metaStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	keyStyle         = lipgloss.NewStyle().Foreground(Yellow).Width(14)
	descStyle        = greyStyle
	authorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#dcdcdd")).Align(lipgloss.Center).Height(10)
	tableHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
	modeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	panelTitleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	borderPaneStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(DarkishGrey).Padding(1, 2)
	boldStyle        = lipgloss.NewStyle().Bold(true)
	whiteSpace       = lipgloss.WithWhitespaceChars("░")
	activeLabelStyle = boldStyle.Foreground(Yellow)
	UnfocusedBorder  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(DarkishGrey)
	FocusedBorder    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(Pink)
)

// Helper wrapper function to draw unified lines
func row(key, desc string) string {
	return keyStyle.Render(" "+key) + descStyle.Render(desc) + "\n"
}

func decorateProfiles(modalBody Strings.Builder, label string, index int, profiles []config.Profile) {
	if len(profiles) == 0 {
		modalBody.WriteString(fmt.Sprintf("  ❌ No configured %s profiles found in database.\n  Go back and add one first.", label))
	} else {
		for i, profile := range profiles {
			if i == index {
				modalBody.WriteString(selectedStyle.Render(fmt.Sprintf("> %s (%s)", profile.Name, profile.Path)) + "\n")
			} else {
				modalBody.WriteString(inactiveStyle.Render(fmt.Sprintf("  %s", profile.Name)) + "\n")
			}
		}
	}
}
