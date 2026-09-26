package main

import "github.com/exr462/go-dark/lsp"

var factory = lsp.NewProviderFactory([]lsp.LanguageProvider{
	lsp.GoProvider{},
	lsp.KotlinProvider{},
	lsp.JavaProvider{},
})
