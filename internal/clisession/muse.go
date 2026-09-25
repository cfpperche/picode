package clisession

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MuseSource lists Muse Code sessions from
// ~/.local/share/muse/session-index.db (or $XDG_DATA_HOME/muse/session-index.db),
// the index Meta's launcher keeps over ~/.local/share/muse/sessions.
// Read-only. Resume form re-measured on Muse Code 1.3.0 (2026-09-24): the
// `resume <session-uuid>` subcommand opens that session; the 1.2.1-era
// `--resume <uuid>` flag is refused ("unexpected argument '--resume'").
//
// The index row is the whole listing — title, first prompt, workspace root,
// model, prompt count and timestamps are columns — so this source parses no
// session log: it stats `session_log_path` for the size. Rows with no
// workspace or no prompt are the launcher's own empty sessions
// (`status = missing_metadata`) and stay out of the picker, the way OpenCode
// drops message-less rows.
type MuseSource struct{}

func (MuseSource) CLI() string { return "muse" }

// MuseTestDB, when set, is MuseDBPath() (tests only).
var MuseTestDB string

// MuseDBPath is Muse Code's session index. Exported so a test (or a future
// metric) opens the same file this package lists from.
func MuseDBPath() string { return museDBPath() }

func museDBPath() string {
	if MuseTestDB != "" {
		return MuseTestDB
	}
	if x := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); x != "" {
		return filepath.Join(x, "muse", "session-index.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "muse", "session-index.db")
}

func (MuseSource) List(cwd string) ([]Summary, error) {
	path := museDBPath()
	if path == "" {
		return nil, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}
	out, err := listMuseDB(path, cwd)
	if err != nil {
		return nil, nil // locked or a future schema: empty, never fail the listing
	}
	sortNewest(out)
	return out, nil
}

func listMuseDB(path, cwd string) ([]Summary, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(1000)")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	cols, err := sqliteTableColumns(db, "sessions")
	if err != nil || !cols["session_id"] {
		return nil, err
	}
	var sel []string
	for _, name := range []string{"session_id", "session_log_path", "workspace_root", "title", "first_user_prompt", "prompt_count", "created_at_us", "updated_at_us", "model_id"} {
		if cols[name] {
			sel = append(sel, name)
		}
	}
	rows, err := db.Query("SELECT " + strings.Join(sel, ", ") + " FROM sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Summary
	for rows.Next() {
		s, ok := scanMuseSession(rows, sel)
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

func scanMuseSession(rows *sql.Rows, sel []string) (Summary, bool) {
	dest := make([]any, len(sel))
	var id, logPath, workspace, title, firstPrompt, model sql.NullString
	var prompts sql.NullInt64
	var created, updated sql.NullInt64
	for i, name := range sel {
		switch name {
		case "session_id":
			dest[i] = &id
		case "session_log_path":
			dest[i] = &logPath
		case "workspace_root":
			dest[i] = &workspace
		case "title":
			dest[i] = &title
		case "first_user_prompt":
			dest[i] = &firstPrompt
		case "prompt_count":
			dest[i] = &prompts
		case "created_at_us":
			dest[i] = &created
		case "updated_at_us":
			dest[i] = &updated
		case "model_id":
			dest[i] = &model
		default:
			var skip any
			dest[i] = &skip
		}
	}
	if rows.Scan(dest...) != nil {
		return Summary{}, false
	}
	sid := strings.TrimSpace(id.String)
	folder := strings.TrimSpace(workspace.String)
	messages := int(prompts.Int64)
	// An empty workspace or no prompt at all is a session the launcher
	// created and nobody used.
	if !id.Valid || sid == "" || folder == "" || messages <= 0 {
		return Summary{}, false
	}
	started := microsEpoch(created)
	ended := microsEpoch(updated)
	if ended == "" {
		ended = started
	}
	log := strings.TrimSpace(logPath.String)
	if ended == "" {
		ended = mtime(log)
	}
	if started == "" {
		started = ended
	}
	if ended == "" {
		return Summary{}, false
	}
	name := strings.TrimSpace(title.String)
	preview := musePrompt(firstPrompt.String)
	if preview == "" {
		preview = name
	}
	s := Summary{
		CLI:        "muse",
		ID:         sid,
		Path:       log,
		ResumeArgs: []string{"resume", sid},
		Name:       name,
		Cwd:        folder,
		CreatedAt:  started,
		UpdatedAt:  ended,
		Preview:    clip(preview, 120),
		Messages:   messages,
		Size:       fileSize(log),
		Model:      strings.TrimSpace(model.String),
	}
	return s, true
}

// musePrompt is the index's first_user_prompt, which Meta writes as the
// literal "None" when a session has no recorded prompt.
func musePrompt(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "none") || raw == "null" {
		return ""
	}
	return raw
}

// microsEpoch reads Muse's microsecond timestamps.
func microsEpoch(v sql.NullInt64) string {
	if !v.Valid || v.Int64 <= 0 {
		return ""
	}
	return time.UnixMicro(v.Int64).UTC().Format(time.RFC3339)
}
