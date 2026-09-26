package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) executeActiveMenuAction() tea.Cmd {
	if len(m.state.Config.Projects) == 0 {
		return nil
	}
	return func() tea.Msg {
		return model.StatusMsg("Manual workspace isolation completed.")
	}
}
