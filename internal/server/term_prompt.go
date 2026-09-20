package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// User-initiated prompt door for Agent CLI terminals (ADR-0089). Stage a
// file under the terminal cwd, then paste the caption plus @path into the
// TUI. Inspector type/run still must not use this door.

const (
	maxDropBytes = 4 * 1024 * 1024
	maxDropFiles = 4
	dropRelDir   = ".picode/drop"
	// dropMaxAge bounds how long a staged attachment survives. Nothing ever
	// deletes it otherwise: the CLI reads the path once, at paste time, and
	// keeps no reference of its own. Swept opportunistically on the next
	// drop in the same project — no daemon, no ticker to forget to wire up.
	dropMaxAge = 7 * 24 * time.Hour
)

func registerTermPromptRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/terminals/{id}/drop", handleTerminalDrop(deps))
	mux.HandleFunc("POST /api/terminals/{id}/prompt", handleTerminalPrompt(deps))
}

var promptInFlight = struct {
	sync.Mutex
	ids map[string]bool
}{ids: map[string]bool{}}

func resetPromptInFlight() {
	promptInFlight.Lock()
	promptInFlight.ids = map[string]bool{}
	promptInFlight.Unlock()
}

func tryLockPrompt(id string) bool {
	promptInFlight.Lock()
	defer promptInFlight.Unlock()
	if promptInFlight.ids == nil {
		promptInFlight.ids = map[string]bool{}
	}
	if promptInFlight.ids[id] {
		return false
	}
	promptInFlight.ids[id] = true
	return true
}

func unlockPrompt(id string) {
	promptInFlight.Lock()
	delete(promptInFlight.ids, id)
	promptInFlight.Unlock()
}

// termHoldsCLI is true when this terminal is an Agent CLI (saved launch or
// live wrapper lease). Plain shells wait (ADR-0089).
func termHoldsCLI(deps Deps, id string) bool {
	if deps.Store != nil {
		if v, err := deps.Store.TerminalLaunch(id); err == nil && v != nil && strings.TrimSpace(v.CLI) != "" {
			return true
		}
	}
	if deps.TermRuntimes != nil {
		if rt, ok := deps.TermRuntimes.Get(id); ok && strings.TrimSpace(rt.CLI) != "" {
			return true
		}
	}
	return false
}

func refuseInspectorCLI(w http.ResponseWriter, deps Deps, id string) bool {
	if !termHoldsCLI(deps, id) {
		return false
	}
	writeJSON(w, http.StatusConflict, map[string]any{
		"error":  "This is an Agent CLI. Send from the attach bar.",
		"reason": "cli",
	})
	return true
}

type dropBody struct {
	Name string `json:"name"`
	Mime string `json:"mime"`
	Data string `json:"data"`
}

func handleTerminalDrop(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if routeBoundPi(deps, w, r, handleAgentDrop(deps)) {
			return
		}
		t, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if !termHoldsCLI(deps, t.ID) {
			writeErr(w, http.StatusConflict, "Attach is for Agent CLI terminals.")
			return
		}
		var req dropBody
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDropBytes*2)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		raw, err := decodeDropData(req.Data)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		cwd := liveTermCwd(deps, r, t)
		out, err := writeDropFile(cwd, req.Name, raw)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type promptBody struct {
	Message string   `json:"message"`
	Paths   []string `json:"paths"`
}

func handleTerminalPrompt(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if routeBoundPi(deps, w, r, handleAgentPrompt(deps)) {
			return
		}
		t, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if !termHoldsCLI(deps, t.ID) {
			writeErr(w, http.StatusConflict, "Attach is for Agent CLI terminals.")
			return
		}
		var req promptBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		cwd := liveTermCwd(deps, r, t)
		rels, err := checkPromptPaths(cwd, req.Paths)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.TrimSpace(req.Message) == "" && len(rels) == 0 {
			writeErr(w, http.StatusBadRequest, "message or file is required")
			return
		}
		payload := buildPromptPaste(req.Message, rels)
		status, body := pasteToTerminal(deps, r.Context(), t, payload)
		if status != http.StatusOK {
			writeJSON(w, status, body)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "typed": true})
	}
}

// pasteToTerminal is the one delivery path of the ADR-0089 door: tmux
// present, session alive, one in-flight paste per terminal, bracketed
// paste + Enter, `terminal.prompt` announced. Callers validate their own
// inputs first; this only delivers. A non-200 status comes with the JSON
// body to write (with a reason the UI can name).
func pasteToTerminal(deps Deps, ctx context.Context, t store.Terminal, payload string) (int, map[string]any) {
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return http.StatusServiceUnavailable, map[string]any{"error": "Need tmux to send to a terminal."}
	}
	session := tmux.ShellSessionName(t.ID)
	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	has, err := deps.Tmux.HasSession(cctx, session)
	if err != nil {
		return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again."}
	}
	if !has {
		return http.StatusConflict, map[string]any{"error": "Start the terminal first.", "reason": "closed"}
	}
	if !tryLockPrompt(t.ID) {
		return http.StatusConflict, map[string]any{"error": "Already sending to this terminal.", "reason": "busy"}
	}
	defer unlockPrompt(t.ID)
	if err := deps.Tmux.PasteText(cctx, session, payload); err != nil {
		return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"}
	}
	if deps.Feed != nil {
		deps.Feed.Ephemeral("terminal.prompt", map[string]any{"termId": t.ID, "typed": true})
	}
	return http.StatusOK, nil
}

func decodeDropData(data string) ([]byte, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, fmt.Errorf("file data is required")
	}
	if i := strings.Index(data, ","); i >= 0 && strings.Contains(data[:i], "base64") {
		data = data[i+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(data)
	}
	if err != nil {
		return nil, fmt.Errorf("file data is not valid")
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("file is empty")
	}
	if len(raw) > maxDropBytes {
		return nil, fmt.Errorf("each file must be under 4 MB")
	}
	return raw, nil
}

func writeDropFile(cwd, name string, raw []byte) (map[string]any, error) {
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("can't write in this folder")
	}
	st, err := os.Stat(cwdAbs)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("can't write in this folder")
	}
	dir := filepath.Join(cwdAbs, filepath.FromSlash(dropRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("can't write in this folder")
	}
	touchDropGitignore(filepath.Dir(dir))
	sweepDropDir(dir, dropMaxAge)
	base := dropFileName(name)
	abs := filepath.Join(dir, base)
	if err := os.WriteFile(abs, raw, 0o644); err != nil {
		return nil, fmt.Errorf("can't write in this folder")
	}
	rel := filepath.ToSlash(filepath.Join(dropRelDir, base))
	return map[string]any{"path": rel, "name": base, "bytes": len(raw)}, nil
}

func dropFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "file"
	}
	var b strings.Builder
	for _, r := range base {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	clean := strings.Trim(b.String(), ".-")
	if clean == "" {
		clean = "file"
	}
	if len(clean) > 80 {
		clean = clean[:80]
	}
	var id [4]byte
	_, _ = rand.Read(id[:])
	return hex.EncodeToString(id[:]) + "-" + clean
}

// touchDropGitignore keeps staged attachments out of git without ever
// touching the project's own tracked .gitignore (that file used to gain a
// silent, uncommitted `.picode/drop/` line the moment anyone attached a
// file — a surprise diff, and one that never appeared at all in a project
// with no root .gitignore to append to). picodeDir is the project's
// .picode folder (the parent of drop/); its own nested, never-committed
// .gitignore covers drop/ unconditionally.
func touchDropGitignore(picodeDir string) {
	p := filepath.Join(picodeDir, ".gitignore")
	if _, err := os.Stat(p); err == nil {
		return
	}
	_ = os.WriteFile(p, []byte("drop/\n"), 0o644)
}

// sweepDropDir deletes staged attachments older than maxAge. Nothing else
// ever removes them: the CLI reads a path once, at paste time, and keeps no
// reference of its own, so without this a project accumulates every image
// ever attached. Run opportunistically on the next drop into the same
// project rather than a background ticker — simpler, and guaranteed to run
// exactly when the folder is touched again.
func sweepDropDir(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

func checkPromptPaths(cwd string, paths []string) ([]string, error) {
	if len(paths) > maxDropFiles {
		return nil, fmt.Errorf("at most 4 files")
	}
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("can't read this folder")
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		abs, rel, err := relUnderCwd(cwdAbs, p)
		if err != nil {
			return nil, fmt.Errorf("that file is outside the folder")
		}
		st, err := os.Stat(abs)
		if err != nil || !st.Mode().IsRegular() {
			return nil, fmt.Errorf("that file is not here")
		}
		if st.Size() > maxDropBytes {
			return nil, fmt.Errorf("each file must be under 4 MB")
		}
		out = append(out, rel)
	}
	return out, nil
}

func buildPromptPaste(message string, paths []string) string {
	msg := strings.TrimRight(message, "\n")
	var b strings.Builder
	if msg != "" {
		b.WriteString(msg)
	}
	for _, p := range paths {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		if !strings.HasPrefix(p, "@") {
			b.WriteByte('@')
		}
		b.WriteString(p)
	}
	return b.String()
}
