package initializer

import tea "github.com/charmbracelet/bubbletea"

type Initializer interface {
	OnAction() tea.Cmd
}
