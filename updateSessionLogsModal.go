package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateSessionLogsModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	default:
		if msg.String() >= "1" && msg.String() <= "9" {
			runes := []rune(msg.String())
			if len(runes) > 0 {
				targetID := int(runes[0] - '0')
				m.state.ViewingSessionID = targetID
				return m, nil
			}
		}
	}
	return m, nil
}
