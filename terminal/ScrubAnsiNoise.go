package terminal

import (
	"regexp"
	"strings"
)

// ScrubAnsiNoise cleanly sweeps artifact layouts before pushing to layout streams
func ScrubAnsiNoise(input string) string {
	reg := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return strings.TrimSpace(reg.ReplaceAllString(input, ""))
}
