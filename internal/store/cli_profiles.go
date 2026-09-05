package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
)

type CLIProfile struct {
	ID        string           `json:"id"`
	CLI       string           `json:"cli"`
	Name      string           `json:"name"`
	Config    clilaunch.Config `json:"config"`
	UpdatedAt string           `json:"updatedAt"`
}

func (s *Store) CLIProfiles() ([]CLIProfile, error) {
	rows, err := s.db.Query(`SELECT id,cli,name,config,updated_at FROM cli_profiles ORDER BY name COLLATE NOCASE,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CLIProfile{}
	for rows.Next() {
		var p CLIProfile
		var raw string
		if err := rows.Scan(&p.ID, &p.CLI, &p.Name, &raw, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &p.Config); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) SetCLIProfile(p CLIProfile) error {
	if p.ID == "" || len(p.ID) > 80 || strings.ContainsAny(p.ID, "/\\\x00\r\n") {
		return fmt.Errorf("Invalid profile ID.")
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len(p.Name) > 80 || strings.ContainsAny(p.Name, "\x00\r\n") {
		return fmt.Errorf("Use a profile name up to 80 characters.")
	}
	if _, ok := clilaunch.Find(p.CLI); !ok {
		return fmt.Errorf("Unknown CLI.")
	}
	if err := clilaunch.Validate(p.Config); err != nil {
		return err
	}
	raw, _ := json.Marshal(clilaunch.Resolve(p.Config, clilaunch.Overrides{}))
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	var count int
	if err := tx.QueryRow(`SELECT count(*) FROM cli_profiles WHERE id<>?`, p.ID).Scan(&count); err != nil {
		return err
	}
	if count >= 64 {
		return fmt.Errorf("Remove a profile before adding another (limit 64).")
	}
	if _, err := tx.Exec(`INSERT INTO cli_profiles(id,cli,name,config,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET cli=excluded.cli,name=excluded.name,config=excluded.config,updated_at=excluded.updated_at`, p.ID, p.CLI, p.Name, string(raw), nowUTC()); err != nil {
		return err
	}
	if err := s.AppendEventTx(tx, "cli.profile", nil, nil, idData(p.ID)); err != nil {
		return err
	}
	return s.commit(tx)
}

func (s *Store) DeleteCLIProfile(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`DELETE FROM cli_profiles WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := s.AppendEventTx(tx, "cli.profile", nil, nil, idData(id)); err != nil {
		return err
	}
	return s.commit(tx)
}

func (s *Store) CLICheck(cli string) (*clilaunch.Diagnostic, error) {
	var raw string
	err := s.db.QueryRow(`SELECT diagnostic FROM cli_checks WHERE cli=?`, cli).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var d clilaunch.Diagnostic
	err = json.Unmarshal([]byte(raw), &d)
	return &d, err
}

func (s *Store) SetCLICheck(cli string, d clilaunch.Diagnostic) error {
	if _, ok := clilaunch.Find(cli); !ok {
		return fmt.Errorf("Unknown CLI.")
	}
	raw, _ := json.Marshal(d)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	if _, err := tx.Exec(`INSERT INTO cli_checks(cli,diagnostic) VALUES(?,?) ON CONFLICT(cli) DO UPDATE SET diagnostic=excluded.diagnostic`, cli, string(raw)); err != nil {
		return err
	}
	if err := s.AppendEventTx(tx, "cli.checked", nil, nil, idData(cli)); err != nil {
		return err
	}
	return s.commit(tx)
}

func (s *Store) SetTerminalLaunchAttempt(id string, a clilaunch.Attempt) error {
	raw, _ := json.Marshal(a)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`UPDATE terminal_launches SET attempt=? WHERE terminal_id=?`, string(raw), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := s.AppendEventTx(tx, "terminal.launch", nil, nil, idData(id)); err != nil {
		return err
	}
	return s.commit(tx)
}
