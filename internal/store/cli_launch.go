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
	Attempt    *clilaunch.Attempt  `json:"attempt,omitempty"`
	// LastSession pins the native CLI conversation this terminal was last
	// running (ADR-0084), so a stopped terminal can offer one-click resume
	// after a deploy, a crash or a daemon restart.
	LastSession *TerminalLastSession `json:"lastSession,omitempty"`
}

// TerminalLastSession is the pinned native session of one CLI terminal.
// It mirrors the identifying fields of a clisession.Summary without the
// store importing that package.
type TerminalLastSession struct {
	CLI        string   `json:"cli"`
	SessionID  string   `json:"sessionId"`
	Path       string   `json:"path,omitempty"`
	Cwd        string   `json:"cwd,omitempty"`
	Name       string   `json:"name,omitempty"`
	UpdatedAt  string   `json:"updatedAt"`
	Preview    string   `json:"preview,omitempty"`
	ResumeArgs []string `json:"resumeArgs,omitempty"`
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

// catalogIntegrationOn is the first-insert default: Activity reporting is
// on for every catalog CLI unless enabled.json names it false. A missing
// key used to read as false in Go, which froze OpenCode off on the first
// boot after it joined the catalog.
func catalogIntegrationOn(enabled map[string]bool, id string) bool {
	if cli, ok := clilaunch.Find(id); ok && !cli.Integrable() {
		return false
	}
	if enabled == nil {
		return true
	}
	v, ok := enabled[id]
	if !ok {
		return true
	}
	return v
}

// ImportCLIConfigs is idempotent and never overwrites a saved owner choice.
func (s *Store) ImportCLIConfigs(enabled map[string]bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	for _, cli := range clilaunch.Catalog() {
		c := clilaunch.Resolve(clilaunch.Config{Integration: catalogIntegrationOn(enabled, cli.ID)}, clilaunch.Overrides{})
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

// v4 (2026-09-17): omp joined the full rows — existing instances (flag
// already "1" on v3) reseed once so omp gets the Activity-on default the
// same first-insert path gave every other CLI. Same bump muse needed.
const catalogIntegrationSeedKey = "cli.catalog-integration-default-v4"

func isEmptyLaunchConfig(c clilaunch.Config) bool {
	return c.Executable == "" && len(c.Args) == 0 && len(c.Env) == 0 && len(c.Path) == 0
}

// SeedCatalogIntegrationDefaults turns Activity on once for catalog CLIs
// that were inserted with the empty default and Integration false (the
// OpenCode-on-first-deploy bug). A later owner off-switch is a second
// SetCLIConfig and is not revisited after this seed flag is set.
//
// allow decides which CLIs may be switched on; nil keeps the historical
// rule (every Integrable row). The server passes its integration-mechanism
// predicate, so a CLI whose launch is editable but mechanism-less is
// never seeded on — seeding it would break every default launch at
// prepare time, where integration without a mechanism is refused.
func (s *Store) SeedCatalogIntegrationDefaults(allow func(id string) bool) error {
	if v, ok, err := s.GetSetting(catalogIntegrationSeedKey); err != nil {
		return err
	} else if ok && v == "1" {
		return nil
	}
	for _, cli := range clilaunch.Catalog() {
		if !cli.Integrable() {
			continue
		}
		if allow != nil && !allow(cli.ID) {
			continue
		}
		c, found, err := s.CLIConfig(cli.ID)
		if err != nil {
			return err
		}
		if !found || c.Integration || !isEmptyLaunchConfig(c) {
			continue
		}
		c.Integration = true
		if err := s.SetCLIConfig(cli.ID, c); err != nil {
			return err
		}
	}
	return s.SetSetting(catalogIntegrationSeedKey, "1")
}

func (s *Store) TerminalLaunch(id string) (*TerminalLaunch, error) {
	v := &TerminalLaunch{TerminalID: id}
	var raw string
	var applied, attempt, lastSession sql.NullString
	err := s.db.QueryRow(`SELECT cli,overrides,applied,attempt,last_session FROM terminal_launches WHERE terminal_id=?`, id).Scan(&v.CLI, &raw, &applied, &attempt, &lastSession)
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
	if attempt.Valid {
		if err := json.Unmarshal([]byte(attempt.String), &v.Attempt); err != nil {
			return nil, err
		}
	}
	if lastSession.Valid && lastSession.String != "" {
		ls := &TerminalLastSession{}
		if err := json.Unmarshal([]byte(lastSession.String), ls); err != nil {
			return nil, err
		}
		v.LastSession = ls
	}
	return v, nil
}

// SetTerminalLastSession pins (or refreshes) the native session a terminal
// was last running (ADR-0084). Writing the same session twice is a no-op:
// state reports arrive per turn and must not flood the feed.
func (s *Store) SetTerminalLastSession(id string, v TerminalLastSession) error {
	if v.CLI == "" || v.SessionID == "" {
		return fmt.Errorf("cli and sessionId are required")
	}
	if _, err := s.GetTerminal(id); err != nil {
		return err
	}
	current, err := s.TerminalLaunch(id)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrNotFound
	}
	if cur := current.LastSession; cur != nil && cur.SessionID == v.SessionID && cur.UpdatedAt == v.UpdatedAt {
		return nil
	}
	raw, _ := json.Marshal(v)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`UPDATE terminal_launches SET last_session=?,updated_at=? WHERE terminal_id=?`, string(raw), nowUTC(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err = s.AppendEventTx(tx, "terminal.last_session", nil, nil, map[string]any{
		"id": id, "termId": id, "sessionId": v.SessionID, "cli": v.CLI,
		"path": v.Path, "cwd": v.Cwd, "name": v.Name, "updatedAt": v.UpdatedAt,
		"preview": v.Preview, "resumeArgs": v.ResumeArgs,
	}); err != nil {
		return err
	}
	return s.commit(tx)
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
	res, err := tx.Exec(`UPDATE terminal_launches SET applied=?,attempt=NULL,updated_at=? WHERE terminal_id=?`, string(raw), nowUTC(), id)
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
