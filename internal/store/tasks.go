package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// Task kinds mirror pi's RPC prompt semantics.
const (
	TaskPrompt   = "prompt"
	TaskSteer    = "steer"
	TaskFollowUp = "follow_up"
)

// Task statuses.
const (
	TaskQueued     = "queued"
	TaskDelivering = "delivering"
	TaskDelivered  = "delivered"
	TaskFailed     = "failed"
	TaskCancelled  = "cancelled"
)

// Task is a queued delivery to an agent.
type Task struct {
	ID          string  `json:"id"`
	AgentID     string  `json:"agentId"`
	Kind        string  `json:"kind"`
	Payload     string  `json:"payload"`
	Source      string  `json:"source"`
	Status      string  `json:"status"`
	Attempts    int     `json:"attempts"`
	LastError   *string `json:"lastError"`
	CreatedAt   string  `json:"createdAt"`
	DeliveredAt *string `json:"deliveredAt"`
}

const taskCols = `id, agent_id, kind, payload, source, status, attempts, last_error, created_at, delivered_at`

// EnqueueTask adds a task (status queued) and records an event.
func (s *Store) EnqueueTask(agentID, kind, payload, source string) (Task, error) {
	switch kind {
	case TaskPrompt, TaskSteer, TaskFollowUp:
	default:
		return Task{}, fmt.Errorf("store: invalid task kind %q", kind)
	}
	if stringsTrimSpace(payload) == "" {
		return Task{}, fmt.Errorf("store: task payload is required")
	}
	if source == "" {
		source = "user"
	}
	t := Task{
		ID:        newID("task", "task"),
		AgentID:   agentID,
		Kind:      kind,
		Payload:   payload,
		Source:    source,
		Status:    TaskQueued,
		CreatedAt: nowUTC(),
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Task{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var agentExists int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM agents WHERE id = ?`, agentID).Scan(&agentExists); err != nil || agentExists == 0 {
		return Task{}, ErrNotFound
	}
	if _, err := tx.Exec(`INSERT INTO tasks (id, agent_id, kind, payload, source, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.AgentID, t.Kind, t.Payload, t.Source, t.Status, t.CreatedAt); err != nil {
		return Task{}, fmt.Errorf("store: insert task: %w", err)
	}
	if err := s.AppendEventTx(tx, "task_enqueued", &agentID, nil, map[string]string{"taskId": t.ID, "kind": t.Kind}); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return t, nil
}

// ListTasks returns an agent's tasks, newest first.
func (s *Store) ListTasks(agentID string, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT `+taskCols+` FROM tasks WHERE agent_id = ? ORDER BY created_at DESC LIMIT ?`, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list tasks: %w", err)
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.AgentID, &t.Kind, &t.Payload, &t.Source, &t.Status, &t.Attempts, &t.LastError, &t.CreatedAt, &t.DeliveredAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// NextQueuedTask claims the oldest queued task for an agent (for the M2
// delivery engine) by moving it to delivering.
func (s *Store) ClaimNextTask(agentID string) (Task, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Task{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var t Task
	row := tx.QueryRow(`SELECT `+taskCols+` FROM tasks WHERE agent_id = ? AND status = ? ORDER BY created_at LIMIT 1`, agentID, TaskQueued)
	err = row.Scan(&t.ID, &t.AgentID, &t.Kind, &t.Payload, &t.Source, &t.Status, &t.Attempts, &t.LastError, &t.CreatedAt, &t.DeliveredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("store: claim task: %w", err)
	}
	if _, err := tx.Exec(`UPDATE tasks SET status = ?, attempts = attempts + 1 WHERE id = ?`, TaskDelivering, t.ID); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	t.Status = TaskDelivering
	t.Attempts++
	return t, nil
}

// FinishTask marks a claimed task delivered or failed.
func (s *Store) FinishTask(id, status string, taskErr string) error {
	var deliveredAt any
	if status == TaskDelivered {
		deliveredAt = nowUTC()
	}
	var errArg any
	if taskErr != "" {
		errArg = taskErr
	}
	res, err := s.db.Exec(`UPDATE tasks SET status = ?, delivered_at = ?, last_error = ? WHERE id = ? AND status = ?`,
		status, deliveredAt, errArg, id, TaskDelivering)
	if err != nil {
		return fmt.Errorf("store: finish task: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
