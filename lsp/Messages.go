package lsp

type FileLoadedMsg struct {
	Path     string
	Content  string
	Provider LanguageProvider
}
type FileErrorMsg struct{ Err error }

type ExecutionResultMsg struct{ Output string }
