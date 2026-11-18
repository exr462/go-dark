// Package runner builds the processes that run a project's configured
// commands (see config.ResolveCommand) with the right toolchain environment.
package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/exr462/go-dark/config"
)

// Command returns a process running shellCmd through the platform shell in
// the project's directory. For JVM projects the assigned JDK/Maven profiles
// are exported (JAVA_HOME, MAVEN_HOME, M2_HOME) and put first on PATH.
func Command(ctx context.Context, cfg config.Config, p config.Project, shellCmd string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", shellCmd)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", shellCmd)
	}
	cmd.Dir = p.Path
	cmd.Env = Env(cfg, p)
	return cmd
}

// Env is the environment a project's commands run with.
func Env(cfg config.Config, p config.Project) []string {
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if !config.UsesJVM(p) {
		return env
	}

	jdkPath := profilePath(cfg.JDKs, p.JDKName)
	mvnPath := profilePath(cfg.Mavens, p.MavenName)
	var prefixes []string
	if jdkPath != "" {
		env = append(env, "JAVA_HOME="+jdkPath)
		prefixes = append(prefixes, filepath.Join(jdkPath, "bin"))
	}
	if mvnPath != "" {
		env = append(env, "MAVEN_HOME="+mvnPath, "M2_HOME="+mvnPath)
		prefixes = append(prefixes, filepath.Join(mvnPath, "bin"))
	}
	if len(prefixes) > 0 {
		sep := string(os.PathListSeparator)
		env = append(env, fmt.Sprintf("PATH=%s%s%s", strings.Join(prefixes, sep), sep, os.Getenv("PATH")))
	}
	return env
}

func profilePath(profiles []config.Profile, name string) string {
	if name == "" {
		return ""
	}
	for _, pr := range profiles {
		if pr.Name == name {
			return pr.Path
		}
	}
	return ""
}
