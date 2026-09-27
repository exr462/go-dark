package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/components"
	"github.com/exr462/go-dark/ui/panels"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) View() string {
	switch m.ui.ViewState {
	case model.StateHelpModal:
		return components.RenderHelpModal(m.ui)
	case model.StateGitConfigurationModal:
		return components.RenderGitConfiguration(m.ui)
	case model.StateGitOperationsModal:
		return components.RenderGitOpsModal(m.ui)
	case model.StateJDKConfigModal:
		return components.RenderJDKConfigModal(m.ui)
	case model.StateMavenConfigModal:
		return components.RenderMvnConfigModal(m.ui)
	case model.StateBuildModal:
		return components.RenderBuildModal(m.ui)
	case model.StateSessionLogsModal:
		return components.RenderSessionLogsModal(m.ui)
	case model.StateFuzzyModal:
		return components.RenderFuzzyModal(m.ui)
	case model.StateConfigDeckModal:
		return components.RenderConfigDeckModal(m.ui)
	case model.StateDockerModal:
		return components.RenderDockerModal(m.ui)
	case model.StateDependencyConfigModal: // 👈 ADD THIS CASE
		return components.RenderDependencyModal(m.ui)
	case model.StateShortcutConfigurationModal:
		return components.RenderShortcutConfigurationModal(m.ui)
	case model.StateEditorModal:
		return components.RenderOpenEditor(m.ui)
	case model.StateSystemCheckModal:
		return components.RenderSystemCheck(m.ui)
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
