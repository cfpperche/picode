package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrLlamaConflict = errors.New("another model operation needs to finish or be checked first")

type LlamaProgress struct {
	File  string `json:"file"`
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
}

type LlamaJob struct {
	ID              string          `json:"id"`
	RequestKey      string          `json:"requestKey"`
	Endpoint        string          `json:"endpoint"`
	Model           string          `json:"model"`
	Operation       string          `json:"operation"`
	ReplaceOthers   bool            `json:"replaceOthers"`
	ConnectionID    string          `json:"-"`
	State           string          `json:"state"`
	Stage           string          `json:"stage"`
	Observed        string          `json:"observed,omitempty"`
	Message         string          `json:"message"`
	Progress        []LlamaProgress `json:"progress"`
	CancelSupported bool            `json:"cancelSupported"`
	CancelRequested bool            `json:"cancelRequested"`
	CancelAccepted  bool            `json:"cancelAccepted"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
	Revision        int             `json:"revision"`
}

func (j LlamaJob) Active() bool {
	return j.State == "queued" || j.State == "running" || j.State == "unknown"
}

func scanLlama(row interface{ Scan(...any) error }) (LlamaJob, error) {
	var j LlamaJob
	var raw, connection string
	if err := row.Scan(&raw, &connection); err != nil {
		return j, err
	}
	err := json.Unmarshal([]byte(raw), &j)
	j.ConnectionID = connection
	return j, err
}

func (s *Store) LlamaJob(id string) (LlamaJob, error) {
	return scanLlama(s.db.QueryRow(`SELECT payload, connection_id FROM llama_jobs WHERE id = ?`, id))
}

func (s *Store) LlamaJobs() ([]LlamaJob, error) {
	rows, err := s.db.Query(`SELECT payload, connection_id FROM llama_jobs WHERE state IN ('queued','running','unknown') OR id IN (SELECT id FROM llama_jobs ORDER BY created_at DESC LIMIT 50) ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LlamaJob{}
	for rows.Next() {
		j, e := scanLlama(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// BeginLlamaJob atomically deduplicates the request and reserves the model.
// Unknown outcomes retain their reservation; a replace reserves the endpoint.
func (s *Store) BeginLlamaJob(j LlamaJob) (LlamaJob, bool, error) {
	if j.RequestKey == "" || len(j.RequestKey) > 128 || j.Model == "" || len(j.Model) > 512 || j.Endpoint == "" || j.ConnectionID == "" || (j.Operation != "load" && j.Operation != "unload" && j.Operation != "download") || (j.ReplaceOthers && j.Operation != "load") {
		return j, false, fmt.Errorf("invalid model operation")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return j, false, err
	}
	defer s.rollback(tx)
	old, err := scanLlama(tx.QueryRow(`SELECT payload,connection_id FROM llama_jobs WHERE request_key=?`, j.RequestKey))
	if err == nil {
		if old.Endpoint != j.Endpoint || old.ConnectionID != j.ConnectionID || old.Model != j.Model || old.Operation != j.Operation || old.ReplaceOthers != j.ReplaceOthers {
			return old, false, ErrLlamaConflict
		}
		return old, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return j, false, err
	}
	var conflicts, total int
	err = tx.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN model=? OR replace_others=1 OR ? THEN 1 ELSE 0 END),0) FROM llama_jobs WHERE endpoint=? AND state IN ('queued','running','unknown')`, j.Model, j.ReplaceOthers, j.Endpoint).Scan(&total, &conflicts)
	if err != nil {
		return j, false, err
	}
	if conflicts > 0 || total >= 4 {
		return j, false, ErrLlamaConflict
	}
	j.ID, j.State, j.Stage, j.CreatedAt = newID("llama", "job"), "queued", "queued", nowUTC()
	j.UpdatedAt, j.Revision, j.Message = j.CreatedAt, 1, "Waiting to start."
	j.Progress = []LlamaProgress{}
	raw, _ := json.Marshal(j)
	_, err = tx.Exec(`INSERT INTO llama_jobs (id,request_key,endpoint,model,replace_others,state,connection_id,created_at,payload) VALUES (?,?,?,?,?,?,?,?,?)`, j.ID, j.RequestKey, j.Endpoint, j.Model, j.ReplaceOthers, j.State, j.ConnectionID, j.CreatedAt, string(raw))
	if err != nil {
		return j, false, err
	}
	if err = s.AppendEventTx(tx, "llama.job", nil, nil, j); err != nil {
		return j, false, err
	}
	return j, true, s.commit(tx)
}

// UpdateLlamaJob uses revision checks so cancellation and observations cannot
// overwrite each other. Identity and completed outcomes are immutable.
func (s *Store) UpdateLlamaJob(j LlamaJob) (LlamaJob, error) {
	switch j.State {
	case "queued", "running", "unknown", "succeeded", "failed", "canceled", "interrupted":
	default:
		return j, fmt.Errorf("invalid model job state")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return j, err
	}
	defer s.rollback(tx)
	old, err := scanLlama(tx.QueryRow(`SELECT payload,connection_id FROM llama_jobs WHERE id=?`, j.ID))
	if err != nil {
		return j, err
	}
	if !old.Active() || old.Revision != j.Revision {
		return old, ErrLlamaConflict
	}
	if j.RequestKey != old.RequestKey || j.Endpoint != old.Endpoint || j.ConnectionID != old.ConnectionID || j.Model != old.Model || j.Operation != old.Operation || j.ReplaceOthers != old.ReplaceOthers {
		return old, ErrLlamaConflict
	}
	j.CreatedAt = old.CreatedAt
	j.UpdatedAt = nowUTC()
	j.Revision++
	raw, err := json.Marshal(j)
	if err != nil {
		return j, err
	}
	_, err = tx.Exec(`UPDATE llama_jobs SET state=?,payload=? WHERE id=?`, j.State, string(raw), j.ID)
	if err != nil {
		return j, err
	}
	if err = s.AppendEventTx(tx, "llama.job", nil, nil, j); err != nil {
		return j, err
	}
	return j, s.commit(tx)
}
