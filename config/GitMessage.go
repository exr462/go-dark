package config

type GitStatusLoadedMsg string
type GitStatusErrorMsg error
type GitBranchesLoadedMsg []string
type GitBranchesErrorMsg error
type GitCheckoutCompleteMsg struct {
	Output string
	Err    error
}
