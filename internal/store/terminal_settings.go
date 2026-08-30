package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// TerminalSettings returns the fields set at this scope — the global defaults
// for termopts.GlobalScope, or one terminal's overrides for a terminal id.
// A scope nobody has written is not an error: it is an empty map, which is
// what "inherit everything" looks like.
func (s *Store) TerminalSettings(scope string) (map[string]string, error) {
	var raw string
	err := s.db.QueryRow(`SELECT settings FROM terminal_settings WHERE scope = ?`, scope).Scan(&raw)
	if err == sql.ErrNoRows {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: terminal settings %q: %w", scope, err)
	}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		// A row we cannot parse is worse than no row: it would fail every read
		// of this scope forever, and the user could not even reset it from the
		// panel. Treat it as unset and let the next write replace it.
		return map[string]string{}, nil
	}
	return out, nil
}

// SetTerminalSettings replaces everything stored at this scope. An empty map
// deletes the row rather than storing "{}" — the two mean the same thing, and
// a terminal that overrides nothing should leave nothing behind.
func (s *Store) SetTerminalSettings(scope string, v map[string]string) error {
	if len(v) == 0 {
		return s.DeleteTerminalSettings(scope)
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("store: encode terminal settings: %w", err)
	}
	_, err = s.db.Exec(`INSERT INTO terminal_settings (scope, settings, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(scope) DO UPDATE SET settings = excluded.settings, updated_at = excluded.updated_at`,
		scope, string(raw), nowUTC())
	if err != nil {
		return fmt.Errorf("store: save terminal settings %q: %w", scope, err)
	}
	return nil
}

// DeleteTerminalSettings drops a scope. Deleting one that was never written is
// not an error.
func (s *Store) DeleteTerminalSettings(scope string) error {
	if _, err := s.db.Exec(`DELETE FROM terminal_settings WHERE scope = ?`, scope); err != nil {
		return fmt.Errorf("store: delete terminal settings %q: %w", scope, err)
	}
	return nil
}
