// Package clijob owns durable CLI lifecycle operations (ADR-0087). The
// vendor command runs once; a PiCode restart never replays an install.
package clijob

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// ErrTerminalsRunning wraps guard refusals: live terminals of that CLI must
// be confirmed before a lifecycle job may run.
var ErrTerminalsRunning = errors.New("live terminals must be confirmed")

// Exec describes one resolved command the service may run.
type Exec struct {
	Exe  string
	Args []string
	Env  []string
	// Dir is the working directory the vendor command runs in. A plugin
	// action with a project scope must run inside the workspace folder
	// (ADR-0167), and the empty value keeps the daemon's own directory.
	Dir string
}

// Resolve returns the command for one CLI + action, or an error when the
// pair offers no managed lifecycle. LiveTerminals counts that CLI's running
// terminals; AfterSuccess refreshes cached facts after a succeeded job.
type Deps struct {
	Store         *store.Store
	Resolve       func(cli, action, payload string) (Exec, error)
	LiveTerminals func(cli string) int
	AfterSuccess  func(cli, action string)
	RunTimeout    time.Duration
}

type Service struct {
	deps    Deps
	ctx     context.Context
	stop    context.CancelFunc
	mu      sync.Mutex
	workers map[string]bool
	wg      sync.WaitGroup
	closed  bool
}

func New(deps Deps) (*Service, error) {
	if deps.RunTimeout == 0 {
		deps.RunTimeout = 10 * time.Minute
	}
	ctx, stop := context.WithCancel(context.Background())
	s := &Service{deps: deps, ctx: ctx, stop: stop, workers: map[string]bool{}}
	jobs, err := deps.Store.CLIJobs()
	if err != nil {
		stop()
		return nil, err
	}
	for _, j := range jobs {
		if !j.Active() {
			continue
		}
		if _, err := s.update(j.ID, func(j *store.CLIJob) {
			j.State = "interrupted"
			j.Message = "PiCode restarted while the operation was running. Nothing was retried; check the CLI's state."
		}); err != nil {
			s.Close()
			return nil, err
		}
	}
	return s, nil
}

// Close cancels a running vendor command and waits. A canceled operation is
// interrupted, never failed: the CLI's state is unknown, not broken.
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.stop()
	s.mu.Unlock()
	s.wg.Wait()
}

// List returns active jobs plus the most recent records for UI recovery.
func (s *Service) List() ([]store.CLIJob, error) { return s.deps.Store.CLIJobs() }

func (s *Service) update(id string, change func(*store.CLIJob)) (store.CLIJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, err := s.deps.Store.CLIJob(id)
	if err != nil || !j.Active() {
		return j, err
	}
	change(&j)
	return s.deps.Store.UpdateCLIJob(j)
}

// Start validates the guards, reserves the durable job and runs the vendor
// command. Repeating an identical request key returns the existing job.
func (s *Service) Start(cli, action, requestKey, payload string, confirmTerminals bool) (store.CLIJob, error) {
	if s.deps.Resolve == nil {
		return store.CLIJob{}, errors.New("CLI lifecycle is unavailable.")
	}
	exe, err := s.deps.Resolve(cli, action, payload)
	if err != nil {
		return store.CLIJob{}, err
	}
	if s.deps.LiveTerminals != nil {
		if live := s.deps.LiveTerminals(cli); live > 0 && !confirmTerminals {
			return store.CLIJob{}, fmt.Errorf("%w: %d %s terminal(s) are running. Restart or stop them first, or confirm to continue — they keep the old version until restarted.", ErrTerminalsRunning, live, cli)
		}
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return store.CLIJob{}, errors.New("CLI lifecycle is stopping")
	}
	j, created, err := s.deps.Store.BeginCLIJob(store.CLIJob{CLI: cli, Action: action, RequestKey: requestKey, Payload: payload})
	if err != nil || !created {
		s.mu.Unlock()
		return j, err
	}
	s.workers[j.ID] = true
	s.wg.Add(1)
	s.mu.Unlock()
	go s.run(j.ID, exe)
	return j, nil
}

func (s *Service) run(id string, exe Exec) {
	defer func() {
		s.mu.Lock()
		delete(s.workers, id)
		s.mu.Unlock()
		s.wg.Done()
	}()
	if _, err := s.update(id, func(j *store.CLIJob) {
		j.State = "running"
		j.Message = "Running " + j.Action + "."
	}); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, s.deps.RunTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe.Exe, exe.Args...)
	cmd.Env = exe.Env
	cmd.Dir = exe.Dir
	var out boundedOutput
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.WaitDelay = 5 * time.Second
	runErr := cmd.Run()
	state, msg := "succeeded", "Done."
	switch {
	case s.ctx.Err() != nil:
		state = "interrupted"
		msg = "PiCode shut down during the operation. Nothing was retried; check the CLI's state."
	case ctx.Err() != nil:
		state = "failed"
		msg = "The operation timed out."
	case runErr != nil:
		state = "failed"
		msg = firstLine(runErr.Error())
		if tail := strings.TrimSpace(out.String()); tail != "" {
			msg = firstLine(tail)
		}
	}
	if _, err := s.update(id, func(j *store.CLIJob) {
		j.State = state
		j.Message = msg
		j.Output = out.String()
	}); err != nil {
		return
	}
	if state == "succeeded" && s.deps.AfterSuccess != nil {
		s.afterSuccess(id)
	}
}

func (s *Service) afterSuccess(id string) {
	j, err := s.deps.Store.CLIJob(id)
	if err != nil {
		return
	}
	s.deps.AfterSuccess(j.CLI, j.Action)
}

type boundedOutput struct{ b []byte }

func (o *boundedOutput) Write(p []byte) (int, error) {
	o.b = append(o.b, p...)
	if len(o.b) > 8192 {
		o.b = append([]byte{}, o.b[len(o.b)-8192:]...)
	}
	return len(p), nil
}

func (o *boundedOutput) String() string { return strings.TrimSpace(string(o.b)) }

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
