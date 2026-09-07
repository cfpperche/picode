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

	"github.com/cfpperche/picode/internal/tmux"
)

// User-initiated prompt door for Agent CLI terminals (ADR-0089). Stage a
// file under the terminal cwd, then paste the caption plus @path into the
// TUI. Inspector type/run still must not use this door.

const (
	maxDropBytes   = 4 * 1024 * 1024
	maxDropFiles   = 4
	dropRelDir     = ".picode/drop"
	dropIgnoreLine = ".picode/drop/"
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
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "Need tmux to send to a terminal.")
			return
		}
		session := tmux.ShellSessionName(t.ID)
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		has, err := deps.Tmux.HasSession(ctx, session)
		if err != nil {
			writeErr(w, http.StatusConflict, "Open the terminal first, then try again.")
			return
		}
		if !has {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Start the terminal first.", "reason": "closed"})
			return
		}
		if !tryLockPrompt(t.ID) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Already sending to this terminal.", "reason": "busy"})
			return
		}
		defer unlockPrompt(t.ID)
		payload := buildPromptPaste(req.Message, rels)
		if err := deps.Tmux.PasteText(ctx, session, payload); err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"})
			return
		}
		if deps.Feed != nil {
			deps.Feed.Ephemeral("terminal.prompt", map[string]any{"termId": t.ID, "typed": true})
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "typed": true})
	}
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
	touchDropGitignore(cwdAbs)
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

func touchDropGitignore(cwd string) {
	p := filepath.Join(cwd, ".gitignore")
	raw, err := os.ReadFile(p)
	if err != nil {
		return
	}
	text := string(raw)
	if strings.Contains(text, dropIgnoreLine) {
		return
	}
	if !strings.HasSuffix(text, "\n") && text != "" {
		text += "\n"
	}
	_ = os.WriteFile(p, []byte(text+dropIgnoreLine+"\n"), 0o644)
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
