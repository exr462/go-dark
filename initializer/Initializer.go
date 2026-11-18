package initializer

import tea "charm.land/bubbletea/v2"

type Initializer interface {
	OnAction() tea.Cmd
}
