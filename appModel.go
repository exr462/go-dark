package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/shell"
	"github.com/exr462/go-dark/storage"
)

type appModel struct {
	ui              *model.UI
	contentLoader   storage.ContentLoader
	providerFactory *lsp.ProviderFactory
	// sessions owns every PTY-backed shell/build session. It may be nil in
	// tests that don't exercise sessions.
	sessions *shell.Manager
}

// listenSessions starts (or continues) the single session event loop.
func (m *appModel) listenSessions() tea.Cmd {
	if m.sessions == nil {
		return nil
	}
	return m.sessions.Listen()
}
