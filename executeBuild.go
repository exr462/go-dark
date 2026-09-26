package main

import (
	"context"
	"os/exec"
	"strings"
)

// Underlying execution function parsing input logic
func executeBuild(ctx context.Context, dir string, buildArgs string) error {
	// Clean and parse build parameters safely
	args := []string{"clean"}
	for arg := range strings.SplitSeq(buildArgs, " ") {
		if trimmed := strings.TrimSpace(arg); trimmed != "" {
			args = append(args, trimmed)
		}
	}

	// Assuming a Maven environment given your project logs
	cmd := exec.CommandContext(ctx, "mvn", args...)
	cmd.Dir = dir

	// Capture output or pipe to a tracker log if debugging is needed
	err := cmd.Run()
	return err
}
