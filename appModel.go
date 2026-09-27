package main

import (
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/storage"
)

type appModel struct {
	state           *model.UIState
	contentLoader   storage.ContentLoader
	providerFactory *lsp.ProviderFactory
	maxParallelism  int
}
