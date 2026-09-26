package lsp

import (
	"path/filepath"
	"strings"
)

type LanguageRegistry struct {
	providers map[string]LanguageProvider
}

func NewLanguageRegistry() *LanguageRegistry {
	reg := &LanguageRegistry{providers: make(map[string]LanguageProvider)}

	java := JavaProvider{}
	for _, ext := range java.Extensions() {
		reg.providers[ext] = java
	}

	goprov := GoProvider{}
	for _, ext := range goprov.Extensions() {
		reg.providers[ext] = goprov
	}

	return reg
}

func (r *LanguageRegistry) GetProviderForFile(path string) (LanguageProvider, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	lp, exists := r.providers[ext]
	return lp, exists
}
