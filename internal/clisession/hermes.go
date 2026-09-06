package clisession

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// HermesSource lists Hermes Agent sessions from ~/.hermes/state.db (or
// $HERMES_HOME/state.db). Read-only. Resume flag verified against a real
// install (Hermes Agent v0.18.2, `hermes --help` 2026-09-06):
// `--resume <SESSION>` / `-r`. Profiles under ~/.hermes/profiles/ are not
// scanned — resume without `-p` looks in the active home and would miss
// or mix them.
//
// Only source=cli and source=tui rows with at least one message and a
// folder (cwd or git_repo_root) are listed. Gateway, ACP, cron and
// subagent sessions stay out of the coding-CLI picker. Preview is the
// session title; the messages table is not queried. Size stays 0: the
// SQLite file is shared across every row and is not a per-session size.
type HermesSource struct{}

func (HermesSource) CLI() string { return "hermes" }

func hermesStatePath() string {
	if h := strings.TrimSpace(os.Getenv("HERMES_HOME")); h != "" {
		return filepath.Join(h, "state.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".hermes", "state.db")
}

func (HermesSource) List(cwd string) ([]Summary, error) {
	path := hermesStatePath()
	if path == "" {
		return nil, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}
	out, err := listHermesDB(path, cwd)
	if err != nil {
		return nil, nil // locked or malformed: empty, never fail the listing
	}
	sortNewest(out)
	return out, nil
}

func listHermesDB(path, cwd string) ([]Summary, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(1000)")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	cols, err := hermesSessionColumns(db)
	if err != nil || !cols["id"] || !cols["source"] {
		return nil, err
	}

	sel := []string{"id", "source"}
	for _, name := range []string{"title", "model", "cwd", "git_repo_root", "started_at", "ended_at", "message_count", "archived"} {
		if cols[name] {
			sel = append(sel, name)
		}
	}
	q := "SELECT " + strings.Join(sel, ", ") + " FROM sessions"
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Summary
	for rows.Next() {
		s, ok := scanHermesSession(rows, sel, path)
		if !ok {
			continue
		}
		if cwd != "" && s.Cwd != cwd {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func hermesSessionColumns(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(sessions)")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, nil
}

func scanHermesSession(rows *sql.Rows, sel []string, path string) (Summary, bool) {
	dest := make([]any, len(sel))
	var id, source, title, model, sessionCwd, repo sql.NullString
	var started, ended sql.NullFloat64
	var messages, archived sql.NullInt64
	for i, name := range sel {
		switch name {
		case "id":
			dest[i] = &id
		case "source":
			dest[i] = &source
		case "title":
			dest[i] = &title
		case "model":
			dest[i] = &model
		case "cwd":
			dest[i] = &sessionCwd
		case "git_repo_root":
			dest[i] = &repo
		case "started_at":
			dest[i] = &started
		case "ended_at":
			dest[i] = &ended
		case "message_count":
			dest[i] = &messages
		case "archived":
			dest[i] = &archived
		default:
			var skip any
			dest[i] = &skip
		}
	}
	if rows.Scan(dest...) != nil || !id.Valid || strings.TrimSpace(id.String) == "" {
		return Summary{}, false
	}
	src := strings.TrimSpace(source.String)
	if src != "cli" && src != "tui" {
		return Summary{}, false
	}
	if archived.Valid && archived.Int64 != 0 {
		return Summary{}, false
	}
	if messages.Valid && messages.Int64 <= 0 {
		return Summary{}, false
	}
	folder := strings.TrimSpace(sessionCwd.String)
	if folder == "" {
		folder = strings.TrimSpace(repo.String)
	}
	if folder == "" {
		return Summary{}, false
	}
	created := unixFloat(started)
	updated := unixFloat(ended)
	if updated == "" {
		updated = created
	}
	if updated == "" {
		return Summary{}, false
	}
	name := strings.TrimSpace(title.String)
	s := Summary{
		CLI:        "hermes",
		ID:         strings.TrimSpace(id.String),
		Path:       path,
		ResumeArgs: []string{"--resume", strings.TrimSpace(id.String)},
		Name:       name,
		Cwd:        folder,
		CreatedAt:  created,
		UpdatedAt:  updated,
		Preview:    clip(name, 120),
		Model:      strings.TrimSpace(model.String),
	}
	if messages.Valid {
		s.Messages = int(messages.Int64)
	}
	return s, true
}

func unixFloat(v sql.NullFloat64) string {
	if !v.Valid || v.Float64 <= 0 {
		return ""
	}
	sec := int64(v.Float64)
	nsec := int64((v.Float64 - float64(sec)) * 1e9)
	if nsec < 0 {
		nsec = 0
	}
	return time.Unix(sec, nsec).UTC().Format(time.RFC3339)
}
