package docker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

type Resolver func(context.Context) (*Client, error)

type Service struct {
	Store         *store.Store
	Resolve       Resolver
	changed       func()
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	wg            sync.WaitGroup
	closed        bool
	watchEndpoint string
	watchCancel   context.CancelFunc
}

func NewService(ctx context.Context, st *store.Store, resolve Resolver, changed func()) (*Service, error) {
	if err := st.RecoverDockerOperations(); err != nil {
		return nil, err
	}
	if resolve == nil {
		resolve = LocalClient
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Service{Store: st, Resolve: resolve, changed: changed, ctx: ctx, cancel: cancel}, nil
}

func (s *Service) Close() { s.mu.Lock(); s.closed = true; s.cancel(); s.mu.Unlock(); s.wg.Wait() }

func (s *Service) client(ctx context.Context) (*Client, error) {
	c, err := s.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.Check(ctx); err != nil {
		c.Close()
		return nil, err
	}
	s.watch(c.Endpoint)
	return c, nil
}

func (s *Service) watch(endpoint string) {
	if s.changed == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.watchEndpoint == endpoint {
		return
	}
	if s.watchCancel != nil {
		s.watchCancel()
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.watchCancel = cancel
	s.watchEndpoint = endpoint
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		c, err := NewClient(endpoint)
		if err != nil {
			return
		}
		defer c.Close()
		for ctx.Err() == nil {
			_ = c.Events(ctx, s.changed)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				s.changed()
			}
		}
	}()
}

type Inventory struct {
	Endpoint   string      `json:"endpoint"`
	Containers []Container `json:"containers"`
	SampledAt  string      `json:"sampledAt"`
}

func (s *Service) Inventory(ctx context.Context) (Inventory, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	c, err := s.client(ctx)
	if err != nil {
		return Inventory{}, err
	}
	defer c.Close()
	rows, err := c.Containers(ctx)
	return Inventory{Endpoint: c.Endpoint, Containers: rows, SampledAt: time.Now().UTC().Format(time.RFC3339)}, err
}

type Detail struct {
	Container  Container `json:"container"`
	Stats      *Stats    `json:"stats,omitempty"`
	StatsError string    `json:"statsError,omitempty"`
	Logs       *Logs     `json:"logs,omitempty"`
	LogsError  string    `json:"logsError,omitempty"`
	Endpoint   string    `json:"endpoint"`
}

func (s *Service) Detail(ctx context.Context, id string) (Detail, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	c, err := s.client(ctx)
	if err != nil {
		return Detail{}, err
	}
	defer c.Close()
	container, err := c.Inspect(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	d := Detail{Container: container, Endpoint: c.Endpoint}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		l, err := c.Logs(ctx, container)
		if err != nil {
			d.LogsError = err.Error()
		} else {
			d.Logs = &l
		}
	}()
	if container.State == "running" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats, err := c.Stats(ctx, id)
			if err != nil {
				d.StatsError = err.Error()
			} else {
				d.Stats = &stats
			}
		}()
	}
	wg.Wait()
	return d, nil
}

type Request struct {
	Action      string `json:"action"`
	ContainerID string `json:"containerId"`
	RequestKey  string `json:"requestKey"`
	AgentID     string `json:"agentId,omitempty"`
	Actor       string `json:"-"`
}

var requestKey = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

func ValidAction(action, state string) bool {
	switch action {
	case "start":
		return state == "created" || state == "exited"
	case "stop", "restart":
		return state == "running"
	}
	return false
}

func (s *Service) Start(ctx context.Context, req Request) (store.DockerOperation, error) {
	if !fullID.MatchString(req.ContainerID) || !requestKey.MatchString(req.RequestKey) || (req.Action != "start" && req.Action != "stop" && req.Action != "restart") {
		return store.DockerOperation{}, errors.New("Provide start, stop or restart, a full container ID, and a request key (8–128 letters, numbers, underscores or hyphens)")
	}
	if req.AgentID != "" {
		a, err := s.Store.GetAgent(req.AgentID)
		if err != nil {
			return store.DockerOperation{}, errors.New("Unknown sysadmin agent")
		}
		req.Actor = "Agent: " + a.Name
	}
	if old, err := s.Store.DockerOperationByKey(req.RequestKey); err == nil {
		if old.ContainerID != req.ContainerID || old.Action != req.Action || old.AgentID != req.AgentID {
			return old, store.ErrDockerConflict
		}
		return old, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return store.DockerOperation{}, err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	c, err := s.client(checkCtx)
	if err != nil {
		return store.DockerOperation{}, err
	}
	before, err := c.Inspect(checkCtx, req.ContainerID)
	if err != nil {
		c.Close()
		return store.DockerOperation{}, err
	}
	if !ValidAction(req.Action, before.State) {
		c.Close()
		return store.DockerOperation{}, fmt.Errorf("Cannot %s a container that is %s. Refresh its state", req.Action, before.State)
	}
	if err = ctx.Err(); err != nil {
		c.Close()
		return store.DockerOperation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		c.Close()
		return store.DockerOperation{}, errors.New("PiCode is shutting down")
	}
	op, created, err := s.Store.BeginDockerOperation(store.DockerOperation{RequestKey: req.RequestKey, Endpoint: c.Endpoint, ContainerID: before.ID, ContainerName: before.Name, Action: req.Action, Actor: req.Actor, AgentID: req.AgentID})
	if err != nil || !created {
		c.Close()
		return op, err
	}
	s.wg.Add(1)
	go func() { defer s.wg.Done(); defer c.Close(); s.run(c, before, op) }()
	return op, nil
}

func (s *Service) run(c *Client, before Container, op store.DockerOperation) {
	ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
	defer cancel()
	state, message := "succeeded", "Container state verified."
	err := c.Mutate(ctx, op.ContainerID, op.Action)
	if err != nil {
		state = "unknown"
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			state = "failed"
		}
		message = err.Error()
	} else {
		after, verifyErr := c.Inspect(ctx, op.ContainerID)
		ok := verifyErr == nil && ((op.Action == "stop" && after.State == "exited") || (op.Action != "stop" && after.State == "running"))
		if op.Action == "restart" && after.StartedAt == before.StartedAt {
			ok = false
		}
		if !ok {
			state = "unknown"
			message = "Docker accepted the action, but its resulting state could not be verified. Refresh before retrying."
			if verifyErr != nil {
				message += " " + verifyErr.Error()
			}
		}
	}
	if err := s.Store.FinishDockerOperation(op.ID, state, message); err != nil {
		log.Printf("docker: record result for %s: %v", op.ID, err)
	}
}
