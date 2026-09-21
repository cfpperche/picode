package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// ErrCLILifecycleConflict mirrors ErrLlamaConflict: another lifecycle job
// must finish first, or the request key already exists with different input.
var ErrCLILifecycleConflict = errors.New("another CLI lifecycle job needs to finish first")

// CLIJob is one durable lifecycle operation: update, reinstall or uninstall
// of a catalogued agent CLI (ADR-0087). PiCode never replays a job after a
// restart — interrupted stays interrupted.
type CLIJob struct {
	ID         string `json:"id"`
	CLI        string `json:"cli"`
	Action     string `json:"action"`
	RequestKey string `json:"requestKey"`
	// Payload carries the arguments of the operation for the actions whose
	// argv is not derivable from (cli, action) alone — the plugin jobs of
	// ADR-0167 name a target, a scope and a workspace. It lives in the same
	// JSON document as the rest of the row, so no schema change was needed.
	Payload   string `json:"payload,omitempty"`
	State     string `json:"state"` // queued running succeeded failed interrupted
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Revision  int    `json:"revision"`
}

func (j CLIJob) Active() bool { return j.State == "queued" || j.State == "running" }

// jobActions is the closed vocabulary of durable CLI jobs: the lifecycle
// actions of ADR-0087 plus the plugin actions of ADR-0167.
var jobActions = map[string]bool{
	"install": true, "update": true, "reinstall": true, "uninstall": true,
	"pkg-install": true, "pkg-remove": true, "pkg-update": true,
	"pkg-marketplace-add": true, "pkg-marketplace-update": true,
}

func validJobAction(action string) bool { return jobActions[action] }

func scanCLIJob(row interface{ Scan(...any) error }) (CLIJob, error) {
	var j CLIJob
	var raw string
	if err := row.Scan(&raw); err != nil {
		return j, err
	}
	err := json.Unmarshal([]byte(raw), &j)
	return j, err
}

// CLIJob reads one job by ID.
func (s *Store) CLIJob(id string) (CLIJob, error) {
	return scanCLIJob(s.db.QueryRow(`SELECT payload FROM cli_jobs WHERE id = ?`, id))
}

// CLIJobs returns every active job plus the most recent 50 records.
func (s *Store) CLIJobs() ([]CLIJob, error) {
	rows, err := s.db.Query(`SELECT payload FROM cli_jobs WHERE state IN ('queued','running') OR id IN (SELECT id FROM cli_jobs ORDER BY created_at DESC LIMIT 50) ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CLIJob{}
	for rows.Next() {
		j, e := scanCLIJob(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// BeginCLIJob atomically deduplicates the request key and enforces at most
// one active lifecycle job at a time. Store write and feed event commit
// together (ADR-0048).
func (s *Store) BeginCLIJob(j CLIJob) (CLIJob, bool, error) {
	if j.RequestKey == "" || len(j.RequestKey) > 128 || !validJobAction(j.Action) {
		return j, false, fmt.Errorf("invalid CLI lifecycle request")
	}
	if _, ok := clilaunch.Find(j.CLI); !ok {
		return j, false, fmt.Errorf("unknown CLI")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return j, false, err
	}
	defer s.rollback(tx)
	old, err := scanCLIJob(tx.QueryRow(`SELECT payload FROM cli_jobs WHERE request_key=?`, j.RequestKey))
	if err == nil {
		// One request key means one operation: the same key with a different
		// target is a conflict, not a cache hit (a plugin job names its
		// target in the payload).
		if old.CLI != j.CLI || old.Action != j.Action || old.Payload != j.Payload {
			return old, false, ErrCLILifecycleConflict
		}
		return old, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return j, false, err
	}
	var active int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM cli_jobs WHERE state IN ('queued','running')`).Scan(&active); err != nil {
		return j, false, err
	}
	if active > 0 {
		return j, false, ErrCLILifecycleConflict
	}
	j.ID, j.State, j.CreatedAt = newID("cli", "job"), "queued", nowUTC()
	j.UpdatedAt, j.Revision, j.Message = j.CreatedAt, 1, "Waiting to start."
	raw, _ := json.Marshal(j)
	if _, err := tx.Exec(`INSERT INTO cli_jobs (id,cli,action,request_key,state,created_at,payload) VALUES (?,?,?,?,?,?,?)`, j.ID, j.CLI, j.Action, j.RequestKey, j.State, j.CreatedAt, string(raw)); err != nil {
		return j, false, err
	}
	if err := s.AppendEventTx(tx, "cli.job", nil, nil, j); err != nil {
		return j, false, err
	}
	return j, true, s.commit(tx)
}

// UpdateCLIJob guards state transitions with a revision check.
func (s *Store) UpdateCLIJob(j CLIJob) (CLIJob, error) {
	switch j.State {
	case "queued", "running", "succeeded", "failed", "interrupted":
	default:
		return j, fmt.Errorf("invalid CLI job state")
	}
	if len(j.Output) > 8192 {
		j.Output = j.Output[len(j.Output)-8192:]
	}
	tx, err := s.db.Begin()
	if err != nil {
		return j, err
	}
	defer s.rollback(tx)
	old, err := scanCLIJob(tx.QueryRow(`SELECT payload FROM cli_jobs WHERE id=?`, j.ID))
	if err != nil {
		return j, err
	}
	if old.State != "queued" && old.State != "running" || old.Revision != j.Revision {
		return old, ErrCLILifecycleConflict
	}
	if old.RequestKey != j.RequestKey || old.CLI != j.CLI || old.Action != j.Action {
		return old, ErrCLILifecycleConflict
	}
	j.CreatedAt = old.CreatedAt
	j.UpdatedAt = nowUTC()
	j.Revision++
	raw, err := json.Marshal(j)
	if err != nil {
		return j, err
	}
	if _, err := tx.Exec(`UPDATE cli_jobs SET state=?,payload=? WHERE id=?`, j.State, string(raw), j.ID); err != nil {
		return j, err
	}
	if err := s.AppendEventTx(tx, "cli.job", nil, nil, j); err != nil {
		return j, err
	}
	return j, s.commit(tx)
}
