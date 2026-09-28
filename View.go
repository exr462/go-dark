package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/component"
	"github.com/exr462/go-dark/ui/panels"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) View() string {
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
	case model.StateDashboard:
		fallthrough
	default:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			panels.RenderMenu(m.ui),
			panels.RenderMainBody(m.ui),
			panels.RenderFooter(m.ui),
		)
	}
}
