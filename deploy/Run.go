package deploy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/exr462/go-dark/kube"
	"github.com/exr462/go-dark/shell"
)

// TagShip marks the sessions of a deployment.
const TagShip = "ship"

// StepState is the state of one step of one target.
type StepState int

const (
	StepPending StepState = iota
	StepRunning
	StepDone
	StepFailed
	StepCancelled
)

// Status is a target's progress.
type Status struct {
	Image, Deploy StepState
	SessionID     int // session of the current/last step
	Err           error
}

// Progress is shared between the deployment goroutines and the UI.
type Progress struct {
	mu     sync.Mutex
	status map[string]Status
}

// NewProgress creates an empty progress tracker.
func NewProgress() *Progress { return &Progress{status: map[string]Status{}} }

// Get returns a target's status by project ID.
func (p *Progress) Get(projectID string) Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status[projectID]
}

func (p *Progress) update(projectID string, fn func(*Status)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.status[projectID]
	fn(&s)
	p.status[projectID] = s
}

// Options configures a deployment run.
type Options struct {
	Sessions *shell.Manager
	Cluster  kube.Cluster
	Env      Env // how the charts are configured
	Parallel int
	Width    int // session terminal size
	Height   int
	Progress *Progress
	// SkipImages deploys charts only (images already built).
	SkipImages bool
}

// ErrCancelled is returned when the context is cancelled mid-run.
var ErrCancelled = errors.New("deployment cancelled")

// Run deploys the waves in order; targets inside a wave run in parallel (up
// to opts.Parallel). Each target builds its image, loads it into the
// cluster, then installs its chart.
// The first failure stops the run: running steps are killed (their logs are
// kept in the session inspector) and the rest are marked cancelled.
func Run(ctx context.Context, waves [][]Target, opts Options) error {
	if err := kube.Guard(opts.Cluster); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "go-dark-charts-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	parallel := max(opts.Parallel, 1)

	var firstErr error
	var errOnce sync.Once
	fail := func(err error) {
		errOnce.Do(func() { firstErr = err; cancel() })
	}

	for _, wave := range waves {
		sem := make(chan struct{}, parallel)
		var wg sync.WaitGroup
		for _, t := range wave {
			wg.Add(1)
			go func(t Target) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					markCancelled(opts.Progress, t)
					return
				}
				defer func() { <-sem }()
				if err := deployTarget(ctx, t, tmp, opts); err != nil {
					fail(fmt.Errorf("%s: %w", t.Project.Name, err))
				}
			}(t)
		}
		wg.Wait()
		if ctx.Err() != nil {
			for _, rest := range waves {
				for _, t := range rest {
					markCancelled(opts.Progress, t)
				}
			}
			if firstErr != nil {
				return firstErr
			}
			return ErrCancelled
		}
	}
	return nil
}

func markCancelled(p *Progress, t Target) {
	p.update(t.Project.ID, func(s *Status) {
		if s.Image == StepPending {
			s.Image = StepCancelled
		}
		if s.Deploy == StepPending {
			s.Deploy = StepCancelled
		}
	})
}

func deployTarget(ctx context.Context, t Target, tmp string, opts Options) error {
	id := t.Project.ID
	if !opts.SkipImages {
		opts.Progress.update(id, func(s *Status) { s.Image = StepRunning })
		err := runStep(ctx, opts, t, "image", ImageArgv(t), nil)
		if err == nil {
			err = runStep(ctx, opts, t, "load image", kube.ImportImageArgv(opts.Cluster, t.Ref()), nil)
		}
		opts.Progress.update(id, func(s *Status) { s.Image, s.Err = result(err), err })
		if err != nil {
			return fmt.Errorf("image: %w", err)
		}
	} else {
		opts.Progress.update(id, func(s *Status) { s.Image = StepDone })
	}

	opts.Progress.update(id, func(s *Status) { s.Deploy = StepRunning })
	st, err := PrepareChart(t, tmp, opts.Env)
	if err == nil && st.HasDeps {
		err = runStep(ctx, opts, t, "chart deps", DependencyBuildArgv(st.Dir), nil)
	}
	if err == nil {
		err = runStep(ctx, opts, t, "helm", HelmArgv(opts.Cluster, t, st), nil)
	}
	opts.Progress.update(id, func(s *Status) { s.Deploy, s.Err = result(err), err })
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	return nil
}

func result(err error) StepState {
	switch {
	case err == nil:
		return StepDone
	case errors.Is(err, ErrCancelled):
		return StepCancelled
	}
	return StepFailed
}

// runStep runs one command as a session and waits for it.
func runStep(ctx context.Context, opts Options, t Target, what string, argv, env []string) error {
	id, err := opts.Sessions.Start(shell.StartOptions{
		ProjectID:   t.Project.ID,
		ProjectName: t.Project.Name,
		Dir:         t.Project.Path,
		Argv:        argv,
		Env:         env,
		Title:       fmt.Sprintf("%s · %s", t.Project.Name, what),
		Tag:         TagShip,
		Width:       opts.Width,
		Height:      opts.Height,
	})
	if err != nil {
		return err
	}
	opts.Progress.update(t.Project.ID, func(s *Status) { s.SessionID = id })
	s := opts.Sessions.Get(id)
	select {
	case <-s.Done():
		if code := s.ExitCode(); code != 0 {
			return fmt.Errorf("%s exited with %d (see session #%d)", argv[0], code, id)
		}
		return nil
	case <-ctx.Done():
		_ = opts.Sessions.Kill(id)
		<-s.Done()
		return ErrCancelled
	}
}
