package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// IntegrationSettings is how a project integrates (ADR-0182): the operation the
// queue runs — whether the target accepts only fast-forwards, and the commands
// that must pass before the operation counts as done. The declaration is
// configuration; authority stays in the queue's own rows.
//
// Scope follows the owner's rule for the queue: a workspace declares first, and
// a workspace that declares nothing falls back to the machine default.
type IntegrationSettings struct {
	Scope string `json:"scope"`
	// Mode is the engine the project declared (ADR-0186): its own provider queue,
	// or PiCode's local runner. Empty means nothing was declared, which is a
	// named blocker — never a guess about which engine to use.
	Mode      string   `json:"mode,omitempty"`
	FFOnly    bool     `json:"ffOnly"`
	Checks    []string `json:"checks,omitempty"`
	Version   int      `json:"version"`
	UpdatedAt string   `json:"updatedAt"`
	FromScope string   `json:"fromScope,omitempty"` // the layer the effective value came from
}

// ErrIntegrationConflict: the declaration changed since the writer read it.
var ErrIntegrationConflict = errors.New("integration settings changed since they were read")

// MachineIntegrationScope is the scope key of the machine default. Every other
// key is a workspace id.
const MachineIntegrationScope = ""

// MaxIntegrationChecks bounds how many commands a project may declare.
const MaxIntegrationChecks = 8

// The integration modes a project may declare (ADR-0186). Under Provider the
// project's own queue performs the integration and PiCode only enqueues and
// observes it; under Local the runner here does, with the declared commands.
const (
	ModeProvider = "provider"
	ModeLocal    = "local"
)

// IntegrationSettingsMutation is one declaration write.
type IntegrationSettingsMutation struct {
	Mode            string   `json:"mode,omitempty"`
	FFOnly          bool     `json:"ffOnly"`
	Checks          []string `json:"checks,omitempty"`
	ExpectedVersion int      `json:"expectedVersion,omitempty"`
}

// ValidateIntegrationSettings checks a declaration's shape: at most eight
// commands, each one non-empty line of at most 300 bytes. Nothing runs them.
func ValidateIntegrationSettings(m IntegrationSettingsMutation) error {
	switch m.Mode {
	case "", ModeProvider, ModeLocal:
	default:
		return fmt.Errorf("mode must be %q or %q", ModeProvider, ModeLocal)
	}
	// The provider's own CI runs its checks; declared commands belong to the
	// local runner, and carrying both would say two engines must run.
	if m.Mode == ModeProvider && len(m.Checks) > 0 {
		return errors.New("provider mode runs the checks of its provider; declare local to run commands here")
	}
	if len(m.Checks) > MaxIntegrationChecks {
		return fmt.Errorf("at most %d checks", MaxIntegrationChecks)
	}
	for i, c := range m.Checks {
		if strings.TrimSpace(c) == "" || len(c) > 300 || strings.ContainsAny(c, "\r\n") {
			return fmt.Errorf("check %d must be one non-empty line, at most 300 bytes", i+1)
		}
	}
	return nil
}

// IntegrationSettingsFor reads one layer. known is false when the layer has no
// declaration of its own — which is a fact, not an error.
func (s *Store) IntegrationSettingsFor(scope string) (IntegrationSettings, bool, error) {
	var raw string
	err := s.db.QueryRow(`SELECT body FROM delivery_integration WHERE scope=?`, scope).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return IntegrationSettings{}, false, nil
	}
	if err != nil {
		return IntegrationSettings{}, false, err
	}
	var v IntegrationSettings
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return IntegrationSettings{}, false, err
	}
	v.FromScope = scope
	return v, true, nil
}

// ListIntegrationSettings reads every declared layer at once, keyed by scope
// ("" for the machine) — the Preferences page's "who follows these rules"
// list, which would otherwise cost one read per workspace.
func (s *Store) ListIntegrationSettings() (map[string]IntegrationSettings, error) {
	rows, err := s.db.Query(`SELECT scope, body FROM delivery_integration`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]IntegrationSettings{}
	for rows.Next() {
		var scope, raw string
		if err := rows.Scan(&scope, &raw); err != nil {
			return nil, err
		}
		var v IntegrationSettings
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, err
		}
		v.FromScope = scope
		out[scope] = v
	}
	return out, rows.Err()
}

// EffectiveIntegrationSettings is the workspace-first fallback: the workspace's
// declaration when it has one, the machine's otherwise, and a default that
// requires nothing beyond a fast-forward when neither exists.
func (s *Store) EffectiveIntegrationSettings(workspaceID string) (IntegrationSettings, error) {
	if workspaceID != MachineIntegrationScope {
		if v, ok, err := s.IntegrationSettingsFor(workspaceID); err != nil {
			return IntegrationSettings{}, err
		} else if ok {
			return v, nil
		}
	}
	if v, ok, err := s.IntegrationSettingsFor(MachineIntegrationScope); err != nil {
		return IntegrationSettings{}, err
	} else if ok {
		return v, nil
	}
	return IntegrationSettings{Scope: MachineIntegrationScope, FFOnly: true, Version: 0, FromScope: "default"}, nil
}

// PutIntegrationSettings writes one layer and announces it, so every surface
// that shows the declaration refetches instead of guessing.
func (s *Store) PutIntegrationSettings(scope string, m IntegrationSettingsMutation) (IntegrationSettings, error) {
	if err := ValidateIntegrationSettings(m); err != nil {
		return IntegrationSettings{}, err
	}
	s.deliveryMu.Lock()
	defer s.deliveryMu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return IntegrationSettings{}, err
	}
	defer s.rollback(tx)
	next := IntegrationSettings{Scope: scope, Mode: m.Mode, FFOnly: m.FFOnly, Checks: append([]string{}, m.Checks...), UpdatedAt: nowUTC()}
	var raw string
	err = tx.QueryRow(`SELECT body FROM delivery_integration WHERE scope=?`, scope).Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return IntegrationSettings{}, err
	}
	current := 0
	if err == nil {
		var cur IntegrationSettings
		if uerr := json.Unmarshal([]byte(raw), &cur); uerr != nil {
			return IntegrationSettings{}, uerr
		}
		current = cur.Version
	}
	// expectedVersion is the version the writer read (0 = "none declared"
	// only when it says so; omitted means the writer does not care). Two
	// settings tabs saving over each other get a conflict, not a silent loss.
	if m.ExpectedVersion > 0 && m.ExpectedVersion != current {
		return IntegrationSettings{}, ErrIntegrationConflict
	}
	next.Version = current + 1
	body, _ := json.Marshal(next)
	if _, err = tx.Exec(`INSERT INTO delivery_integration(scope,body) VALUES(?,?)
		ON CONFLICT(scope) DO UPDATE SET body=excluded.body`, scope, string(body)); err != nil {
		return IntegrationSettings{}, err
	}
	if err = s.AppendEventTx(tx, "delivery.changed", nil, nil, map[string]any{"integration": scope, "version": next.Version}); err != nil {
		return IntegrationSettings{}, err
	}
	if err := s.commit(tx); err != nil {
		return IntegrationSettings{}, err
	}
	return next, nil
}

// DeleteIntegrationSettings drops a scope's own declaration, so the workspace
// inherits the machine's again (or the machine the built-in default). Absent
// is not an error: the caller asked for "inherit", and it now does.
func (s *Store) DeleteIntegrationSettings(scope string) error {
	s.deliveryMu.Lock()
	defer s.deliveryMu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`DELETE FROM delivery_integration WHERE scope=?`, scope)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return s.commit(tx)
	}
	if err := s.AppendEventTx(tx, "delivery.changed", nil, nil, map[string]any{"integration": scope, "removed": true}); err != nil {
		return err
	}
	return s.commit(tx)
}
