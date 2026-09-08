package clisession

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfpperche/picode/internal/transcript"
)

// Hermes writer (ADR-0094): Hermes ships `hermes sessions import [--from
// claude|codex] <path>`, documented as "pull a conversation started in
// Claude Code or Codex CLI into the Hermes session store so it can be
// resumed with 'hermes --resume <id>'. The foreign files are only read,
// never modified." So a handoff into Hermes writes a Claude Code
// transcript to a temporary file outside every CLI's store and lets Hermes
// pull it in — PiCode never touches `state.db`, which the running agent
// holds open.
//
// Verified against Hermes Agent 0.21.0 (2026-09-07): the import prints the
// id it assigned, stamps `source = claude-code` on the row, and flattens
// tool calls into the assistant's text. Hermes owns its ids, so the
// session id cannot be pre-assigned and `WriteRequest.SessionID` is
// ignored.
func (HermesSource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	if req.Run == nil {
		return Summary{}, ErrNoRunner
	}
	// The file handed over is a Claude Code transcript, so the format that
	// must be known is Claude's, not the version of Hermes reading it.
	version, model := claudeNewestFacts(ClaudeProjectsRoot())
	if version == "" {
		return Summary{}, ErrUnknownFormat
	}
	t = t.Prepare()
	// Hermes' importer folds tool calls into the assistant's text either
	// way; asking for them as turns keeps what the next agent reads closer
	// to what the source actually did.
	body, emitted, err := claudeTranscript(t, transcript.NewID(), filepath.Clean(req.Cwd), version, model, true, req.resolvedNow())
	if err != nil {
		return Summary{}, err
	}
	file, err := os.CreateTemp("", "picode-handoff-*.jsonl")
	if err != nil {
		return Summary{}, err
	}
	tmp := file.Name()
	defer os.Remove(tmp)
	if _, err := file.Write(body); err != nil {
		file.Close()
		return Summary{}, err
	}
	if err := file.Close(); err != nil {
		return Summary{}, err
	}
	out, err := req.Run(ctx, "sessions", "import", "--from", "claude", tmp)
	if err != nil {
		return Summary{}, fmt.Errorf("hermes sessions import failed: %w: %s", err, clip(string(out), 200))
	}
	sid := hermesImportedID(string(out))
	if sid == "" {
		return Summary{}, fmt.Errorf("hermes did not report the session it imported: %s", clip(string(out), 200))
	}
	cwd := filepath.Clean(req.Cwd)
	ref := Ref{ID: sid, Cwd: cwd}
	// Hermes rewrites what it imports — tool calls become text in the
	// assistant's message — so the check is that the conversation arrived
	// and is addressable, not that it reads back block for block.
	back, err := (HermesSource{}).Read(ctx, ref)
	if err != nil {
		return Summary{}, fmt.Errorf("re-reading the imported session failed: %w", err)
	}
	want := transcript.Timeline{Events: emitted}.Counts().Messages
	if got := back.Counts().Messages; got == 0 || got < want/2 {
		return Summary{}, fmt.Errorf("the imported session holds %d of %d messages", got, want)
	}
	name := title(t)
	if name == "" {
		name = strings.TrimSpace(back.Header.Title)
	}
	return Summary{
		CLI:        "hermes",
		ID:         sid,
		Path:       hermesStatePath(),
		ResumeArgs: []string{"--resume", sid},
		Name:       clip(name, 120),
		Cwd:        cwd,
		CreatedAt:  rfc3339(req.resolvedNow()),
		UpdatedAt:  rfc3339(req.resolvedNow()),
		Preview:    preview(t),
		Messages:   back.Counts().Messages,
		Model:      hermesCurrentModel(),
	}, nil
}

// hermesImportedID reads the id out of the import's own report
// ("Imported Claude Code session as 20260907_145708_de908b"), which is the
// only place it appears: Hermes assigns it.
var hermesIDPattern = regexp.MustCompile(`\b(\d{8}_\d{6}_[0-9a-f]{6,8})\b`)

func hermesImportedID(out string) string {
	return hermesIDPattern.FindString(out)
}

// hermesCurrentModel is the model of the newest session in the local
// store — what this installation is actually serving.
func hermesCurrentModel() string {
	path := hermesStatePath()
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(1000)")
	if err != nil {
		return ""
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	cols, err := hermesSessionColumns(db)
	if err != nil || !cols["model"] {
		return ""
	}
	var v sql.NullString
	if db.QueryRow(`SELECT model FROM sessions WHERE model IS NOT NULL AND model <> '' ORDER BY started_at DESC LIMIT 1`).Scan(&v) != nil {
		return ""
	}
	return strings.TrimSpace(v.String)
}
