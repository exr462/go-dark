package env

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kube"
)

// CLI is `go-dark env <install|start|stop|uninstall|status> ...`. The
// installer screen runs the lifecycle actions through it, as sessions, so
// each one is a plain process with an inspectable log that also works where
// there is no PTY (Windows); users can run the same commands by hand.
//
// The environment comes from a manifest file (--manifest, with --workspace
// and --project-dir for the services' config) or from the config's teams
// (--team). It returns the process exit code.
func CLI(args []string, stdout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, cliUsage)
		return 2
	}
	action, args := args[0], args[1:]
	fs := flag.NewFlagSet("go-dark env "+action, flag.ContinueOnError)
	fs.SetOutput(stdout)
	if action == "import-image" {
		return importImage(fs, args, stdout)
	}
	manifest := fs.String("manifest", "", "team manifest (go-dark.json) declaring the environment")
	team := fs.String("team", "", "subscribed team whose environment to use (reads config.json)")
	workspace := fs.String("workspace", "", "workspace holding the services (default: config.json's)")
	dir := fs.String("dir", "", "install directory (default: ~/.local/share/go-dark/env/<team>)")
	dbPassword := fs.String("db-password", "", "database password (default: "+DefaultDBPass+")")
	volumes := fs.Bool("volumes", false, "uninstall: remove the cluster and database volumes too")
	var projectDirs stringList
	fs.Var(&projectDirs, "project-dir", "service checkout whose config provides database users (repeatable)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	e, in, err := resolve(*manifest, *team, *workspace, *dir, *dbPassword, projectDirs)
	if err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch action {
	case "install", "repair":
		err = Install(ctx, e, in, stdout)
	case "start":
		err = Start(ctx, e, in, stdout)
	case "stop":
		err = Stop(ctx, e, in, stdout)
	case "uninstall":
		err = Uninstall(ctx, e, in, *volumes, stdout)
	case "status":
		printStatus(stdout, e, Describe(e, in.Dir))
	default:
		fmt.Fprintln(stdout, cliUsage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stdout, "\nFAIL %v\n", err)
		return 1
	}
	return 0
}

const cliUsage = `usage: go-dark env <install|start|stop|uninstall|status> (--manifest go-dark.json [--workspace DIR] [--project-dir DIR]... | --team NAME) [--dir DIR] [--db-password PW] [--volumes]
       go-dark env import-image --container NAME IMAGE`

// importImage is `go-dark env import-image --container NAME IMAGE`: the
// deploy's side-load of a locally built image into the cluster's containerd,
// as a plain process so it runs as a session on every platform.
func importImage(fs *flag.FlagSet, args []string, stdout io.Writer) int {
	container := fs.String("container", "", "the environment's rancher container")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *container == "" || fs.NArg() != 1 {
		fmt.Fprintln(stdout, cliUsage)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := kube.ImportImage(ctx, *container, fs.Arg(0), stdout); err != nil {
		fmt.Fprintf(stdout, "\nFAIL %v\n", err)
		return 1
	}
	return 0
}

type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, ",") }
func (l *stringList) Set(v string) error { *l = append(*l, v); return nil }

// resolve builds the Env and Inputs from the flags.
func resolve(manifest, team, workspace, dir, dbPassword string, projectDirs []string) (Env, Inputs, error) {
	var te config.TeamEnvironment
	switch {
	case manifest != "":
		m, err := config.LoadManifest(manifest)
		if err != nil {
			return Env{}, Inputs{}, err
		}
		if m.Environment == nil {
			return Env{}, Inputs{}, fmt.Errorf("%s declares no environment", manifest)
		}
		te = config.TeamEnvironment{Team: m.Team, Environment: *m.Environment}
		if len(projectDirs) == 0 && workspace != "" {
			for _, p := range m.Projects {
				if p.Deployable {
					projectDirs = append(projectDirs, config.ResolvePath(workspace, p.Name))
				}
			}
		}
	case team != "":
		cfg, _ := config.LoadConfig()
		found := false
		for _, candidate := range cfg.Environments {
			if candidate.Team == team {
				te, found = candidate, true
			}
		}
		if !found {
			return Env{}, Inputs{}, fmt.Errorf("no subscribed team %q declares an environment", team)
		}
		if dbPassword == "" {
			dbPassword = cfg.LocalCluster.Database.Password
		}
		if len(projectDirs) == 0 {
			for _, p := range cfg.Projects {
				if p.Source == team && p.Deployable && p.Fetched {
					projectDirs = append(projectDirs, p.Path)
				}
			}
		}
	default:
		return Env{}, Inputs{}, errors.New("--manifest or --team is required")
	}
	e := New(te, dbPassword)
	if dir == "" {
		var err error
		if dir, err = e.InstallDir(); err != nil {
			return Env{}, Inputs{}, err
		}
	}
	return e, Inputs{Dir: dir, ProjectDirs: projectDirs}, nil
}

func printStatus(w io.Writer, e Env, st Status) {
	fmt.Fprintf(w, "environment %s of team %s\n", e.Name, e.Team)
	fmt.Fprintf(w, "  installed   %v\n", st.Installed)
	fmt.Fprintf(w, "  container   %s (%s)\n", e.Container(), st.Container)
	fmt.Fprintf(w, "  k3s         ready=%v\n", st.K3sReady)
	if st.EtcdBytes > 0 {
		fmt.Fprintf(w, "  etcd        %s\n", HumanBytes(st.EtcdBytes))
	}
	for _, kind := range st.Pressure {
		fmt.Fprintf(w, "  node        %s: %s\n", kind, PressureHint(kind))
	}
	fmt.Fprintf(w, "  ingress     %s\n  database    %s\n", st.Ingress, st.Database)
	fmt.Fprintf(w, "  releases    %s\n", strings.Join(st.Releases, ", "))
	fmt.Fprintf(w, "  urls        %s/<service>  https://%s\n", e.APIURL(), e.RancherHost())
	if len(st.Missing) > 0 {
		fmt.Fprintf(w, "  hosts file  missing %s -> add: %s\n", strings.Join(st.Missing, ", "), e.HostsLine())
	}
}

// Argv is the command line the installer screen runs for an action, going
// through this executable.
func Argv(exe, action string, manifestPath string, in Inputs, dbPassword string, volumes bool) []string {
	argv := []string{exe, "env", action, "--manifest", manifestPath, "--dir", in.Dir}
	for _, d := range in.ProjectDirs {
		argv = append(argv, "--project-dir", d)
	}
	if dbPassword != "" {
		argv = append(argv, "--db-password", dbPassword)
	}
	if volumes {
		argv = append(argv, "--volumes")
	}
	return argv
}

// ManifestPath is the cached manifest file of a subscribed team (by its
// manifest team name).
func ManifestPath(teams []config.Team, team string) (string, error) {
	for _, t := range teams {
		if t.Manifest == team {
			dir, err := config.TeamDir(t)
			if err != nil {
				return "", err
			}
			return filepath.Join(dir, config.ManifestFileName), nil
		}
	}
	return "", fmt.Errorf("team %q is not subscribed", team)
}
