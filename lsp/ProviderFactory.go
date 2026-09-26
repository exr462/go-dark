package lsp

import (
	"path/filepath"
	"strings"
)

// ProviderFactory manages and resolves the correct provider based on file names or extensions
type ProviderFactory struct {
	providers map[string]LanguageProvider
}

// NewProviderFactory registers your supported language providers
func NewProviderFactory(available []LanguageProvider) *ProviderFactory {
	registry := make(map[string]LanguageProvider)

	for _, provider := range available {
		for _, ext := range provider.Extensions() {
			// Ensure extensions are normalized (lowercase and prefixed with a dot)
			normalizedExt := strings.ToLower(ext)
			if !strings.HasPrefix(normalizedExt, ".") {
				normalizedExt = "." + normalizedExt
			}
			registry[normalizedExt] = provider
		}
	}

	return &ProviderFactory{providers: registry}
}

// GetProvider returns the matching provider for a file path, or the DefaultProvider if none match
func (f *ProviderFactory) GetProvider(filePath string) LanguageProvider {
	ext := strings.ToLower(filepath.Ext(filePath))

	if provider, found := f.providers[ext]; found {
		return provider
	}

	return DefaultProvider{}
}
