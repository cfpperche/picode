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

// claudeHookEvents are mapped by picode-hook auto (stdin / extra argv JSON).
var claudeHookEvents = []string{
	"UserPromptSubmit", "SessionStart",
	"Stop", "TaskCompleted", "SessionEnd", "SubagentStop",
	"Notification",
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

const hookMapPy = `import json, sys
raw = sys.stdin.read()
if not raw.strip() and len(sys.argv) > 1:
    raw = sys.argv[1]
try:
    d = json.loads(raw) if raw.strip() else {}
except Exception:
    sys.exit(0)
ev = str(d.get("hook_event_name") or d.get("event") or "")
nt = str(d.get("notification_type") or "")
typ = str(d.get("type") or "")
if typ == "agent-turn-complete":
    print("idle")
    sys.exit(0)
working = {"UserPromptSubmit", "SessionStart", "user_prompt_submit", "session_start"}
idle = {"Stop", "TaskCompleted", "SessionEnd", "SubagentStop", "stop", "session_end", "StopCancelled"}
if ev in working:
    print("working")
elif ev in idle:
    print("idle")
elif ev in ("Notification", "notification"):
    print("idle" if nt in ("agent_completed", "idle_prompt") else "needs-you")
`

const hookScriptTmpl = `#!/bin/sh
# PiCode guest-CLI status hook (ADR-0056).
# Usage: picode-hook <working|needs-you|idle|auto> <cli> [json]
# auto maps Claude/Grok/Codex JSON (stdin or $3) to a state word.
[ -n "$PICODE_TERM_ID" ] || exit 0
state=$1
cli=$2
if [ "$state" = auto ]; then
  MAP="%s/picode-hook-map.py"
  if [ -n "$3" ]; then
    state=$(printf "%%s\n" "$3" | python3 "$MAP" 2>/dev/null)
  else
    state=$(python3 "$MAP" 2>/dev/null)
  fi
  [ -n "$state" ] || exit 0
fi
TOKEN=$(cat "%s/token" 2>/dev/null)
case "$PICODE_TERM_URL" in
  https://127.0.0.1:*) url="https://localhost:${PICODE_TERM_URL##*:}" ;;
  *) url=$PICODE_TERM_URL ;;
esac
curl -fsSk -o /dev/null --max-time 3 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"state\":\"$state\",\"cli\":\"$cli\"}" \
  "$url/api/terminals/$PICODE_TERM_ID/state" 2>/dev/null || true
`

func ensureHookScript(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", errors.New("data directory unknown")
	}
	if err := os.WriteFile(filepath.Join(dataDir, "picode-hook-map.py"), []byte(hookMapPy), 0o644); err != nil {
		return "", err
	}
	path := hookScriptPath(dataDir)
	body := fmt.Sprintf(hookScriptTmpl, dataDir, dataDir)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
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
	for _, event := range claudeHookEvents {
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
