package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/components"
	"github.com/exr462/go-dark/ui/panels"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) View() string {
	switch m.state.ViewState {
	case model.StateHelpModal:
		return components.RenderHelpModal(m.state)
	case model.StateInstaller:
		return components.RenderInstaller(m.state)
	case model.StateGitConfigurationModal:
		return components.RenderGitConfiguration(m.state)
	case model.StateAddProjectModal:
		return components.RenderModal(m.state)
	case model.StateGitOperationsModal:
		return components.RenderGitOpsModal(m.state)
	case model.StateJDKConfigModal:
		return components.RenderJDKConfigModal(m.state)
	case model.StateMavenConfigModal:
		return components.RenderMvnConfigModal(m.state)
	case model.StateBuildModal:
		return components.RenderBuildModal(m.state)
	case model.StateSessionLogsModal:
		return components.RenderSessionLogsModal(m.state)
	case model.StateFuzzyModal:
		return components.RenderFuzzyModal(m.state)
	case model.StateConfigDeckModal:
		return components.RenderConfigDeckModal(m.state)
	case model.StateDockerModal:
		return components.RenderDockerModal(m.state)
	case model.StateDependencyConfigModal: // 👈 ADD THIS CASE
		return components.RenderDependencyModal(m.state)
	case model.StateEditorModal:
		return "Opening external editor..."
	case model.StateDashboard:
		fallthrough
	default:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			panels.RenderMenu(m.state),
			panels.RenderMainBody(m.state),
			panels.RenderFooter(m.state),
		)
	}
}
