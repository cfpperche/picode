package clisession

import (
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgySource lists Antigravity conversations from
// ~/.gemini/antigravity-cli/conversation_summaries.db, the index Google's CLI
// keeps over its per-conversation stores under
// ~/.gemini/antigravity-cli/conversations/<id>.db. Read-only. Resume flag
// verified against a real install (Antigravity 1.2.2, `agy --help`
// 2026-09-14): `--conversation <id>` resumes one, `--continue` the newest.
//
// The CLI's home is ~/.gemini on every platform it ships — it honours no
// XDG_DATA_HOME, so neither does this path.
//
// A row is a conversation when it has steps, a workspace and a time. The
// folder comes from `workspace_uris`, a JSON array of file:// URIs (the July
// conversations predate it and carry none); nested conversations
// (parent_conversation_id, a subagent run) stay out of the picker, the way
// OpenCode's child sessions do.
type AgySource struct{}

func (AgySource) CLI() string { return "agy" }

// AgyTestDB, when set, is AgyDBPath() (tests only).
var AgyTestDB string

// AgyDBPath is Antigravity's conversation index. Exported so a test opens the
// same file this package lists from.
func AgyDBPath() string { return agyDBPath() }

func agyDBPath() string {
	if AgyTestDB != "" {
		return AgyTestDB
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gemini", "antigravity-cli", "conversation_summaries.db")
}

// agyConversationsDir holds one SQLite file per conversation; the picker
// shows its size, and it is also where a future reader looks for transcripts
// (brain/<id>/.system_generated/logs/chunks/transcript/*.jsonl).
func agyConversationsDir() string {
	if AgyTestDB != "" {
		return filepath.Join(filepath.Dir(AgyTestDB), "conversations")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gemini", "antigravity-cli", "conversations")
}

func (AgySource) List(cwd string) ([]Summary, error) {
	path := agyDBPath()
	if path == "" {
		return nil, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}
	out, err := listAgyDB(path, cwd)
	if err != nil {
		return nil, nil // locked or a future schema: empty, never fail the listing
	}
	sortNewest(out)
	return out, nil
}

func listAgyDB(path, cwd string) ([]Summary, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(1000)")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	cols, err := sqliteTableColumns(db, "conversation_summaries")
	if err != nil || !cols["conversation_id"] {
		return nil, err
	}
	var sel []string
	for _, name := range []string{"conversation_id", "parent_conversation_id", "title", "preview", "step_count", "last_modified_time", "workspace_uris"} {
		if cols[name] {
			sel = append(sel, name)
		}
	}
	rows, err := db.Query("SELECT " + strings.Join(sel, ", ") + " FROM conversation_summaries")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dir := agyConversationsDir()
	var out []Summary
	for rows.Next() {
		s, ok := scanAgyConversation(rows, sel, dir)
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

func scanAgyConversation(rows *sql.Rows, sel []string, dir string) (Summary, bool) {
	dest := make([]any, len(sel))
	var id, parent, title, preview, modified, workspaces sql.NullString
	var steps sql.NullInt64
	for i, name := range sel {
		switch name {
		case "conversation_id":
			dest[i] = &id
		case "parent_conversation_id":
			dest[i] = &parent
		case "title":
			dest[i] = &title
		case "preview":
			dest[i] = &preview
		case "step_count":
			dest[i] = &steps
		case "last_modified_time":
			dest[i] = &modified
		case "workspace_uris":
			dest[i] = &workspaces
		default:
			var skip any
			dest[i] = &skip
		}
	}
	if rows.Scan(dest...) != nil {
		return Summary{}, false
	}
	cid := strings.TrimSpace(id.String)
	if !id.Valid || cid == "" || parent.Valid && strings.TrimSpace(parent.String) != "" || steps.Int64 <= 0 {
		return Summary{}, false
	}
	folder := agyWorkspace(workspaces.String)
	if folder == "" {
		return Summary{}, false
	}
	at := agyTime(modified.String)
	if at == "" {
		return Summary{}, false
	}
	name := strings.TrimSpace(title.String)
	line := strings.TrimSpace(preview.String)
	if name == "" {
		name = line
	}
	if line == "" {
		line = name
	}
	db := filepath.Join(dir, cid+".db")
	s := Summary{
		CLI:        "agy",
		ID:         cid,
		Path:       db,
		ResumeArgs: []string{"--conversation", cid},
		Name:       name,
		Cwd:        folder,
		CreatedAt:  at,
		UpdatedAt:  at,
		Preview:    clip(line, 120),
		Messages:   int(steps.Int64),
		Size:       fileSize(db),
	}
	return s, true
}

// agyWorkspace reads the first folder out of `workspace_uris`, a JSON array
// of file:// URIs (`["file:///home/goat/picode"]`). A plain path is accepted
// too, so a schema that drops the scheme still lists.
func agyWorkspace(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var uris []string
	if json.Unmarshal([]byte(raw), &uris) != nil {
		uris = []string{raw}
	}
	for _, u := range uris {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if !strings.HasPrefix(u, "file://") {
			return filepath.Clean(u)
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Path == "" {
			continue
		}
		return filepath.Clean(parsed.Path)
	}
	return ""
}

// agyTime reads the index's timestamps. The column is declared `datetime`
// and holds `2026-09-15 13:49:14.466367991+00:00`, but the SQLite driver
// hands a declared datetime back as RFC3339Nano
// (`2026-09-15T13:49:14.466367991Z`), so both spellings are accepted. The
// zero value Antigravity writes for an unstarted conversation is rejected.
func agyTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00"} {
		at, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}
		if at.Year() < 1970 {
			return ""
		}
		return at.UTC().Format(time.RFC3339)
	}
	return ""
}
