package docker

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

type PlanRequest struct {
	Kind         string `json:"kind"`
	Project      string `json:"project,omitempty"`
	Action       string `json:"action,omitempty"`
	ResourceKind string `json:"resourceKind,omitempty"`
	ResourceID   string `json:"resourceId,omitempty"`
	ContainerID  string `json:"containerId,omitempty"`
	Procedure    string `json:"procedure,omitempty"`
	AgentID      string `json:"agentId,omitempty"`
	Actor        string `json:"-"`
}

type ExecuteRequest struct {
	PlanID     string `json:"planId"`
	RequestKey string `json:"requestKey"`
	Approved   bool   `json:"approved"`
	AgentID    string `json:"agentId,omitempty"`
	Actor      string `json:"-"`
}

func digest(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func containerFingerprint(c Container) string {
	// Human-readable Status and sampled metrics change without a lifecycle
	// transition. Credentials and host configuration never enter this digest.
	return digest([]any{c.ID, c.Name, c.Project, c.Service, c.ImageID, c.State, c.StartedAt, c.RestartCount, c.HasHealthCheck, c.Health})
}

func stepFingerprint(c Container, condition string) string {
	if condition == "stop-restart-loop" {
		// A loop advances its timestamps and counter while the owner reviews
		// it. Keep exact identity and the restarting precondition, rather
		// than making this procedure impossible to confirm during a loop.
		return digest([]any{c.ID, c.Name, c.Project, c.Service, c.ImageID, c.State, condition})
	}
	return containerFingerprint(c)
}

func (s *Service) requester(actor, agentID string) (string, error) {
	if agentID == "" {
		return actor, nil
	}
	a, err := s.Store.GetAgent(agentID)
	if err != nil {
		return "", errors.New("Unknown sysadmin agent")
	}
	return "Agent: " + a.Name + " via " + actor, nil
}

func (s *Service) Preview(ctx context.Context, req PlanRequest) (store.DockerPlan, error) {
	actor, err := s.requester(req.Actor, req.AgentID)
	if err != nil {
		return store.DockerPlan{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	c, err := s.client(ctx)
	if err != nil {
		return store.DockerPlan{}, err
	}
	defer c.Close()
	p, err := s.buildPlan(ctx, c, req)
	if err != nil {
		return p, err
	}
	p.Actor, p.AgentID = actor, req.AgentID
	return s.Store.CreateDockerPlan(p)
}

func (s *Service) buildPlan(ctx context.Context, c *Client, req PlanRequest) (store.DockerPlan, error) {
	p := store.DockerPlan{Kind: req.Kind, Project: req.Project, Endpoint: c.Endpoint, Steps: []store.DockerStep{}}
	if len(req.Project) > 256 {
		return p, errors.New("Project name is too long")
	}
	input := req
	input.Actor, input.AgentID = "", ""
	p.Input, _ = json.Marshal(input)
	switch req.Kind {
	case "project":
		if req.Action != "start" && req.Action != "stop" && req.Action != "restart" {
			return p, errors.New("Choose start, stop or restart")
		}
		members, err := inspectProject(ctx, c, req.Project, false)
		if err != nil {
			return p, err
		}
		if len(members) == 0 {
			return p, errors.New("No containers remain in this project. Refresh the inventory")
		}
		fingerprints := []string{}
		eligible := 0
		for _, h := range members {
			if h.Error != "" {
				return p, errors.New("Project membership or state changed during review. Create a fresh preview")
			}
			ct := h.Container
			step := store.DockerStep{Kind: "container", Target: ct.ID, Name: ct.Name, Action: req.Action, Fingerprint: containerFingerprint(ct), State: "queued", Message: "Current state: " + ct.State}
			if !ValidAction(req.Action, ct.State) {
				step.State, step.Message = "skipped", "Unchanged: "+req.Action+" does not apply to state "+ct.State
			} else {
				eligible++
			}
			p.Steps = append(p.Steps, step)
			fingerprints = append(fingerprints, step.Fingerprint)
		}
		if eligible == 0 {
			return p, errors.New("All containers are unchanged for this action. Choose another action or refresh")
		}
		p.Title = strings.ToUpper(req.Action[:1]) + req.Action[1:] + " project · " + projectName(req.Project)
		p.Impact = fmt.Sprintf("%s will change; %d will remain unchanged. Connections may be interrupted. Existing containers are processed in ID order; Compose dependency order is not applied.", CountLabel(eligible, "container"), len(members)-eligible)
		p.Fingerprint = digest(fingerprints)
	case "resource":
		if _, err := resourcePath(req.ResourceKind, req.ResourceID); err != nil {
			return p, err
		}
		inv, err := c.Resources(ctx)
		if err != nil {
			return p, err
		}
		r, err := findResource(inv, req.ResourceKind, req.ResourceID)
		if err != nil {
			return p, err
		}
		if !r.Removable {
			return p, fmt.Errorf("Resource cannot be removed: %s", r.Reason)
		}
		p.Title = "Remove " + r.Kind + " · " + r.Name
		p.Impact = "Remove only the selected resource. No containers currently reference it. Future deployments may need to download the image or recreate the network. Reported size is not a promise of reclaimed space."
		p.Fingerprint = digest(r)
		p.Steps = []store.DockerStep{{Kind: r.Kind, Target: r.ID, Name: r.Name, Action: "remove", Fingerprint: p.Fingerprint, State: "queued"}}
	case "procedure":
		ct, err := c.Inspect(ctx, req.ContainerID)
		if err != nil {
			return p, err
		}
		if ct.Project != req.Project {
			return p, errors.New("Container project changed. Inspect again")
		}
		action := ""
		memoryThreshold := 0
		switch req.Procedure {
		case "stop-restart-loop":
			if ct.State != "restarting" {
				return p, errors.New("No restart-loop evidence remains for this container")
			}
			action = "stop"
			p.Title = "Stop repeated restarts · " + ct.Name
			p.Impact = "Stop this container to interrupt repeated restarts. Its service will be unavailable until started again. This does not correct the underlying cause."
		case "restart-unhealthy":
			if ct.State != "running" || ct.Health != "unhealthy" {
				return p, errors.New("This container no longer has a failed health check")
			}
			action = "restart"
			p.Title = "Restart unhealthy service · " + ct.Name
			p.Impact = "Restart this service and wait up to 30 seconds for its existing health check. Connections will be interrupted. Failure stops the procedure."
		case "start-stopped-service":
			if !ValidAction("start", ct.State) {
				return p, errors.New("This container is no longer stopped")
			}
			action = "start"
			p.Title = "Start stopped service · " + ct.Name
			p.Impact = "Start this container and verify its state. It may be an unavailable dependency; no dependency relationship or recovery of other services has been established."
		case "restart-high-memory":
			if ct.State != "running" {
				return p, errors.New("Resource usage is unavailable for a stopped container")
			}
			stats, err := c.Stats(ctx, ct.ID)
			if err != nil {
				return p, err
			}
			m, err := s.Store.DockerMonitor(c.Endpoint, ct.Project)
			if errors.Is(err, sql.ErrNoRows) {
				m = store.DefaultDockerMonitor(c.Endpoint, ct.Project)
			} else if err != nil {
				return p, err
			}
			if stats.LimitBytes == 0 || float64(stats.MemoryBytes)/float64(stats.LimitBytes)*100 < float64(m.MemoryPercent) {
				return p, errors.New("Memory is below the project's threshold. Check health again")
			}
			action = "restart"
			p.Title = "Restart to reduce memory · " + ct.Name
			memoryThreshold = m.MemoryPercent
			p.Impact = fmt.Sprintf("Restart this container, then check memory against %d%% of the reported limit. Connections will be interrupted. This is a temporary mitigation, not a root-cause fix.", m.MemoryPercent)
		default:
			return p, errors.New("Choose a supported maintenance procedure")
		}
		p.Fingerprint = stepFingerprint(ct, req.Procedure)
		p.Steps = []store.DockerStep{
			{Kind: "container", Target: ct.ID, Name: ct.Name, Action: action, Condition: req.Procedure, Fingerprint: p.Fingerprint, State: "queued", Message: "Current state: " + ct.State},
			{Kind: "verify", Target: ct.ID, Name: ct.Name, Action: req.Procedure, MemoryPercent: memoryThreshold, State: "queued", Message: "Check the observed result; stop if it cannot be verified."},
		}
	default:
		return p, errors.New("Choose a project action, resource removal or supervised procedure")
	}
	if err := ctx.Err(); err != nil {
		return p, err
	}
	return p, nil
}

func projectName(project string) string {
	if project == "" {
		return "Standalone containers"
	}
	return project
}

func CountLabel(n int, label string) string {
	if n != 1 {
		label += "s"
	}
	return fmt.Sprintf("%d %s", n, label)
}

func (s *Service) Execute(ctx context.Context, req ExecuteRequest) (store.DockerJob, error) {
	if !req.Approved {
		return store.DockerJob{}, errors.New("Review and confirm this plan before execution")
	}
	if !requestKey.MatchString(req.RequestKey) {
		return store.DockerJob{}, errors.New("Provide a request key of 8–128 letters, numbers, underscores or hyphens")
	}
	actor, err := s.requester(req.Actor, req.AgentID)
	if err != nil {
		return store.DockerJob{}, err
	}
	if old, e := s.Store.DockerJobByKey(req.RequestKey); e == nil {
		if old.PlanID != req.PlanID || old.ApproverAgentID != req.AgentID {
			return old, store.ErrDockerConflict
		}
		return old, nil
	} else if !errors.Is(e, sql.ErrNoRows) {
		return store.DockerJob{}, e
	}
	p, err := s.Store.DockerPlan(req.PlanID)
	if err != nil {
		return store.DockerJob{}, errors.New("Maintenance plan not found")
	}
	expires, err := time.Parse(time.RFC3339Nano, p.ExpiresAt)
	if err != nil || !time.Now().Before(expires) {
		return store.DockerJob{}, errors.New("This preview expired. Create a fresh preview before confirming")
	}
	check, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	c, err := s.client(check)
	if err != nil {
		return store.DockerJob{}, err
	}
	accepted := false
	defer func() {
		if !accepted {
			c.Close()
		}
	}()
	if c.Endpoint != p.Endpoint {
		return store.DockerJob{}, errors.New("Docker connection changed. Create a fresh preview")
	}
	var input PlanRequest
	if err = json.Unmarshal(p.Input, &input); err != nil {
		return store.DockerJob{}, err
	}
	fresh, err := s.buildPlan(check, c, input)
	if err != nil {
		return store.DockerJob{}, err
	}
	if fresh.Fingerprint != p.Fingerprint || fresh.Impact != p.Impact {
		return store.DockerJob{}, errors.New("Targets or state changed since review. Create a fresh preview")
	}
	if !time.Now().Before(expires) {
		return store.DockerJob{}, errors.New("This preview expired during validation. Create a fresh preview")
	}
	if err = ctx.Err(); err != nil {
		return store.DockerJob{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return store.DockerJob{}, errors.New("PiCode is shutting down")
	}
	j, created, err := s.Store.BeginDockerJob(store.DockerJob{RequestKey: req.RequestKey, PlanID: p.ID, Kind: p.Kind, Title: p.Title, Endpoint: p.Endpoint, Project: p.Project, Actor: p.Actor, AgentID: p.AgentID, ApprovedBy: actor, ApproverAgentID: req.AgentID, Steps: p.Steps})
	if err != nil || !created {
		return j, err
	}
	accepted = true
	s.wg.Add(1)
	go func() { defer s.wg.Done(); defer c.Close(); s.runJob(c, j) }()
	return j, nil
}

func (s *Service) runJob(c *Client, j store.DockerJob) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()
	failed, unknown, succeeded := false, false, 0
	for i := range j.Steps {
		step := &j.Steps[i]
		if step.State == "skipped" {
			continue
		}
		if ctx.Err() != nil || (j.Kind == "procedure" && (failed || unknown)) {
			step.State, step.Message = "skipped", "An earlier step failed or execution was interrupted. Not executed."
			if ctx.Err() != nil {
				unknown = true
			}
			continue
		}
		step.State = "running"
		if err := s.Store.UpdateDockerJob(j); err != nil {
			log.Printf("docker: record step: %v", err)
			return
		}
		step.State, step.Message = s.runStep(ctx, c, j.Project, *step)
		switch step.State {
		case "failed":
			failed = true
		case "unknown":
			unknown = true
		case "succeeded":
			succeeded++
		}
		if err := s.Store.UpdateDockerJob(j); err != nil {
			log.Printf("docker: record step outcome: %v", err)
			return
		}
	}
	j.State, j.Message = "succeeded", "Every requested step was verified. Unchanged members are listed as skipped."
	if failed {
		j.State, j.Message = "failed", "A step failed. Review each result before creating another plan."
		if succeeded > 0 {
			j.State = "partial"
		}
	}
	if unknown {
		j.State, j.Message = "unknown", "A result could not be verified. Inspect the targets before creating another plan; no step will be replayed."
	}
	if err := s.Store.UpdateDockerJob(j); err != nil {
		log.Printf("docker: record job outcome: %v", err)
	}
}

func (s *Service) runStep(parent context.Context, c *Client, project string, step store.DockerStep) (string, string) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	if step.Kind == "verify" {
		return s.verifyProcedure(ctx, c, project, step)
	}
	if step.Kind == "container" {
		before, err := c.Inspect(ctx, step.Target)
		if err != nil {
			return "failed", "Current container state could not be read; no action was sent."
		}
		if before.Project != project || stepFingerprint(before, step.Condition) != step.Fingerprint {
			return "failed", "Container state changed after review; no action was sent."
		}
		if err = c.Mutate(ctx, step.Target, step.Action); err != nil {
			return mutationError(err, before.Secrets)
		}
		after, err := c.Inspect(ctx, step.Target)
		ok := err == nil && ((step.Action == "stop" && after.State == "exited") || (step.Action != "stop" && after.State == "running"))
		if step.Action == "restart" && after.StartedAt == before.StartedAt {
			ok = false
		}
		if !ok {
			return "unknown", "Docker accepted the action, but the resulting state could not be verified."
		}
		return "succeeded", "Container state verified: " + after.State + "."
	}
	inv, err := c.Resources(ctx)
	if err != nil {
		return "failed", "Resource references could not be read; no removal was sent."
	}
	r, err := findResource(inv, step.Kind, step.Target)
	if err != nil || !r.Removable || digest(r) != step.Fingerprint {
		return "failed", "Resource or references changed after review; no removal was sent."
	}
	if err = c.RemoveResource(ctx, step.Kind, step.Target); err != nil {
		return mutationError(err, nil)
	}
	return "succeeded", "Selected resource removal verified."
}

func mutationError(err error, secrets []string) (string, string) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return "failed", Redact(err.Error(), secrets)
	}
	return "unknown", "The result could not be verified. Inspect before retrying."
}

func (s *Service) verifyProcedure(parent context.Context, c *Client, project string, step store.DockerStep) (string, string) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	for {
		ct, err := c.Inspect(ctx, step.Target)
		if err != nil || ct.Project != project {
			return "unknown", "Verification could not read the reviewed container."
		}
		if step.Action == "stop-restart-loop" {
			if ct.State == "exited" {
				return "succeeded", "Container is stopped. The cause of repeated restarts still needs investigation."
			}
			return "failed", "The container did not remain stopped."
		}
		if ct.State != "running" {
			return "failed", "The container did not remain running."
		}
		if step.Action == "restart-high-memory" {
			stats, e := c.Stats(ctx, ct.ID)
			if e != nil || stats.LimitBytes == 0 {
				return "unknown", "Memory after restart could not be read."
			}
			percent := float64(stats.MemoryBytes) / float64(stats.LimitBytes) * 100
			if percent >= float64(step.MemoryPercent) {
				return "failed", fmt.Sprintf("Memory remains %.1f%% of the reported limit.", percent)
			}
			return "succeeded", fmt.Sprintf("Memory is %.1f%% of the reported limit. This sample does not establish a permanent fix.", percent)
		}
		if !ct.HasHealthCheck {
			if step.Action == "restart-unhealthy" {
				return "unknown", "The reviewed health check is no longer available."
			}
			return "succeeded", "Container is running without a health check. Recovery of dependent services is unverified."
		}
		if ct.Health == "healthy" {
			return "succeeded", "The container's configured health check is healthy."
		}
		select {
		case <-ctx.Done():
			return "unknown", "A healthy result was not observed within 30 seconds. Check health again."
		case <-time.After(time.Second):
		}
	}
}
