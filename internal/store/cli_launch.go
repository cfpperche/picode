package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/cfpperche/picode/internal/clilaunch"
)

type TerminalLaunch struct {
	TerminalID string              `json:"terminalId"`
	CLI        string              `json:"cli"`
	Overrides  clilaunch.Overrides `json:"overrides"`
	Applied    *clilaunch.Snapshot `json:"applied,omitempty"`
}

func (s *Store) CLIConfig(id string) (clilaunch.Config, bool, error) {
	var raw string
	err := s.db.QueryRow(`SELECT config FROM cli_configs WHERE id=?`, id).Scan(&raw)
	if err == sql.ErrNoRows {
		return clilaunch.Resolve(clilaunch.Config{}, clilaunch.Overrides{}), false, nil
	}
	if err != nil {
		return clilaunch.Config{}, false, err
	}
	var c clilaunch.Config
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return c, true, err
	}
	return clilaunch.Resolve(c, clilaunch.Overrides{}), true, nil
}

func (s *Store) SetCLIConfig(id string, c clilaunch.Config) error {
	if _, ok := clilaunch.Find(id); !ok {
		return fmt.Errorf("Unknown CLI.")
	}
	if err := clilaunch.Validate(c); err != nil {
		return err
	}
	raw, _ := json.Marshal(clilaunch.Resolve(c, clilaunch.Overrides{}))
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	if _, err = tx.Exec(`INSERT INTO cli_configs(id,config,updated_at) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET config=excluded.config, updated_at=excluded.updated_at`, id, string(raw), nowUTC()); err != nil {
		return err
	}
	if err = s.AppendEventTx(tx, "cli.updated", nil, nil, idData(id)); err != nil {
		return err
	}
	return s.commit(tx)
}

// ImportCLIConfigs is idempotent and never overwrites a saved owner choice.
func (s *Store) ImportCLIConfigs(enabled map[string]bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	for _, cli := range clilaunch.Catalog() {
		c := clilaunch.Resolve(clilaunch.Config{Integration: enabled[cli.ID]}, clilaunch.Overrides{})
		raw, _ := json.Marshal(c)
		res, err := tx.Exec(`INSERT OR IGNORE INTO cli_configs(id,config,updated_at) VALUES(?,?,?)`, cli.ID, string(raw), nowUTC())
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			if err := s.AppendEventTx(tx, "cli.updated", nil, nil, idData(cli.ID)); err != nil {
				return err
			}
		}
	}
	return s.commit(tx)
}

func (s *Store) TerminalLaunch(id string) (*TerminalLaunch, error) {
	v := &TerminalLaunch{TerminalID: id}
	var raw string
	var applied sql.NullString
	err := s.db.QueryRow(`SELECT cli,overrides,applied FROM terminal_launches WHERE terminal_id=?`, id).Scan(&v.CLI, &raw, &applied)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), &v.Overrides); err != nil {
		return nil, err
	}
	if applied.Valid {
		if err := json.Unmarshal([]byte(applied.String), &v.Applied); err != nil {
			return nil, err
		}
	}
	return v, nil
}

func (s *Store) SetTerminalLaunch(id, cli string, v clilaunch.Overrides) error {
	if _, ok := clilaunch.Find(cli); !ok {
		return fmt.Errorf("Unknown CLI.")
	}
	if _, err := s.GetTerminal(id); err != nil {
		return err
	}
	if err := clilaunch.Validate(clilaunch.Resolve(clilaunch.Config{}, v)); err != nil {
		return err
	}
	raw, _ := json.Marshal(v)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	if _, err = tx.Exec(`INSERT INTO terminal_launches(terminal_id,cli,overrides,updated_at) VALUES(?,?,?,?) ON CONFLICT(terminal_id) DO UPDATE SET cli=excluded.cli, overrides=excluded.overrides, updated_at=excluded.updated_at`, id, cli, string(raw), nowUTC()); err != nil {
		return err
	}
	if err = s.AppendEventTx(tx, "terminal.launch", nil, nil, idData(id)); err != nil {
		return err
	}
	return s.commit(tx)
}

func (s *Store) SetTerminalLaunchApplied(id string, v clilaunch.Snapshot) error {
	raw, _ := json.Marshal(v)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`UPDATE terminal_launches SET applied=?,updated_at=? WHERE terminal_id=?`, string(raw), nowUTC(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err = s.AppendEventTx(tx, "terminal.launch", nil, nil, idData(id)); err != nil {
		return err
	}
	return s.commit(tx)
}
