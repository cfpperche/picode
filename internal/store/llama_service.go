package store

import (
	"database/sql"
	"encoding/json"
	"errors"
)

// LlamaService returns the durable owned-service document and its CAS revision.
func (s *Store) LlamaService() (json.RawMessage, int, error) {
	var raw string
	var revision int
	err := s.db.QueryRow(`SELECT payload,revision FROM llama_service WHERE id=1`).Scan(&raw, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return json.RawMessage(`{}`), 0, nil
	}
	return json.RawMessage(raw), revision, err
}

// SaveLlamaService atomically persists configuration/job state and announces it.
func (s *Store) SaveLlamaService(raw json.RawMessage, revision int) (int, error) {
	if !json.Valid(raw) {
		return revision, errors.New("invalid service document")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return revision, err
	}
	defer s.rollback(tx)
	result, err := tx.Exec(`INSERT INTO llama_service(id,revision,payload) SELECT 1,1,? WHERE ?=0 ON CONFLICT(id) DO NOTHING`, string(raw), revision)
	if err != nil {
		return revision, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return revision, err
	}
	if n == 0 {
		result, err = tx.Exec(`UPDATE llama_service SET payload=?,revision=revision+1 WHERE id=1 AND revision=?`, string(raw), revision)
		if err != nil {
			return revision, err
		}
		n, err = result.RowsAffected()
		if err != nil {
			return revision, err
		}
	}
	if n != 1 {
		return revision, ErrLlamaConflict
	}
	if err = s.AppendEventTx(tx, "llama.service", nil, nil, map[string]int{"revision": revision + 1}); err != nil {
		return revision, err
	}
	return revision + 1, s.commit(tx)
}
