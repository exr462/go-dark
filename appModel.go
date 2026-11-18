package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/storage"
)

type appModel struct {
	ui              *model.UI
	contentLoader   storage.ContentLoader
	providerFactory *lsp.ProviderFactory
}

// listenSessions starts (or continues) the single session event loop.
// The manager lives on model.UI (Shells) so screens can show sessions; it
// may be nil in tests that don't exercise sessions.
func (m *appModel) listenSessions() tea.Cmd {
	if m.ui.Shells == nil {
		return nil
	}
	return m.ui.Shells.Listen()
}
