package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// DockerStep contains only reviewed identifiers, observations and outcomes.
// Fingerprints never include logs or environment values.
type DockerStep struct {
	Kind          string `json:"kind"`
	Target        string `json:"target"`
	Name          string `json:"name"`
	Action        string `json:"action"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	MemoryPercent int    `json:"memoryPercent,omitempty"`
	Condition     string `json:"condition,omitempty"`
	State         string `json:"state"`
	Message       string `json:"message,omitempty"`
}

type DockerPlan struct {
	ID          string          `json:"id"`
	Kind        string          `json:"kind"`
	Title       string          `json:"title"`
	Impact      string          `json:"impact"`
	Endpoint    string          `json:"endpoint"`
	Project     string          `json:"project"`
	Actor       string          `json:"actor"`
	AgentID     string          `json:"agentId,omitempty"`
	CreatedAt   string          `json:"createdAt"`
	ExpiresAt   string          `json:"expiresAt"`
	Fingerprint string          `json:"fingerprint"`
	Input       json.RawMessage `json:"input"`
	Steps       []DockerStep    `json:"steps"`
	InboxID     string          `json:"inboxId,omitempty"`
}

type DockerJob struct {
	ID              string       `json:"id"`
	RequestKey      string       `json:"requestKey"`
	PlanID          string       `json:"planId"`
	Kind            string       `json:"kind"`
	Title           string       `json:"title"`
	Endpoint        string       `json:"endpoint"`
	Project         string       `json:"project"`
	Actor           string       `json:"actor"`
	AgentID         string       `json:"agentId,omitempty"`
	ApprovedBy      string       `json:"approvedBy"`
	ApproverAgentID string       `json:"approverAgentId,omitempty"`
	State           string       `json:"state"`
	Message         string       `json:"message,omitempty"`
	Steps           []DockerStep `json:"steps"`
	CreatedAt       string       `json:"createdAt"`
	FinishedAt      string       `json:"finishedAt,omitempty"`
}

func scanDockerJSON[T any](row interface{ Scan(...any) error }) (T, error) {
	var v T
	var raw string
	if err := row.Scan(&raw); err != nil {
		return v, err
	}
	err := json.Unmarshal([]byte(raw), &v)
	return v, err
}

func (s *Store) DockerPlan(id string) (DockerPlan, error) {
	return scanDockerJSON[DockerPlan](s.db.QueryRow(`SELECT payload FROM docker_plans WHERE id = ?`, id))
}

func (s *Store) CreateDockerPlan(p DockerPlan) (DockerPlan, error) {
	p.ID, p.CreatedAt = newID("docker", "plan"), nowUTC()
	p.ExpiresAt = time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339Nano)
	tx, err := s.db.Begin()
	if err != nil {
		return p, err
	}
	defer s.rollback(tx)
	raw, err := json.Marshal(p)
	if err != nil {
		return p, err
	}
	if _, err = tx.Exec(`INSERT INTO docker_plans (id, payload) VALUES (?, ?)`, p.ID, string(raw)); err != nil {
		return p, err
	}
	if err = s.AppendEventTx(tx, "docker.plan", nil, nil, map[string]string{"id": p.ID}); err != nil {
		return p, err
	}
	return p, s.commit(tx)
}

// RequestDockerReview files one Inbox link per preview. Inbox triage is never
// permission to execute; the owner reviews and confirms in the Docker App.
func (s *Store) RequestDockerReview(id string) (DockerPlan, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return DockerPlan{}, err
	}
	defer s.rollback(tx)
	p, err := scanDockerJSON[DockerPlan](tx.QueryRow(`SELECT payload FROM docker_plans WHERE id = ?`, id))
	if err != nil || p.InboxID != "" {
		return p, err
	}
	now := nowUTC()
	it := InboxItem{ID: newID("Docker review", "inb"), Kind: InboxFYI, SourceKind: InboxFromSystem,
		SourceID: p.ID, Reason: "docker-plan", Title: "Docker maintenance needs review",
		Body:    "A Docker maintenance plan is waiting for your review. No operation has started.\n\n[Review plan](#/app/docker/plan/" + p.ID + ")\n\nExpired plans require a fresh preview before confirmation.",
		Allowed: []string{VerbIgnore}, State: InboxUnread, CreatedAt: now, UpdatedAt: now}
	if _, err = tx.Exec(`INSERT INTO inbox_items
		(id, kind, source_kind, source_id, workspace_id, reason, title, body, blocking, allowed_responses, state, created_at, updated_at, session_path)
		VALUES (?, ?, ?, ?, '', ?, ?, ?, 0, '["ignore"]', ?, ?, ?, '')`,
		it.ID, it.Kind, it.SourceKind, it.SourceID, it.Reason, it.Title, it.Body, it.State, now, now); err != nil {
		return p, err
	}
	p.InboxID = it.ID
	raw, _ := json.Marshal(p)
	if _, err = tx.Exec(`UPDATE docker_plans SET payload = ? WHERE id = ?`, string(raw), id); err != nil {
		return p, err
	}
	if err = s.AppendEventTx(tx, "inbox.created", nil, nil, it); err != nil {
		return p, err
	}
	if err = s.AppendEventTx(tx, "docker.plan", nil, nil, map[string]string{"id": p.ID}); err != nil {
		return p, err
	}
	return p, s.commit(tx)
}

func (s *Store) DockerJob(id string) (DockerJob, error) {
	return scanDockerJSON[DockerJob](s.db.QueryRow(`SELECT payload FROM docker_jobs WHERE id = ?`, id))
}

func (s *Store) DockerJobByKey(key string) (DockerJob, error) {
	return scanDockerJSON[DockerJob](s.db.QueryRow(`SELECT payload FROM docker_jobs WHERE request_key = ?`, key))
}

func (s *Store) DockerJobForPlan(id string) (DockerJob, error) {
	return scanDockerJSON[DockerJob](s.db.QueryRow(`SELECT payload FROM docker_jobs WHERE plan_id = ?`, id))
}

func (s *Store) DockerJobs() ([]DockerJob, error) {
	rows, err := s.db.Query(`SELECT payload FROM docker_jobs WHERE state = 'running' OR id IN
		(SELECT id FROM docker_jobs ORDER BY created_at DESC LIMIT 50) ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DockerJob{}
	for rows.Next() {
		j, err := scanDockerJSON[DockerJob](rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func reserveDocker(tx *sql.Tx, endpoint, kind, target, owner string) error {
	_, err := tx.Exec(`INSERT INTO docker_locks (endpoint, kind, target, owner) VALUES (?, ?, ?, ?)`, endpoint, kind, target, owner)
	if err != nil {
		var existing string
		if e := tx.QueryRow(`SELECT owner FROM docker_locks WHERE endpoint = ? AND kind = ? AND target = ?`, endpoint, kind, target).Scan(&existing); e == nil {
			return ErrDockerConflict
		}
	}
	return err
}

// BeginDockerJob reserves all targets in one transaction, including skipped
// project members whose reviewed state must remain unchanged within PiCode.
func (s *Store) BeginDockerJob(j DockerJob) (DockerJob, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return j, false, err
	}
	defer s.rollback(tx)
	old, err := scanDockerJSON[DockerJob](tx.QueryRow(`SELECT payload FROM docker_jobs WHERE request_key = ?`, j.RequestKey))
	if err == nil {
		if old.PlanID != j.PlanID || old.ApproverAgentID != j.ApproverAgentID {
			return old, false, ErrDockerConflict
		}
		return old, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return j, false, err
	}
	var used int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM docker_jobs WHERE plan_id = ?`, j.PlanID).Scan(&used); err != nil {
		return j, false, err
	}
	if used > 0 {
		return j, false, ErrDockerConflict
	}
	j.ID, j.State, j.CreatedAt = newID("docker", "job"), "running", nowUTC()
	j.Message = "Execution in progress. Step results update below."
	reserved := map[string]bool{}
	for _, step := range j.Steps {
		if step.Kind == "verify" {
			continue
		}
		key := step.Kind + ":" + step.Target
		if reserved[key] {
			continue
		}
		if err = reserveDocker(tx, j.Endpoint, step.Kind, step.Target, j.ID); err != nil {
			return j, false, err
		}
		reserved[key] = true
	}
	raw, err := json.Marshal(j)
	if err != nil {
		return j, false, err
	}
	if _, err = tx.Exec(`INSERT INTO docker_jobs (id, request_key, plan_id, state, created_at, payload) VALUES (?, ?, ?, ?, ?, ?)`, j.ID, j.RequestKey, j.PlanID, j.State, j.CreatedAt, string(raw)); err != nil {
		return j, false, err
	}
	if err = s.AppendEventTx(tx, "docker.job", nil, nil, j); err != nil {
		return j, false, err
	}
	return j, true, s.commit(tx)
}

// UpdateDockerJob advances a running job and releases its reservations only
// with a durable final result. Completed jobs cannot be rewritten or replayed.
func (s *Store) UpdateDockerJob(j DockerJob) error {
	if j.State != "running" && j.State != "succeeded" && j.State != "partial" && j.State != "failed" && j.State != "unknown" {
		return fmt.Errorf("invalid Docker job outcome")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	old, err := scanDockerJSON[DockerJob](tx.QueryRow(`SELECT payload FROM docker_jobs WHERE id = ?`, j.ID))
	if err != nil || old.State != "running" {
		return err
	}
	// Immutable provenance and targets come from the accepted job.
	if old.PlanID != j.PlanID || old.RequestKey != j.RequestKey || len(old.Steps) != len(j.Steps) {
		return fmt.Errorf("Docker job identity changed")
	}
	for i := range old.Steps {
		if old.Steps[i].Kind != j.Steps[i].Kind || old.Steps[i].Target != j.Steps[i].Target || old.Steps[i].Action != j.Steps[i].Action {
			return fmt.Errorf("Docker job targets changed")
		}
		old.Steps[i].State, old.Steps[i].Message = j.Steps[i].State, j.Steps[i].Message
	}
	old.State, old.Message = j.State, j.Message
	if j.State != "running" {
		old.FinishedAt = nowUTC()
		if _, err = tx.Exec(`DELETE FROM docker_locks WHERE owner = ?`, j.ID); err != nil {
			return err
		}
	}
	raw, _ := json.Marshal(old)
	if _, err = tx.Exec(`UPDATE docker_jobs SET state = ?, payload = ? WHERE id = ?`, old.State, string(raw), old.ID); err != nil {
		return err
	}
	if err = s.AppendEventTx(tx, "docker.job", nil, nil, old); err != nil {
		return err
	}
	return s.commit(tx)
}

func (s *Store) RecoverDockerJobs() error {
	jobs, err := s.DockerJobs()
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.State != "running" {
			continue
		}
		j.State, j.Message = "unknown", "PiCode restarted before the result was recorded. Inspect the targets before creating a new plan."
		for i := range j.Steps {
			switch j.Steps[i].State {
			case "running":
				j.Steps[i].State, j.Steps[i].Message = "unknown", "Interrupted; not replayed."
			case "queued":
				j.Steps[i].State, j.Steps[i].Message = "skipped", "PiCode restarted; not executed."
			}
		}
		if err = s.UpdateDockerJob(j); err != nil {
			return err
		}
	}
	return nil
}
