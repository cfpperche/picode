package server

// Guest CLI intercept HTTP (ADR-0056). Enable/disable writes wrappers
// under <dataDir>/bin — never the user's ~/.claude, ~/.codex, ~/.grok.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const wiringMarker = "picode-hook"

// claudeHookEvents are the three lifecycle points: a prompt starts a
// turn, Stop ends it, Notification asks for the human.
var claudeHookEvents = map[string]string{
	"UserPromptSubmit": "working",
	"Stop":             "idle",
	"Notification":     "needs-you",
}

type wiringRow struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Bin       string `json:"bin"`
	Installed bool   `json:"installed"`
	Wired     bool   `json:"wired"`
	Note      string `json:"note,omitempty"`
}

func homeFile(path ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{home}, path...)...), nil
}

func claudeSettingsPath() (string, error) { return homeFile(".claude", "settings.json") }

func hookScriptPath(dataDir string) string { return filepath.Join(dataDir, wiringMarker) }

const hookScriptTmpl = `#!/bin/sh
# PiCode guest-CLI status hook (ADR-0056). Installed in the data dir.
# Usage: picode-hook <working|needs-you|idle> <cli>.
# Outside a PiCode terminal ($PICODE_TERM_ID empty) this is a no-op.
[ -n "$PICODE_TERM_ID" ] || exit 0
TOKEN=$(cat "%s/token" 2>/dev/null)
curl -fsS -o /dev/null --max-time 3 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"state\":\"$1\",\"cli\":\"$2\"}" \
  "$PICODE_TERM_URL/api/terminals/$PICODE_TERM_ID/state" 2>/dev/null || true
`

func ensureHookScript(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", errors.New("data directory unknown")
	}
	path := hookScriptPath(dataDir)
	if err := os.WriteFile(path, []byte(fmt.Sprintf(hookScriptTmpl, dataDir)), 0o755); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func groupHasMarker(group any) bool {
	m, ok := group.(map[string]any)
	if !ok {
		return false
	}
	hooks, ok := m["hooks"].([]any)
	if !ok {
		return false
	}
	for _, h := range hooks {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		if cmd, _ := hm["command"].(string); strings.Contains(cmd, wiringMarker) {
			return true
		}
	}
	return false
}

// claudeSetWiring only strips legacy marker entries from a settings
// JSON (enable=false). Enable of intercept must never call this with
// true — that was the user-home pollution we retired.
func claudeSetWiring(settingsPath, scriptPath string, enable bool) (bool, error) {
	if enable {
		return false, errors.New("refusing to write user Claude settings")
	}
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		return false, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return false, nil // don't clobber a file we no longer own
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		return false, nil
	}
	changed := false
	for event := range claudeHookEvents {
		groups, _ := hooks[event].([]any)
		kept := make([]any, 0, len(groups))
		for _, g := range groups {
			if groupHasMarker(g) {
				changed = true
				continue
			}
			kept = append(kept, g)
		}
		if len(kept) > 0 {
			hooks[event] = kept
		} else {
			delete(hooks, event)
		}
	}
	if !changed {
		return false, nil
	}
	if len(hooks) > 0 {
		doc["hooks"] = hooks
	} else {
		delete(doc, "hooks")
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(settingsPath, append(out, '\n'), 0o600)
}

func installedOnPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func wiringRows(dataDir string) []wiringRow {
	return []wiringRow{
		{
			ID: "claude-code", Label: "Claude Code", Bin: "claude",
			Installed: installedOnPath("claude"),
			Wired:     interceptWired(dataDir, "claude-code", "claude"),
			Note:      "Injects --settings in PiCode terminals only.",
		},
		{
			ID: "codex", Label: "Codex", Bin: "codex",
			Installed: installedOnPath("codex"),
			Wired:     interceptWired(dataDir, "codex", "codex"),
			Note:      "Injects notify via -c. End-of-turn only.",
		},
		{
			ID: "grok", Label: "Grok", Bin: "grok",
			Installed: installedOnPath("grok"),
			Wired:     interceptWired(dataDir, "grok", "grok"),
			Note:      "GROK_HOME overlay in PiCode's data dir. Auth stays yours.",
		},
	}
}

func handleWiringStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"clis": wiringRows(deps.DataDir)})
	}
}

func handleWiringEnable(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli := r.PathValue("cli")
		if err := installIntercept(deps.DataDir, cli); err != nil {
			if strings.Contains(err.Error(), "unknown CLI") {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"clis": wiringRows(deps.DataDir)})
	}
}

func handleWiringDisable(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli := r.PathValue("cli")
		if err := uninstallIntercept(deps.DataDir, cli); err != nil {
			if strings.Contains(err.Error(), "unknown CLI") {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"clis": wiringRows(deps.DataDir)})
	}
}

func registerTerminalWiringRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/terminals/wiring", handleWiringStatus(deps))
	mux.HandleFunc("POST /api/terminals/wiring/{cli}/enable", handleWiringEnable(deps))
	mux.HandleFunc("POST /api/terminals/wiring/{cli}/disable", handleWiringDisable(deps))
}
