package store

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrDockerConflict = errors.New("another Docker operation is already running or this request key was used for a different operation")

// DockerOperation records orchestration, never container logs or configuration.
type DockerOperation struct {
	ID            string `json:"id"`
	RequestKey    string `json:"requestKey"`
	Endpoint      string `json:"endpoint"`
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	Action        string `json:"action"`
	Actor         string `json:"actor"`
	AgentID       string `json:"agentId,omitempty"`
	State         string `json:"state"`
	Message       string `json:"message,omitempty"`
	CreatedAt     string `json:"createdAt"`
	FinishedAt    string `json:"finishedAt,omitempty"`
}

const dockerColumns = `id, request_key, endpoint, container_id, container_name, action, actor, agent_id, state, message, created_at, finished_at`

func scanDocker(row interface{ Scan(...any) error }) (DockerOperation, error) {
	var op DockerOperation
	err := row.Scan(&op.ID, &op.RequestKey, &op.Endpoint, &op.ContainerID, &op.ContainerName, &op.Action, &op.Actor, &op.AgentID, &op.State, &op.Message, &op.CreatedAt, &op.FinishedAt)
	return op, err
}

func (s *Store) DockerOperation(id string) (DockerOperation, error) {
	return scanDocker(s.db.QueryRow(`SELECT `+dockerColumns+` FROM docker_operations WHERE id = ?`, id))
}

func (s *Store) DockerOperationByKey(key string) (DockerOperation, error) {
	return scanDocker(s.db.QueryRow(`SELECT `+dockerColumns+` FROM docker_operations WHERE request_key = ?`, key))
}

func (s *Store) DockerOperations(limit int) ([]DockerOperation, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	// Keep every active operation visible even when newer completed jobs
	// fill the recent-history window.
	rows, err := s.db.Query(`SELECT `+dockerColumns+` FROM docker_operations WHERE state = 'running' OR id IN (SELECT id FROM docker_operations ORDER BY created_at DESC LIMIT ?) ORDER BY created_at DESC`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ops := []DockerOperation{}
	for rows.Next() {
		op, err := scanDocker(rows)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, rows.Err()
}

// BeginDockerOperation reserves the container and idempotency key atomically.
func (s *Store) BeginDockerOperation(op DockerOperation) (DockerOperation, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return op, false, err
	}
	defer s.rollback(tx)
	old, err := scanDocker(tx.QueryRow(`SELECT `+dockerColumns+` FROM docker_operations WHERE request_key = ?`, op.RequestKey))
	if err == nil {
		if old.Endpoint != op.Endpoint || old.ContainerID != op.ContainerID || old.Action != op.Action || old.AgentID != op.AgentID {
			return old, false, ErrDockerConflict
		}
		return old, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return op, false, err
	}
	op.ID, op.State, op.CreatedAt = newID("docker", "op"), "running", nowUTC()
	if err = reserveDocker(tx, op.Endpoint, "container", op.ContainerID, op.ID); err != nil {
		return op, false, err
	}
	_, err = tx.Exec(`INSERT INTO docker_operations (`+dockerColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		op.ID, op.RequestKey, op.Endpoint, op.ContainerID, op.ContainerName, op.Action, op.Actor, op.AgentID, op.State, "", op.CreatedAt, "")
	if err != nil {
		// A unique active-container/key constraint is a conflict; all other
		// database failures retain their real diagnostic.
		var n int
		if e := tx.QueryRow(`SELECT COUNT(*) FROM docker_operations WHERE request_key = ? OR (endpoint = ? AND container_id = ? AND state = 'running')`, op.RequestKey, op.Endpoint, op.ContainerID).Scan(&n); e == nil && n > 0 {
			return op, false, ErrDockerConflict
		}
		return op, false, err
	}
	if err = s.AppendEventTx(tx, "docker.operation", nil, nil, op); err != nil {
		return op, false, err
	}
	return op, true, s.commit(tx)
}

func (s *Store) FinishDockerOperation(id, state, message string) error {
	if state != "succeeded" && state != "failed" && state != "unknown" {
		return fmt.Errorf("invalid Docker outcome")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	op, err := scanDocker(tx.QueryRow(`SELECT `+dockerColumns+` FROM docker_operations WHERE id = ?`, id))
	if err != nil {
		return err
	}
	if op.State != "running" {
		return nil
	}
	op.State, op.Message, op.FinishedAt = state, message, nowUTC()
	if _, err = tx.Exec(`DELETE FROM docker_locks WHERE owner = ?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE docker_operations SET state = ?, message = ?, finished_at = ? WHERE id = ? AND state = 'running'`, state, message, op.FinishedAt, id); err != nil {
		return err
	}
	if err = s.AppendEventTx(tx, "docker.operation", nil, nil, op); err != nil {
		return err
	}
	return s.commit(tx)
}

// RecoverDockerOperations never replays a host mutation after a daemon restart.
func (s *Store) RecoverDockerOperations() error {
	rows, err := s.db.Query(`SELECT id FROM docker_operations WHERE state = 'running'`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.FinishDockerOperation(id, "unknown", "PiCode restarted before the result was recorded. Refresh the container before retrying."); err != nil {
			return err
		}
	}
	return nil
}
