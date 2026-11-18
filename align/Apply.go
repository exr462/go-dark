// Package align writes version alignment edits into the workspace repos,
// one local commit per repo on a dedicated branch. It never pushes.
package align

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/exr462/go-dark/maven"
	"github.com/exr462/go-dark/model"
)

// Branch is where alignment commits go in every touched repo.
const Branch = "go-dark/align-versions"

// CommitMessage is used for every alignment commit.
const CommitMessage = "chore: align workspace versions (go-dark)"

// Identity is used for commits when the repo has no git user configured.
type Identity struct{ Name, Email string }

// Result reports what happened in one repo.
type Result = model.AlignResult

// Apply edits and commits each repo independently: a repo that can't be
// changed safely is skipped and reported, the others go ahead.
func Apply(changes map[string][]maven.Change, dirs map[string]string, who Identity) []Result {
	ids := make([]string, 0, len(changes))
	for id := range changes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	results := make([]Result, 0, len(ids))
	for _, id := range ids {
		results = append(results, applyRepo(id, dirs[id], changes[id], who))
	}
	return results
}

func applyRepo(id, dir string, changes []maven.Change, identity Identity) Result {
	res := Result{ProjectID: id, Dir: dir}
	if dir == "" {
		res.Err = fmt.Errorf("unknown project directory")
		return res
	}
	who := identityArgs(dir, identity)
	files := uniqueFiles(changes)
	rel := make([]string, len(files))
	for i, f := range files {
		rel[i] = strings.TrimPrefix(strings.TrimPrefix(f, dir), string(os.PathSeparator))
	}

	status, err := git(dir, who, append([]string{"status", "--porcelain", "--"}, rel...)...)
	if err != nil {
		res.Err = err
		return res
	}
	if strings.TrimSpace(status) != "" {
		res.Skipped = "uncommitted changes in " + strings.Join(rel, ", ")
		return res
	}

	current, err := git(dir, who, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		res.Err = err
		return res
	}
	if strings.TrimSpace(current) != Branch {
		if _, err := git(dir, who, "rev-parse", "--verify", "--quiet", "refs/heads/"+Branch); err == nil {
			res.Skipped = fmt.Sprintf("branch %s already exists (but isn't checked out); merge or delete it first", Branch)
			return res
		}
		if _, err := git(dir, who, "switch", "--quiet", "-c", Branch); err != nil {
			res.Err = err
			return res
		}
	}

	if _, err := maven.Apply(changes); err != nil {
		res.Err = err
		return res
	}
	res.Files = files
	// The pathspec form commits only these files, even if other changes
	// are staged in the repo.
	if _, err := git(dir, who, append([]string{"commit", "--quiet", "-m", CommitMessage, "--"}, rel...)...); err != nil {
		res.Err = err
		return res
	}
	hash, err := git(dir, who, "rev-parse", "--short", "HEAD")
	if err != nil {
		res.Err = err
		return res
	}
	res.Commit = strings.TrimSpace(hash)
	return res
}

func uniqueFiles(changes []maven.Change) []string {
	seen := map[string]bool{}
	var files []string
	for _, c := range changes {
		if !seen[c.File] {
			seen[c.File] = true
			files = append(files, c.File)
		}
	}
	sort.Strings(files)
	return files
}

// identityArgs passes go-dark's git identity only when the repo/user has
// none configured, so a configured identity always wins.
func identityArgs(dir string, who Identity) []string {
	var args []string
	if who.Name != "" && configMissing(dir, "user.name") {
		args = append(args, "-c", "user.name="+who.Name)
	}
	if who.Email != "" && configMissing(dir, "user.email") {
		args = append(args, "-c", "user.email="+who.Email)
	}
	return args
}

// git runs a non-interactive git command in dir.
func git(dir string, identity []string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	full := append(append([]string(nil), identity...), args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func configMissing(dir, key string) bool {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir
	out, err := cmd.Output()
	return err != nil || strings.TrimSpace(string(out)) == ""
}
