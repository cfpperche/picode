package store

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
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
	defer func() { s.rollback(tx) }()

	var agentExists int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM agents WHERE id = ?`, agentID).Scan(&agentExists); err != nil || agentExists == 0 {
		return Task{}, ErrNotFound
	}
	if _, err := tx.Exec(`INSERT INTO tasks (id, agent_id, kind, payload, source, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.AgentID, t.Kind, t.Payload, t.Source, t.Status, t.CreatedAt); err != nil {
		return Task{}, fmt.Errorf("store: insert task: %w", err)
	}
	if err := s.AppendEventTx(tx, "task.enqueued", &agentID, nil, t); err != nil {
		return Task{}, err
	}
	if err := s.commit(tx); err != nil {
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
	defer func() { s.rollback(tx) }()

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
	if err := s.AppendEventTx(tx, "task.claimed", &agentID, nil, map[string]string{"id": t.ID}); err != nil {
		return Task{}, err
	}
	if err := s.commit(tx); err != nil {
		return Task{}, err
	}
	t.Status = TaskDelivering
	t.Attempts++
	return t, nil
}

// ClaimTask claims one exact queued task. Transient reply bursts use the
// correlation id returned by RespondAndPark instead of draining unrelated
// work that may already be waiting for the agent.
func (s *Store) ClaimTask(agentID, taskID string) (Task, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Task{}, err
	}
	defer func() { s.rollback(tx) }()

	var t Task
	row := tx.QueryRow(`SELECT `+taskCols+` FROM tasks WHERE id = ? AND agent_id = ? AND status = ?`, taskID, agentID, TaskQueued)
	if err := row.Scan(&t.ID, &t.AgentID, &t.Kind, &t.Payload, &t.Source, &t.Status, &t.Attempts, &t.LastError, &t.CreatedAt, &t.DeliveredAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, fmt.Errorf("store: claim task: %w", err)
	}
	if _, err := tx.Exec(`UPDATE tasks SET status = ?, attempts = attempts + 1 WHERE id = ?`, TaskDelivering, t.ID); err != nil {
		return Task{}, err
	}
	if err := s.AppendEventTx(tx, "task.claimed", &agentID, nil, map[string]string{"id": t.ID}); err != nil {
		return Task{}, err
	}
	if err := s.commit(tx); err != nil {
		return Task{}, err
	}
	t.Status = TaskDelivering
	t.Attempts++
	return t, nil
}

// CountQueuedTasks reports parked work for diagnostics and tests.
func (s *Store) CountQueuedTasks(agentID string) (int, error) {
	n := 0
	err := s.db.QueryRow(`SELECT count(*) FROM tasks WHERE agent_id = ? AND status = ?`, agentID, TaskQueued).Scan(&n)
	return n, err
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
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { s.rollback(tx) }()
	res, err := tx.Exec(`UPDATE tasks SET status = ?, delivered_at = ?, last_error = ? WHERE id = ? AND status = ?`,
		status, deliveredAt, errArg, id, TaskDelivering)
	if err != nil {
		return fmt.Errorf("store: finish task: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := s.AppendEventTx(tx, "task.finished", nil, nil, map[string]string{"id": id, "status": status}); err != nil {
		return err
	}
	return s.commit(tx)
}

// EndReplyBurst closes one correlated task and optionally reopens its source
// Inbox item with the previous human response preserved. Failure/cancellation
// before durable materialization must offer a truthful retry, not leave a
// green "done" item whose reply never reached the session.
func (s *Store) EndReplyBurst(id, status, taskErr, reopenNote string) error {
	if status != TaskFailed && status != TaskCancelled {
		return fmt.Errorf("store: reply burst may end only failed or cancelled")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { s.rollback(tx) }()
	var agentID, source string
	if err := tx.QueryRow(`SELECT agent_id, source FROM tasks WHERE id = ? AND status IN (?, ?)`, id, TaskQueued, TaskDelivering).Scan(&agentID, &source); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	var errArg any
	if taskErr != "" {
		errArg = taskErr
	}
	if _, err := tx.Exec(`UPDATE tasks SET status = ?, delivered_at = NULL, last_error = ? WHERE id = ?`, status, errArg, id); err != nil {
		return fmt.Errorf("store: end reply burst: %w", err)
	}
	if err := s.AppendEventTx(tx, "task.finished", &agentID, nil, map[string]string{"id": id, "status": status}); err != nil {
		return err
	}
	if strings.TrimSpace(reopenNote) != "" && strings.HasPrefix(source, "inbox-burst:") {
		itemID := strings.TrimPrefix(source, "inbox-burst:")
		note := "\n\n> " + reopenNote
		res, err := tx.Exec(`UPDATE inbox_items SET state = ?, responded_at = NULL, snoozed_until = NULL,
			body = CASE WHEN ? = '' OR instr(body, ?) > 0 THEN body ELSE body || ? END, updated_at = ?
			WHERE id = ? AND state = ?`, InboxUnread, reopenNote, reopenNote, note, nowUTC(), itemID, InboxDone)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			var it InboxItem
			if err := scanInboxItem(tx.QueryRow(`SELECT `+inboxCols+` FROM inbox_items WHERE id = ?`, itemID), &it); err != nil {
				return err
			}
			if err := s.AppendEventTx(tx, "inbox.updated", nil, nil, it); err != nil {
				return err
			}
		}
	}
	return s.commit(tx)
}

// recoverInterruptedBursts runs once at store open. A burst's holder restores
// the TUI when the daemon disappears; this transaction restores the matching
// Inbox truth. A delivering task with a timestamp-correlated JSONL user row
// is finalized as delivered; all other queued/delivering tasks cannot be
// resumed by the vanished coordinator, so they fail and reopen the exact item
// with the previous response preserved for prefill.
func (s *Store) recoverInterruptedBursts() (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { s.rollback(tx) }()
	type interrupted struct {
		taskID    string
		agentID   string
		source    string
		status    string
		kind      string
		payload   string
		createdAt string
	}
	rows, err := tx.Query(`SELECT id, agent_id, source, status, kind, payload, created_at FROM tasks
		WHERE source LIKE 'inbox-burst:%' AND status IN (?, ?)`, TaskQueued, TaskDelivering)
	if err != nil {
		return 0, fmt.Errorf("store: find interrupted bursts: %w", err)
	}
	var pending []interrupted
	for rows.Next() {
		var row interrupted
		if err := rows.Scan(&row.taskID, &row.agentID, &row.source, &row.status, &row.kind, &row.payload, &row.createdAt); err != nil {
			_ = rows.Close()
			return 0, err
		}
		pending = append(pending, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		s.rollback(tx)
		return 0, nil
	}

	const message = "Reply delivery was interrupted by a PiCode restart. Send it again from this item."
	now := nowUTC()
	for _, row := range pending {
		itemID := strings.TrimPrefix(row.source, "inbox-burst:")
		var sessionPath string
		_ = tx.QueryRow(`SELECT session_path FROM inbox_items WHERE id = ?`, itemID).Scan(&sessionPath)
		if row.status == TaskDelivering && burstTaskMaterialized(sessionPath, row.payload, row.createdAt) {
			if _, err := tx.Exec(`UPDATE tasks SET status = ?, delivered_at = ?, last_error = NULL WHERE id = ?`, TaskDelivered, now, row.taskID); err != nil {
				return 0, err
			}
			if err := s.AppendEventTx(tx, "task.finished", &row.agentID, nil, map[string]string{"id": row.taskID, "status": TaskDelivered}); err != nil {
				return 0, err
			}
			if err := s.AppendEventTx(tx, "task.delivered", &row.agentID, nil, map[string]string{"taskId": row.taskID, "kind": row.kind}); err != nil {
				return 0, err
			}
			continue
		}
		if _, err := tx.Exec(`UPDATE tasks SET status = ?, delivered_at = NULL, last_error = ? WHERE id = ?`, TaskFailed, message, row.taskID); err != nil {
			return 0, err
		}
		if err := s.AppendEventTx(tx, "task.finished", &row.agentID, nil, map[string]string{"id": row.taskID, "status": TaskFailed}); err != nil {
			return 0, err
		}
		note := "\n\n> " + message
		res, err := tx.Exec(`UPDATE inbox_items SET state = ?, responded_at = NULL, snoozed_until = NULL,
			body = CASE WHEN instr(body, ?) = 0 THEN body || ? ELSE body END, updated_at = ?
			WHERE id = ? AND state = ?`, InboxUnread, message, note, now, itemID, InboxDone)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			continue
		}
		var it InboxItem
		if err := scanInboxItem(tx.QueryRow(`SELECT `+inboxCols+` FROM inbox_items WHERE id = ?`, itemID), &it); err != nil {
			return 0, err
		}
		if err := s.AppendEventTx(tx, "inbox.updated", nil, nil, it); err != nil {
			return 0, err
		}
	}
	if err := s.commit(tx); err != nil {
		return 0, err
	}
	return len(pending), nil
}

// burstTaskMaterialized resolves the only ambiguous crash state: a claimed
// task may have reached JSONL just before the daemon died but before SQLite was
// marked delivered. A full-payload user row at/after task creation is durable
// proof; without that proof startup reopens the item for retry.
func burstTaskMaterialized(sessionPath, payload, createdAt string) bool {
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil || strings.TrimSpace(sessionPath) == "" {
		return false
	}
	needle := strings.Join(strings.Fields(payload), " ")
	if needle == "" {
		return false
	}
	f, err := os.Open(sessionPath)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() || st.Size() == 0 {
		return false
	}
	// One interrupted turn can include large tool results after its user row.
	// Scan a generous bounded tail without loading it all into memory.
	const tail = int64(64 * 1024 * 1024)
	start := st.Size() - tail
	if start < 0 {
		start = 0
	}
	partial := false
	if start > 0 {
		one := []byte{0}
		if _, err := f.ReadAt(one, start-1); err != nil || one[0] != '\n' {
			partial = true
		}
	}
	if _, err := f.Seek(start, 0); err != nil {
		return false
	}
	scan := bufio.NewScanner(f)
	// The first tail line may be the remainder of a very large tool result.
	// Let Scanner consume the whole bounded tail so it can continue to a user
	// row after that line instead of stopping at its default/max token size.
	scan.Buffer(make([]byte, 64*1024), int(tail)+64*1024)
	if partial {
		_ = scan.Scan()
	}
	// Pi's JSONL timestamp is millisecond-granular, so the task's creation
	// instant is compared at that same precision—never against an earlier
	// millisecond that could belong to a prior identical reply.
	threshold := created.Truncate(time.Millisecond)
	for scan.Scan() {
		var row struct {
			Type      string `json:"type"`
			Timestamp string `json:"timestamp"`
			Message   struct {
				Role      string `json:"role"`
				Timestamp int64  `json:"timestamp"`
				Content   []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(scan.Bytes(), &row) != nil || row.Type != "message" || row.Message.Role != "user" {
			continue
		}
		stamp, stampErr := time.Parse(time.RFC3339Nano, row.Timestamp)
		if stampErr != nil && row.Message.Timestamp > 0 {
			stamp = time.UnixMilli(row.Message.Timestamp)
			stampErr = nil
		}
		if stampErr != nil || stamp.Before(threshold) {
			continue
		}
		var text strings.Builder
		for _, part := range row.Message.Content {
			text.WriteByte(' ')
			text.WriteString(part.Text)
		}
		if strings.Join(strings.Fields(text.String()), " ") == needle {
			return true
		}
	}
	return false
}
