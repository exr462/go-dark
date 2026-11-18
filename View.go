package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/component"
	"github.com/exr462/go-dark/ui/panel"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) render() string {
	switch m.ui.ViewState {
	case model.StateHelpModal:
		return component.RenderHelpModal(m.ui)
	case model.StateGitConfigurationModal:
		return component.RenderGitConfiguration(m.ui)
	case model.StateGitOperationsModal:
		return component.RenderGitOpsModal(m.ui)
	case model.StateJDKConfigModal:
		return component.RenderJDKConfigModal(m.ui)
	case model.StateMavenConfigModal:
		return component.RenderMvnConfigModal(m.ui)
	case model.StateBuildModal:
		return component.RenderBuildModal(m.ui)
	case model.StateSessionLogsModal:
		return component.RenderSessionLogsModal(m.ui)
	case model.StateFuzzyModal:
		return component.RenderFuzzyModal(m.ui)
	case model.StateConfigDeckModal:
		return component.RenderConfigDeckModal(m.ui)
	case model.StateDockerModal:
		return component.RenderDockerModal(m.ui)
	case model.StateDependencyConfigModal: // 👈 ADD THIS CASE
		return component.RenderDependencyModal(m.ui)
	case model.StateShortcutConfigurationModal:
		return component.RenderShortcutConfigurationModal(m.ui)
	case model.StateEditorModal:
		return component.RenderOpenEditor(m.ui)
	case model.StateSystemCheckModal:
		return component.RenderSystemCheck(m.ui)
	case model.StateDeployModal:
		return component.RenderDeployModal(m.ui)
	case model.StateTeamsModal:
		return component.RenderTeamsModal(m.ui)
	case model.StateProjectsModal:
		return component.RenderProjectsModal(m.ui)
	case model.StateDashboard:
		fallthrough
	default:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			panel.RenderMenu(m.ui),
			panel.RenderMainBody(m.ui),
			panel.RenderFooter(m.ui),
		)
	}
}

// View wraps the rendered screen in a Bubble Tea v2 view, which now owns
// terminal features such as the alternate screen.
//
//goland:noinspection GoMixedReceiverTypes
func (m *appModel) View() tea.View {
	if m.ui.ViewState == model.StateTerminalCockpit && m.ui.Shells != nil {
		content, cursor := component.RenderTerminalExecution(m.ui, m.ui.Shells)
		v := tea.NewView(content)
		v.AltScreen = true
		v.Cursor = cursor
		return v
	}
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}
