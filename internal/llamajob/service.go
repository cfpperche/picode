// Package llamajob owns durable observations, never the external llama process.
package llamajob

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/llama"
	"github.com/cfpperche/picode/internal/store"
)

type Connection func() (string, string)

type Service struct {
	store               *store.Store
	connection          Connection
	ctx                 context.Context
	stop                context.CancelFunc
	mu                  sync.Mutex
	workers             map[string]bool
	wg                  sync.WaitGroup
	interval, timeLimit time.Duration
	closed              bool
	completed           func(store.LlamaJob)
	// stoppedFn reports an endpoint of PiCode's own llama.cpp service that is
	// not running (SetStopped).
	stoppedFn func(endpoint string) bool
}

func identity(endpoint, key string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(endpoint+"\x00"+key)))
}

func New(st *store.Store, connection Connection, completed ...func(store.LlamaJob)) (*Service, error) {
	ctx, stop := context.WithCancel(context.Background())
	s := &Service{store: st, connection: connection, ctx: ctx, stop: stop, workers: map[string]bool{}, interval: 2 * time.Second, timeLimit: 30 * time.Minute}
	if len(completed) > 0 {
		s.completed = completed[0]
	}
	jobs, err := st.LlamaJobs()
	if err != nil {
		stop()
		return nil, err
	}
	for _, j := range jobs {
		if !j.Active() {
			continue
		}
		if j.State == "queued" {
			_, err = s.update(j.ID, func(j *store.LlamaJob) {
				j.State = "interrupted"
				j.Message = "PiCode restarted before sending the operation."
			})
		} else {
			_, err = s.update(j.ID, func(j *store.LlamaJob) {
				j.State = "unknown"
				j.Message = "Checking the result after PiCode restarted."
			})
			if err == nil {
				s.launch(j.ID, false, nil)
			}
		}
		if err != nil {
			s.Close()
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) Close() { s.mu.Lock(); s.closed = true; s.stop(); s.mu.Unlock(); s.wg.Wait() }

func (s *Service) update(id string, change func(*store.LlamaJob)) (store.LlamaJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, err := s.store.LlamaJob(id)
	if err != nil || !j.Active() {
		return j, err
	}
	change(&j)
	j, err = s.store.UpdateLlamaJob(j)
	if err == nil && j.State == "succeeded" && j.Operation == "download" && s.completed != nil {
		go s.completed(j)
	}
	return j, err
}

func (s *Service) Start(model, operation, key string, replace bool) (store.LlamaJob, error) {
	endpoint, secret := s.connection()
	normalized, err := llama.NormalizeURL(endpoint)
	if err != nil {
		return store.LlamaJob{}, err
	}
	c, err := llama.New(normalized, secret)
	if err != nil {
		return store.LlamaJob{}, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return store.LlamaJob{}, fmt.Errorf("model operations are stopping")
	}
	j, created, err := s.store.BeginLlamaJob(store.LlamaJob{RequestKey: key, Endpoint: normalized, ConnectionID: identity(normalized, secret), Model: model, Operation: operation, ReplaceOthers: replace})
	s.mu.Unlock()
	if err == nil && created {
		s.launch(j.ID, true, c)
	}
	return j, err
}

func (s *Service) Jobs() ([]store.LlamaJob, error) { return s.store.LlamaJobs() }

func (s *Service) Cancel(id string) (store.LlamaJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, err := s.store.LlamaJob(id)
	if err != nil {
		return j, err
	}
	if j.State != "running" || j.Operation != "download" || !j.CancelSupported || j.Observed != "downloading" {
		return j, store.ErrLlamaConflict
	}
	if j.CancelRequested {
		return j, nil
	}
	j.CancelRequested = true
	j.Message = "Cancel requested. Waiting for the server."
	return s.store.UpdateLlamaJob(j)
}

func (s *Service) Reconcile(id string) (store.LlamaJob, error) {
	j, err := s.store.LlamaJob(id)
	if err != nil {
		return j, err
	}
	if j.State == "unknown" {
		if s.stopped(j.Endpoint) {
			return s.interrupt(j.ID)
		}
		s.launch(j.ID, false, nil)
	}
	return j, nil
}

// Abandon ends an unknown job on the owner's word (ADR-0083 amendment,
// 2026-09-23): the job stops holding its model and its endpoint slot, and
// PiCode does not touch the server. Only an unknown job — one PiCode cannot
// resolve — can be abandoned; a worker still following it exits at its next
// read, since the job is no longer active.
func (s *Service) Abandon(id string) (store.LlamaJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, err := s.store.LlamaJob(id)
	if err != nil {
		return j, err
	}
	if j.State != "unknown" {
		return j, store.ErrLlamaConflict
	}
	j.State = "abandoned"
	j.Message = "Abandoned by you. PiCode stopped following it and did not touch the server."
	return s.store.UpdateLlamaJob(j)
}

// SetStopped tells the service how to recognize an endpoint whose server is
// PiCode's own and is not running (llamaservice); nil means none is.
func (s *Service) SetStopped(fn func(endpoint string) bool) {
	s.mu.Lock()
	s.stoppedFn = fn
	s.mu.Unlock()
}

func (s *Service) stopped(endpoint string) bool {
	s.mu.Lock()
	fn := s.stoppedFn
	s.mu.Unlock()
	return fn != nil && fn(endpoint)
}

// InterruptStopped ends every active job on an endpoint whose server is
// PiCode's own and is not running (ADR-0083/0090 amendments): nothing can be in flight on
// a server that is not running, and waiting for it to answer is what kept such
// jobs unknown forever while blocking the start that would answer them.
func (s *Service) InterruptStopped() error {
	jobs, err := s.store.LlamaJobs()
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.Active() && s.stopped(j.Endpoint) {
			if _, err := s.interrupt(j.ID); err != nil && !errors.Is(err, store.ErrLlamaConflict) {
				return err
			}
		}
	}
	return nil
}

func (s *Service) interrupt(id string) (store.LlamaJob, error) {
	return s.update(id, func(j *store.LlamaJob) {
		j.State = "interrupted"
		j.Message = "PiCode's local llama.cpp service is not running, so nothing is in flight. Start it and try again."
	})
}

func (s *Service) launch(id string, dispatch bool, c *llama.Client) {
	s.mu.Lock()
	if s.workers[id] || s.closed {
		s.mu.Unlock()
		return
	}
	s.workers[id] = true
	s.wg.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.wg.Done()
		defer func() { s.mu.Lock(); delete(s.workers, id); s.mu.Unlock() }()
		s.run(id, dispatch, c)
	}()
}

func find(models []llama.Model, id string) llama.Model {
	for _, m := range models {
		if m.ID == id {
			return m
		}
	}
	return llama.Model{ID: id, Status: "missing"}
}
func desired(operation, state string) bool {
	switch operation {
	case "load":
		return state == "loaded" || state == "sleeping"
	case "unload":
		return state == "unloaded" || state == "missing"
	case "download":
		return state == "unloaded" || state == "loaded" || state == "sleeping"
	}
	return false
}
func moving(state string) bool {
	return state == "loading" || state == "downloading" || state == "unloading"
}

func (s *Service) finish(id, state, message string, observed ...string) {
	_, _ = s.update(id, func(j *store.LlamaJob) {
		j.State = state
		j.Message = message
		if len(observed) > 0 {
			j.Observed = observed[0]
		}
		if state == "succeeded" && j.Operation == "download" {
			for i := range j.Progress {
				if j.Progress[i].Total > 0 {
					j.Progress[i].Done = j.Progress[i].Total
				}
			}
		}
	})
}

func (s *Service) run(id string, dispatch bool, c *llama.Client) {
	j, err := s.store.LlamaJob(id)
	if err != nil || !j.Active() {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, s.timeLimit)
	defer cancel()
	if c == nil {
		endpoint, key := s.connection()
		endpoint, err = llama.NormalizeURL(endpoint)
		if err != nil || identity(endpoint, key) != j.ConnectionID {
			s.finish(id, "unknown", "Restore the original server connection, then check the result.")
			return
		}
		c, err = llama.New(endpoint, key)
		if err != nil {
			return
		}
	}
	caps := c.Capabilities(ctx)
	if !dispatch {
		if _, err = s.update(id, func(j *store.LlamaJob) { j.CancelSupported = caps.CancelDownload }); err != nil {
			return
		}
	}
	if dispatch {
		models, e := c.ListContext(ctx)
		if e != nil {
			s.finish(id, "failed", llama.ConnectionFailure(e).Message)
			return
		}
		target := find(models, j.Model)
		if moving(target.Status) {
			s.finish(id, "failed", "This model already has an operation on the server.")
			return
		}
		if j.Operation == "load" && target.Status == "missing" {
			s.finish(id, "failed", "Download this model before loading it.")
			return
		}
		if desired(j.Operation, target.Status) && !j.ReplaceOthers {
			s.finish(id, "succeeded", "The model is already in the requested state.", target.Status)
			return
		}
		if _, err = s.update(id, func(j *store.LlamaJob) {
			j.State = "running"
			j.Stage = "preparing"
			j.CancelSupported = caps.CancelDownload
			j.Message = "Preparing the operation."
		}); err != nil {
			return
		}
		if j.ReplaceOthers {
			// Refuse an externally active operation before unloading any model.
			for _, m := range models {
				if moving(m.Status) {
					s.finish(id, "failed", "Another model is busy on the server. Try again when it finishes.")
					return
				}
			}
			for _, m := range models {
				if m.ID == j.Model || (m.Status != "loaded" && m.Status != "sleeping") {
					continue
				}
				if _, err = s.update(id, func(j *store.LlamaJob) { j.Stage = "unloadingOthers"; j.Message = "Freeing memory from other models." }); err != nil {
					return
				}
				if err = c.Mutate(ctx, "unload", m.ID); err != nil {
					s.finish(id, "unknown", "Could not confirm unloading other models. The requested load did not run.")
					return
				}
				if err = s.waitUnload(ctx, c, m.ID); err != nil {
					s.finish(id, "unknown", "Could not confirm unloading other models. The requested load did not run.")
					return
				}
			}
		}
		if _, err = s.update(id, func(j *store.LlamaJob) { j.Stage = "dispatching"; j.Message = "Sending the operation." }); err != nil {
			return
		}
		err = c.Mutate(ctx, j.Operation, j.Model)
		if err != nil {
			code := llama.ConnectionFailure(err).Code
			if code == "request_rejected" || code == "authentication" || code == "unsupported" {
				s.finish(id, "failed", llama.ConnectionFailure(err).Message)
				return
			}
			s.finish(id, "unknown", "The server did not confirm the request. Checking its actual result.")
		} else {
			if _, err = s.update(id, func(j *store.LlamaJob) { j.Stage = "observing"; j.Message = "Waiting for the server." }); err != nil {
				return
			}
		}
	}
	wake := make(chan struct{}, 1)
	modelID := j.Model
	var progressMu sync.Mutex
	streamProgress := map[string]llama.FileProgress{}
	if caps.Events {
		go func() {
			for {
				_ = c.Observe(ctx, func(event llama.ModelEvent) {
					if event.Model == modelID && len(event.Progress) > 0 {
						progressMu.Lock()
						for _, p := range event.Progress {
							if _, exists := streamProgress[p.File]; exists || len(streamProgress) < 64 {
								streamProgress[p.File] = p
							}
						}
						progressMu.Unlock()
					}
					select {
					case wake <- struct{}{}:
					default:
					}
				})
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
			}
		}()
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		j, err = s.store.LlamaJob(id)
		if err != nil || !j.Active() {
			return
		}
		models, e := c.ListContext(ctx)
		if e == nil {
			m := find(models, j.Model)
			if j.Stage == "unloadingOthers" || j.Stage == "preparing" {
				active := false
				for _, other := range models {
					active = active || moving(other.Status)
				}
				if !active {
					s.finish(id, "interrupted", "Preparation was interrupted. Start a new operation when ready.")
					return
				}
			} else if desired(j.Operation, m.Status) {
				s.finish(id, "succeeded", "The model reached the requested state.", m.Status)
				return
			} else if j.CancelAccepted && (m.Status == "missing" || m.Status == "failed") {
				s.finish(id, "canceled", "The server stopped the download.", m.Status)
				return
			} else if !dispatch && j.Operation == "load" && m.Status == "unloaded" {
				s.finish(id, "interrupted", "Loading was interrupted. The model is not loaded.")
				return
			} else if !dispatch && j.Operation == "unload" && (m.Status == "loaded" || m.Status == "sleeping") {
				// The mirror of the load rule (ADR-0083 amendment): after a
				// restart, a model still loaded means the unload did not happen.
				s.finish(id, "interrupted", "Unloading was interrupted. The model is still loaded.", m.Status)
				return
			} else if m.Status == "failed" {
				s.finish(id, "failed", "The server reported that the model operation failed.", m.Status)
				return
			}
			if j.CancelRequested && !j.CancelAccepted && j.Stage != "cancelDispatch" && m.Status == "downloading" {
				if _, err = s.update(id, func(j *store.LlamaJob) { j.Stage = "cancelDispatch" }); err != nil {
					return
				}
				if err = c.Mutate(ctx, "unload", j.Model); err != nil {
					s.finish(id, "unknown", "The server did not confirm cancellation. Check the result before trying again.")
					return
				}
				if _, err = s.update(id, func(j *store.LlamaJob) { j.CancelAccepted = true; j.Message = "Waiting for the download to stop." }); err != nil {
					return
				}
			}
			if j.State == "unknown" && moving(m.Status) && j.Stage != "cancelDispatch" && j.Stage != "unloadingOthers" {
				if _, err = s.update(id, func(j *store.LlamaJob) {
					j.State = "running"
					j.Stage = "observing"
					j.Message = "Reconnected. Following the operation on the server."
				}); err != nil {
					return
				}
			}
			progressMu.Lock()
			for _, p := range m.Progress {
				if _, exists := streamProgress[p.File]; exists || len(streamProgress) < 64 {
					streamProgress[p.File] = p
				}
			}
			observedProgress := make([]llama.FileProgress, 0, len(streamProgress))
			for _, p := range streamProgress {
				observedProgress = append(observedProgress, p)
			}
			progressMu.Unlock()
			sort.Slice(observedProgress, func(i, j int) bool { return observedProgress[i].File < observedProgress[j].File })
			progress := make([]store.LlamaProgress, 0, len(observedProgress))
			for _, p := range observedProgress {
				progress = append(progress, store.LlamaProgress{File: p.File, Done: p.Done, Total: p.Total})
			}
			// Persist bounded observations only when they change. Download URLs and
			// credentials never enter the job payload or the browser feed.
			if j.Observed != m.Status || (len(progress) > 0 && fmt.Sprint(j.Progress) != fmt.Sprint(progress)) {
				if _, err = s.update(id, func(j *store.LlamaJob) {
					j.Observed = m.Status
					if len(progress) > 0 {
						j.Progress = progress
					}
					if !j.CancelRequested {
						j.Message = "Waiting for the server: " + m.Status + "."
					}
				}); err != nil {
					return
				}
			}
		} else if s.stopped(j.Endpoint) {
			// PiCode's own server there is not running: nothing is in flight.
			_, _ = s.interrupt(id)
			return
		} else if j.State != "unknown" {
			s.finish(id, "unknown", "Connection lost. Checking the result without repeating the operation.")
		}
		select {
		case <-ctx.Done():
			s.finish(id, "unknown", "Result unknown. Check the server for the latest status.")
			return
		case <-ticker.C:
		case <-wake:
			// Cap SSE-driven catalog refreshes and durable writes at two per second.
			select {
			case <-ctx.Done():
				s.finish(id, "unknown", "Result unknown. Check the server for the latest status.")
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
}

func (s *Service) waitUnload(ctx context.Context, c *llama.Client, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		models, err := c.ListContext(ctx)
		if err != nil {
			return err
		}
		m := find(models, id)
		if desired("unload", m.Status) {
			return nil
		}
		if m.Status == "failed" {
			return fmt.Errorf("unload failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.interval):
		}
	}
}
