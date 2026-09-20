package clisession

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OpenCodeSource lists OpenCode sessions from
// ~/.local/share/opencode/opencode.db (or $XDG_DATA_HOME/opencode/opencode.db).
// Read-only. Resume flag verified against a real install (OpenCode 1.18.29,
// `opencode --help` 2026-09-06): `--session <id>` / `-s`. The CLI's own
// `session list` is project-scoped and is not used.
//
// Child sessions (parent_id), archived rows, empty directories and sessions
// with no messages stay out of the coding-CLI picker. Preview is the
// session title. Size stays 0: the SQLite file is shared across every row.
// Cost is stored but omitted (CLI listing policy).
type OpenCodeSource struct{}

func (OpenCodeSource) CLI() string { return "opencode" }

// OpenCodeTestDB, when set, is OpenCodeDBPath() (tests only).
var OpenCodeTestDB string

// OpenCodeDBPath is OpenCode's SQLite store. Exported so climetrics opens
// the same file this package lists from.
func OpenCodeDBPath() string { return opencodeDBPath() }

func opencodeDBPath() string {
	if OpenCodeTestDB != "" {
		return OpenCodeTestDB
	}
	if x := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); x != "" {
		return filepath.Join(x, "opencode", "opencode.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

func (OpenCodeSource) List(cwd string) ([]Summary, error) {
	path := opencodeDBPath()
	if path == "" {
		return nil, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}
	out, err := listOpenCodeDB(path, cwd)
	if err != nil {
		return nil, nil // locked or malformed: empty, never fail the listing
	}
	sortNewest(out)
	return out, nil
}

func listOpenCodeDB(path, cwd string) ([]Summary, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(1000)")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	cols, err := sqliteTableColumns(db, "session")
	if err != nil || !cols["id"] {
		return nil, err
	}

	sel := []string{"id"}
	for _, name := range []string{"parent_id", "directory", "title", "model", "time_created", "time_updated", "time_archived"} {
		if cols[name] {
			sel = append(sel, name)
		}
	}
	counts, err := opencodeMessageCounts(db)
	if err != nil {
		counts = map[string]int{}
	}

	q := "SELECT " + strings.Join(sel, ", ") + " FROM session"
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Summary
	for rows.Next() {
		s, ok := scanOpenCodeSession(rows, sel, path, counts)
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

// SQLiteTableColumns probes a table's columns so a reader can select only
// what a given vendor's schema version actually has. Exported for
// climetrics, which reads more columns than the picker does and must
// degrade rather than fail when one is missing.
func SQLiteTableColumns(db *sql.DB, table string) (map[string]bool, error) {
	return sqliteTableColumns(db, table)
}

func sqliteTableColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
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

func opencodeMessageCounts(db *sql.DB) (map[string]int, error) {
	cols, err := sqliteTableColumns(db, "message")
	if err != nil || !cols["session_id"] {
		return map[string]int{}, err
	}
	rows, err := db.Query("SELECT session_id, COUNT(*) FROM message GROUP BY session_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, nil
}

func scanOpenCodeSession(rows *sql.Rows, sel []string, path string, counts map[string]int) (Summary, bool) {
	dest := make([]any, len(sel))
	var id, parent, directory, title, model sql.NullString
	var created, updated, archived sql.NullFloat64
	for i, name := range sel {
		switch name {
		case "id":
			dest[i] = &id
		case "parent_id":
			dest[i] = &parent
		case "directory":
			dest[i] = &directory
		case "title":
			dest[i] = &title
		case "model":
			dest[i] = &model
		case "time_created":
			dest[i] = &created
		case "time_updated":
			dest[i] = &updated
		case "time_archived":
			dest[i] = &archived
		default:
			var skip any
			dest[i] = &skip
		}
	}
	if rows.Scan(dest...) != nil || !id.Valid || strings.TrimSpace(id.String) == "" {
		return Summary{}, false
	}
	if parent.Valid && strings.TrimSpace(parent.String) != "" {
		return Summary{}, false
	}
	if archived.Valid && archived.Float64 != 0 {
		return Summary{}, false
	}
	folder := strings.TrimSpace(directory.String)
	if folder == "" {
		return Summary{}, false
	}
	sid := strings.TrimSpace(id.String)
	messages := counts[sid]
	if messages <= 0 {
		return Summary{}, false
	}
	started := unixEpoch(created)
	ended := unixEpoch(updated)
	if ended == "" {
		ended = started
	}
	if ended == "" {
		return Summary{}, false
	}
	name := strings.TrimSpace(title.String)
	s := Summary{
		CLI:        "opencode",
		ID:         sid,
		Path:       path,
		ResumeArgs: []string{"--session", sid},
		Name:       name,
		Cwd:        folder,
		CreatedAt:  started,
		UpdatedAt:  ended,
		Preview:    clip(name, 120),
		Model:      opencodeModel(model.String),
		Messages:   messages,
	}
	return s, true
}

func unixEpoch(v sql.NullFloat64) string {
	if !v.Valid || v.Float64 <= 0 {
		return ""
	}
	n := v.Float64
	if n >= 1e12 {
		return time.UnixMilli(int64(n)).UTC().Format(time.RFC3339)
	}
	sec := int64(n)
	nsec := int64((n - float64(sec)) * 1e9)
	if nsec < 0 {
		nsec = 0
	}
	return time.Unix(sec, nsec).UTC().Format(time.RFC3339)
}

func opencodeModel(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var m struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
	}
	if json.Unmarshal([]byte(raw), &m) != nil {
		return clip(raw, 80)
	}
	id := strings.TrimSpace(m.ID)
	prov := strings.TrimSpace(m.ProviderID)
	if prov != "" && id != "" {
		return prov + "/" + id
	}
	return id
}
