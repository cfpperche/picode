package store

import (
	"encoding/json"
	"fmt"
	"time"
)

type DockerMonitor struct {
	Endpoint        string          `json:"endpoint"`
	Project         string          `json:"project"`
	Enabled         bool            `json:"enabled"`
	IntervalSeconds int             `json:"intervalSeconds"`
	CPUPercent      int             `json:"cpuPercent"`
	MemoryPercent   int             `json:"memoryPercent"`
	BadSamples      int             `json:"badSamples"`
	RetentionDays   int             `json:"retentionDays"`
	Revision        int             `json:"revision"`
	Actor           string          `json:"actor"`
	UpdatedAt       string          `json:"updatedAt"`
	SampledAt       string          `json:"sampledAt,omitempty"`
	EvaluatedAt     string          `json:"evaluatedAt,omitempty"`
	Snapshot        json.RawMessage `json:"snapshot,omitempty"`
}

type DockerSignal struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	State  string `json:"state"` // good | bad | unknown
	Detail string `json:"detail"`
}

type DockerIncident struct {
	ID         string `json:"id"`
	Endpoint   string `json:"endpoint"`
	Project    string `json:"project"`
	Signal     string `json:"signal"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	State      string `json:"state"` // pending | open | resolved
	BadStreak  int    `json:"badStreak"`
	GoodStreak int    `json:"goodStreak"`
	Revision   int    `json:"revision"`
	CreatedAt  string `json:"createdAt"`
	OpenedAt   string `json:"openedAt,omitempty"`
	UpdatedAt  string `json:"updatedAt"`
	ResolvedAt string `json:"resolvedAt,omitempty"`
}

func DefaultDockerMonitor(endpoint, project string) DockerMonitor {
	return DockerMonitor{Endpoint: endpoint, Project: project, IntervalSeconds: 60, CPUPercent: 90, MemoryPercent: 85, BadSamples: 3, RetentionDays: 7}
}

func (m DockerMonitor) Validate() error {
	if m.Endpoint == "" || len(m.Project) > 256 || (m.IntervalSeconds != 30 && m.IntervalSeconds != 60 && m.IntervalSeconds != 300) ||
		(m.CPUPercent != 80 && m.CPUPercent != 90 && m.CPUPercent != 200) || (m.MemoryPercent != 80 && m.MemoryPercent != 85 && m.MemoryPercent != 95) ||
		(m.BadSamples != 2 && m.BadSamples != 3 && m.BadSamples != 5) || (m.RetentionDays != 7 && m.RetentionDays != 30) {
		return fmt.Errorf("Choose one of the available monitoring settings")
	}
	return nil
}

func (s *Store) DockerMonitor(endpoint, project string) (DockerMonitor, error) {
	return scanDockerJSON[DockerMonitor](s.db.QueryRow(`SELECT payload FROM docker_monitors WHERE endpoint = ? AND project = ?`, endpoint, project))
}

func (s *Store) DockerMonitors() ([]DockerMonitor, error) {
	rows, err := s.db.Query(`SELECT payload FROM docker_monitors ORDER BY endpoint,project`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DockerMonitor{}
	for rows.Next() {
		m, err := scanDockerJSON[DockerMonitor](rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SaveDockerMonitor uses a revision to reject stale forms and in-flight
// samples. Snapshots are retained as dated evidence when monitoring stops.
func (s *Store) SaveDockerMonitor(m DockerMonitor) (DockerMonitor, error) {
	if err := m.Validate(); err != nil {
		return m, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return m, err
	}
	defer s.rollback(tx)
	var revision int
	if err = tx.QueryRow(`SELECT COALESCE(MAX(revision),0) FROM docker_monitors WHERE endpoint = ? AND project = ?`, m.Endpoint, m.Project).Scan(&revision); err != nil {
		return m, err
	}
	if revision != m.Revision {
		return m, ErrDockerConflict
	}
	if revision > 0 {
		old, e := scanDockerJSON[DockerMonitor](tx.QueryRow(`SELECT payload FROM docker_monitors WHERE endpoint = ? AND project = ?`, m.Endpoint, m.Project))
		if e != nil {
			return m, e
		}
		m.Snapshot, m.SampledAt = old.Snapshot, old.SampledAt
	}
	if m.Enabled {
		// A bound on work per daemon, including disconnected pinned engines.
		var total int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM docker_monitors WHERE json_extract(payload,'$.enabled') = 1 AND NOT (endpoint = ? AND project = ?)`, m.Endpoint, m.Project).Scan(&total); err != nil {
			return m, err
		}
		if total >= 32 {
			return m, fmt.Errorf("Up to 32 projects can be monitored. Disable another project first")
		}
	}
	m.Revision, m.UpdatedAt, m.EvaluatedAt = revision+1, nowUTC(), ""
	raw, _ := json.Marshal(m)
	if _, err = tx.Exec(`INSERT INTO docker_monitors (endpoint,project,revision,payload) VALUES (?,?,?,?) ON CONFLICT(endpoint,project) DO UPDATE SET revision=excluded.revision,payload=excluded.payload`, m.Endpoint, m.Project, m.Revision, string(raw)); err != nil {
		return m, err
	}
	if err = s.AppendEventTx(tx, "docker.monitor", nil, nil, map[string]any{"endpoint": m.Endpoint, "project": m.Project, "enabled": m.Enabled, "revision": m.Revision}); err != nil {
		return m, err
	}
	return m, s.commit(tx)
}

func (s *Store) DockerIncidents(endpoint, project string) ([]DockerIncident, error) {
	rows, err := s.db.Query(`SELECT payload FROM docker_incidents WHERE endpoint = ? AND project = ? AND
		(state = 'open' OR id IN (SELECT id FROM docker_incidents WHERE endpoint = ? AND project = ? AND state != 'pending' ORDER BY updated_at DESC LIMIT 100)) ORDER BY updated_at DESC`, endpoint, project, endpoint, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DockerIncident{}
	for rows.Next() {
		v, err := scanDockerJSON[DockerIncident](rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// RecordDockerHealth writes one snapshot and its incident transitions together.
// Signals absent from an observation are unknown, not evidence of recovery.
func (s *Store) RecordDockerHealth(endpoint, project string, revision int, snapshot json.RawMessage, signals []DockerSignal, at time.Time) error {
	if !json.Valid(snapshot) || len(snapshot) > 1024*1024 {
		return fmt.Errorf("Invalid or oversized Docker health snapshot")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	m, err := scanDockerJSON[DockerMonitor](tx.QueryRow(`SELECT payload FROM docker_monitors WHERE endpoint = ? AND project = ?`, endpoint, project))
	if err != nil {
		return err
	}
	if m.Revision != revision {
		return ErrDockerConflict
	}
	previousSample, _ := time.Parse(time.RFC3339Nano, m.SampledAt)
	if !previousSample.IsZero() && !at.After(previousSample) {
		return nil
	}
	m.Snapshot, m.SampledAt = snapshot, at.UTC().Format(time.RFC3339Nano)
	last, _ := time.Parse(time.RFC3339Nano, m.EvaluatedAt)
	if m.Enabled && (last.IsZero() || at.Sub(last) >= time.Duration(m.IntervalSeconds)*time.Second) {
		observed := map[string]DockerSignal{}
		for _, sig := range signals {
			if sig.State != "good" && sig.State != "bad" && sig.State != "unknown" {
				return fmt.Errorf("Invalid Docker health signal")
			}
			observed[sig.Key] = sig
		}
		rows, e := tx.Query(`SELECT payload FROM docker_incidents WHERE endpoint = ? AND project = ? AND state IN ('pending','open')`, endpoint, project)
		if e != nil {
			return e
		}
		active := map[string]DockerIncident{}
		for rows.Next() {
			v, e := scanDockerJSON[DockerIncident](rows)
			if e != nil {
				_ = rows.Close()
				return e
			}
			active[v.Signal] = v
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		for key, inc := range active {
			if _, ok := observed[key]; !ok {
				observed[key] = DockerSignal{Key: key, Title: inc.Title, State: "unknown", Detail: "No current observation for this signal."}
			}
		}
		gap := !last.IsZero() && at.Sub(last) > time.Duration(2*m.IntervalSeconds)*time.Second
		for key, sig := range observed {
			inc, exists := active[key]
			if !exists && sig.State != "bad" {
				continue
			}
			if !exists {
				inc = DockerIncident{ID: newID("docker", "incident"), Endpoint: endpoint, Project: project, Signal: key, State: "pending", CreatedAt: m.SampledAt}
			}
			if inc.Revision != m.Revision || gap {
				inc.BadStreak, inc.GoodStreak = 0, 0
			}
			inc.Revision, inc.Title, inc.Detail, inc.UpdatedAt = m.Revision, sig.Title, sig.Detail, m.SampledAt
			switch sig.State {
			case "bad":
				inc.BadStreak++
				inc.GoodStreak = 0
			case "good":
				inc.GoodStreak++
				inc.BadStreak = 0
			default:
				inc.GoodStreak, inc.BadStreak = 0, 0
			}
			if inc.State == "pending" && inc.BadStreak >= m.BadSamples {
				inc.State, inc.OpenedAt = "open", m.SampledAt
			}
			if inc.GoodStreak >= 2 {
				inc.State, inc.ResolvedAt = "resolved", m.SampledAt
			}
			raw, _ := json.Marshal(inc)
			if _, err = tx.Exec(`INSERT INTO docker_incidents (id,endpoint,project,signal,state,updated_at,payload) VALUES (?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET state=excluded.state,updated_at=excluded.updated_at,payload=excluded.payload`, inc.ID, endpoint, project, key, inc.State, inc.UpdatedAt, string(raw)); err != nil {
				return err
			}
		}
		m.EvaluatedAt = m.SampledAt
	}
	cutoff := at.Add(-time.Duration(m.RetentionDays) * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	if _, err = tx.Exec(`DELETE FROM docker_incidents WHERE endpoint = ? AND project = ? AND state != 'open' AND updated_at < ?`, endpoint, project, cutoff); err != nil {
		return err
	}
	raw, _ := json.Marshal(m)
	if _, err = tx.Exec(`UPDATE docker_monitors SET payload = ? WHERE endpoint = ? AND project = ?`, string(raw), endpoint, project); err != nil {
		return err
	}
	if err = s.AppendEventTx(tx, "docker.health", nil, nil, map[string]string{"endpoint": endpoint, "project": project, "sampledAt": m.SampledAt}); err != nil {
		return err
	}
	return s.commit(tx)
}
