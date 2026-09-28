package storage

import "os"

// ContentLoader defines the behavior for reading and writing files using strings.
type ContentLoader interface {
	WriteContent(content string, path string) error
	GetContent(path string) (string, error)
}

type localFileLoader struct{}

// WriteContent accepts a string and handles the byte conversion internally.
func (l *localFileLoader) WriteContent(content string, path string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// GetContent reads the file and converts the bytes to a string before returning.
func (l *localFileLoader) GetContent(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// NewContentLoader initializes and returns the loader.
func NewContentLoader() ContentLoader {
	return &localFileLoader{}
}
