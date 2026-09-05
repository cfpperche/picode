package docker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

const maxProjectContainers = 128

type HealthContainer struct {
	Container  Container `json:"container"`
	Stats      *Stats    `json:"stats,omitempty"`
	Error      string    `json:"error,omitempty"`
	StatsError string    `json:"statsError,omitempty"`
}

type HealthSnapshot struct {
	Endpoint   string            `json:"endpoint"`
	Project    string            `json:"project"`
	SampledAt  string            `json:"sampledAt"`
	Error      string            `json:"error,omitempty"`
	Containers []HealthContainer `json:"containers"`
}

type HealthView struct {
	Monitor   store.DockerMonitor    `json:"monitor"`
	Snapshot  *HealthSnapshot        `json:"snapshot,omitempty"`
	Stale     bool                   `json:"stale"`
	Incidents []store.DockerIncident `json:"incidents"`
}

func (s *Service) Health(endpoint, project string) (HealthView, error) {
	m, err := s.Store.DockerMonitor(endpoint, project)
	if errors.Is(err, sql.ErrNoRows) {
		m = store.DefaultDockerMonitor(endpoint, project)
	} else if err != nil {
		return HealthView{}, err
	}
	v := HealthView{Monitor: m, Stale: true}
	if len(m.Snapshot) > 0 {
		var snap HealthSnapshot
		if err = json.Unmarshal(m.Snapshot, &snap); err != nil {
			return v, err
		}
		v.Snapshot = &snap
		at, _ := time.Parse(time.RFC3339Nano, snap.SampledAt)
		v.Stale = snap.Error != "" || time.Since(at) > time.Duration(2*m.IntervalSeconds)*time.Second
	}
	v.Monitor.Snapshot = nil // one copy of the bounded snapshot in the response
	v.Incidents, err = s.Store.DockerIncidents(endpoint, project)
	return v, err
}

func (s *Service) ConfigureMonitor(ctx context.Context, m store.DockerMonitor) (store.DockerMonitor, error) {
	if err := validateHealthTarget(m.Endpoint, m.Project); err != nil {
		return m, err
	}
	if err := m.Validate(); err != nil {
		return m, err
	}
	// Disabling a pinned monitor must work even while its engine is offline.
	if m.Enabled {
		check, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		c, err := s.client(check)
		if err != nil {
			return m, err
		}
		defer c.Close()
		if c.Endpoint != m.Endpoint {
			return m, errors.New("Docker connection changed. Refresh before enabling monitoring")
		}
		members, e := projectContainers(check, c, m.Project)
		if e != nil {
			return m, e
		}
		if len(members) == 0 {
			err = errors.New("No containers remain in this project. Refresh projects")
		}
		if err != nil {
			return m, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return m, errors.New("PiCode is shutting down")
	}
	saved, err := s.Store.SaveDockerMonitor(m)
	if err != nil {
		return m, err
	}
	if active, ok := s.collecting[monitorKey(m.Endpoint, m.Project)]; ok {
		active.cancel()
	}
	select {
	case s.monitorWake <- struct{}{}:
	default:
	}
	return saved, nil
}

func (s *Service) CheckHealth(ctx context.Context, endpoint, project string) (HealthView, error) {
	if err := validateHealthTarget(endpoint, project); err != nil {
		return HealthView{}, err
	}
	m, err := s.Store.DockerMonitor(endpoint, project)
	if errors.Is(err, sql.ErrNoRows) {
		m, err = s.Store.SaveDockerMonitor(store.DefaultDockerMonitor(endpoint, project))
	}
	if err != nil {
		return HealthView{}, err
	}
	snap := s.sampleHealth(ctx, endpoint, project)
	if err = ctx.Err(); err != nil {
		return HealthView{}, err
	}
	if err = s.recordHealth(m, snap); err != nil {
		return HealthView{}, err
	}
	return s.Health(endpoint, project)
}

func validateHealthTarget(endpoint, project string) error {
	normalized, err := EndpointFrom(func(key string) string {
		if key == "PICODE_DOCKER_HOST" {
			return endpoint
		}
		return ""
	}, nil)
	if err != nil || normalized != endpoint || len(project) > 256 {
		return errors.New("Choose a project on a valid local Docker connection")
	}
	return nil
}

func projectContainers(ctx context.Context, c *Client, project string) ([]Container, error) {
	if len(project) > 256 {
		return nil, errors.New("Project name is too long")
	}
	rows, err := c.Containers(ctx)
	if err != nil {
		return nil, err
	}
	out := []Container{}
	for _, ct := range rows {
		if ct.Project == project {
			out = append(out, ct)
		}
	}
	if len(out) > maxProjectContainers {
		return nil, fmt.Errorf("This project exceeds the limit of %d containers per operation", maxProjectContainers)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func inspectProject(ctx context.Context, c *Client, project string, stats bool) ([]HealthContainer, error) {
	rows, err := projectContainers(ctx, c, project)
	if err != nil {
		return nil, err
	}
	out := make([]HealthContainer, len(rows))
	var wg sync.WaitGroup
	work := make(chan int)
	for range min(4, len(rows)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				h := HealthContainer{Container: rows[i]}
				ct, e := c.Inspect(ctx, rows[i].ID)
				if e != nil {
					h.Error = "Container state could not be read."
				} else {
					h.Container = ct
					if ct.Project != project {
						h.Error = "Project membership changed. Check again."
					}
					if stats && ct.State == "running" {
						v, e := c.Stats(ctx, ct.ID)
						if e != nil {
							h.StatsError = "Resource usage could not be read."
						} else {
							h.Stats = &v
						}
					}
				}
				out[i] = h
			}
		}()
	}
	for i := range rows {
		work <- i
	}
	close(work)
	wg.Wait()
	return out, nil
}

func (s *Service) sampleHealth(parent context.Context, endpoint, project string) HealthSnapshot {
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	snap := HealthSnapshot{Endpoint: endpoint, Project: project, Containers: []HealthContainer{}}
	c, err := s.client(ctx)
	if err == nil {
		defer c.Close()
		if c.Endpoint != endpoint {
			err = errors.New("connection changed")
		} else {
			snap.Containers, err = inspectProject(ctx, c, project, true)
		}
	}
	if err != nil {
		snap.Error = "Docker could not be sampled on this project's saved connection."
	}
	snap.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
	return snap
}

func healthSignals(m store.DockerMonitor, snap HealthSnapshot) []store.DockerSignal {
	signals := []store.DockerSignal{}
	add := func(key, title, state, detail string) {
		signals = append(signals, store.DockerSignal{Key: key, Title: title, State: state, Detail: detail})
	}
	state := "good"
	if snap.Error != "" {
		state = "bad"
	}
	add("connection", "Docker connection", state, snap.Error)
	if snap.Error != "" {
		return signals
	}
	state = "good"
	if len(snap.Containers) == 0 {
		state = "bad"
	}
	add("members", "Project containers", state, fmt.Sprintf("%d containers observed.", len(snap.Containers)))
	var previous HealthSnapshot
	_ = json.Unmarshal(m.Snapshot, &previous)
	before := map[string]Container{}
	for _, h := range previous.Containers {
		if h.Error == "" {
			before[h.Container.ID] = h.Container
		}
	}
	for _, h := range snap.Containers {
		c := h.Container
		if h.Error != "" {
			continue
		}
		binary := func(bad bool) string {
			if bad {
				return "bad"
			}
			return "good"
		}
		key := c.ID + ":"
		healthState := "unknown"
		if c.State == "running" && c.HasHealthCheck {
			if c.Health == "healthy" {
				healthState = "good"
			}
			if c.Health == "unhealthy" {
				healthState = "bad"
			}
		}
		add(key+"health", c.Name+": health check", healthState, "Health check: "+c.Health)
		old, known := before[c.ID]
		restarting := c.State == "restarting" || (known && c.RestartCount > old.RestartCount)
		add(key+"restarts", c.Name+": repeated restarts", binary(restarting), fmt.Sprintf("State %s; %d recorded restarts.", c.State, c.RestartCount))
		add(key+"exit", c.Name+": failed exit", binary(c.State == "exited" && c.ExitCode != 0), fmt.Sprintf("State %s; exit code %d.", c.State, c.ExitCode))
		add(key+"oom", c.Name+": memory limit reached", binary(c.OOMKilled), fmt.Sprintf("Docker reported an out-of-memory termination: %t.", c.OOMKilled))
		if h.Stats != nil && c.State == "running" {
			add(key+"cpu", c.Name+": high CPU", binary(h.Stats.CPUPercent >= float64(m.CPUPercent)), fmt.Sprintf("CPU %.1f%%; threshold %d%% of one core.", h.Stats.CPUPercent, m.CPUPercent))
			if h.Stats.LimitBytes > 0 {
				used := float64(h.Stats.MemoryBytes) / float64(h.Stats.LimitBytes) * 100
				add(key+"memory", c.Name+": high memory", binary(used >= float64(m.MemoryPercent)), fmt.Sprintf("Memory %.1f%% of reported limit; threshold %d%%.", used, m.MemoryPercent))
			}
		}
	}
	return signals
}

func (s *Service) recordHealth(m store.DockerMonitor, snap HealthSnapshot) error {
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	at, err := time.Parse(time.RFC3339Nano, snap.SampledAt)
	if err != nil {
		return err
	}
	return s.Store.RecordDockerHealth(m.Endpoint, m.Project, m.Revision, raw, healthSignals(m, snap), at)
}

type collection struct {
	revision int
	cancel   context.CancelFunc
}

func monitorKey(endpoint, project string) string { return endpoint + "\n" + project }

func (s *Service) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		s.collectDue()
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
		case <-s.monitorWake:
		}
	}
}

func (s *Service) collectDue() {
	monitors, err := s.Store.DockerMonitors()
	if err != nil {
		log.Printf("docker: read monitors: %v", err)
		return
	}
	for _, m := range monitors {
		if !m.Enabled {
			continue
		}
		at, _ := time.Parse(time.RFC3339Nano, m.SampledAt)
		if m.EvaluatedAt != "" && time.Since(at) < time.Duration(m.IntervalSeconds)*time.Second {
			continue
		}
		key := monitorKey(m.Endpoint, m.Project)
		s.mu.Lock()
		if s.closed || len(s.collecting) >= 4 {
			s.mu.Unlock()
			return
		}
		if _, ok := s.collecting[key]; ok {
			s.mu.Unlock()
			continue
		}
		ctx, cancel := context.WithCancel(s.ctx)
		s.collecting[key] = collection{revision: m.Revision, cancel: cancel}
		s.wg.Add(1)
		s.mu.Unlock()
		go func(m store.DockerMonitor) {
			defer s.wg.Done()
			defer cancel()
			snap := s.sampleHealth(ctx, m.Endpoint, m.Project)
			if ctx.Err() == nil {
				if err := s.recordHealth(m, snap); err != nil && !errors.Is(err, store.ErrDockerConflict) {
					log.Printf("docker: record health: %v", err)
				}
			}
			s.mu.Lock()
			delete(s.collecting, key)
			s.mu.Unlock()
		}(m)
	}
}
