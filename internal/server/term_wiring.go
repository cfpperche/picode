package server

// Guest CLI wiring — the one-click half of ADR-0056 tier 1. The guide
// (www/guide/terminal-status.md) documents the manual path; this turns it
// into two buttons for the one CLI with a machine-readable config:
//
//	Claude Code → ~/.claude/settings.json (JSON, stdlib-mergeable).
//	Codex       → config.toml, no stdlib TOML — manual, shown read-only.
//
// PiCode installs a tiny reporter at <dataDir>/picode-hook and merges
// three hook entries into Claude's settings. A disable removes exactly
// the entries carrying the "picode-hook" marker and nothing else — user
// content survives byte-for-byte.

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

// claudeHookEvents are the three lifecycle points tier 1 needs:
// a prompt starts a turn, Stop ends it, Notification asks for the human.
var claudeHookEvents = map[string]string{ // event → reporter state
	"UserPromptSubmit": "working",
	"Stop":             "idle",
	"Notification":     "needs-you",
}

type wiringRow struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Installed  bool   `json:"installed"`
	Wired      bool   `json:"wired"`
	ConfigPath string `json:"configPath,omitempty"`
	Manual     bool   `json:"manual,omitempty"`
	Note       string `json:"note,omitempty"`
}

func homeFile(path ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{home}, path...)...), nil
}

func claudeSettingsPath() (string, error) { return homeFile(".claude", "settings.json") }
func codexConfigPath() (string, error)    { return homeFile(".codex", "config.toml") }

// hookScriptPath lives in the data dir, not ~/.local/bin: no PATH
// assumptions, and the token it reads sits in the same directory.
func hookScriptPath(dataDir string) string { return filepath.Join(dataDir, wiringMarker) }

const hookScriptTmpl = `#!/bin/sh
# PiCode guest-CLI status hook (ADR-0056 tier 1) — installed by PiCode.
# Usage: picode-hook <working|needs-you|idle> <cli>. Outside a PiCode
# terminal ($PICODE_TERM_ID empty) it is a no-op, so global installs are
# safe. A failed report is dropped: status is a courtesy, never an alert.
[ -n "$PICODE_TERM_ID" ] || exit 0
TOKEN=$(cat "%s/token" 2>/dev/null)
curl -fsS -o /dev/null --max-time 3 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"state\":\"$1\",\"cli\":\"$2\"}" \
  "$PICODE_TERM_URL/api/terminals/$PICODE_TERM_ID/state" 2>/dev/null || true
`

// ensureHookScript (re)writes the reporter. Refreshing on every enable
// keeps the embedded token path and the report format current.
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

// groupHasMarker reports whether one hooks[] group carries a command
// referencing the PiCode reporter.
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

// claudeWired reports whether any hook entry references the reporter.
func claudeWired(settingsPath string) bool {
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		return false
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return false
	}
	hooks, _ := doc["hooks"].(map[string]any)
	for _, groups := range hooks {
		list, ok := groups.([]any)
		if !ok {
			continue
		}
		for _, g := range list {
			if groupHasMarker(g) {
				return true
			}
		}
	}
	return false
}

// claudeSetWiring merges (enable) or strips (disable) PiCode's hook
// entries, preserving every other byte of meaning in the document.
func claudeSetWiring(settingsPath, scriptPath string, enable bool) (bool, error) {
	var doc map[string]any
	if raw, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return false, fmt.Errorf("%s is not valid JSON — fix or remove it first", settingsPath)
		}
	}
	if doc == nil {
		doc = map[string]any{}
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	changed := false
	for event, state := range claudeHookEvents {
		groups, _ := hooks[event].([]any)
		kept := make([]any, 0, len(groups))
		hadOurs := false
		for _, g := range groups {
			if groupHasMarker(g) {
				hadOurs = true
				continue
			}
			kept = append(kept, g)
		}
		if enable {
			kept = append(kept, map[string]any{
				"hooks": []any{map[string]any{
					"type":    "command",
					"command": fmt.Sprintf("%s %s claude-code", scriptPath, state),
				}},
			})
			hooks[event] = kept
			changed = true
			continue
		}
		// disable: strip ours; drop the event key when nothing is left.
		if hadOurs {
			changed = true
			if len(kept) > 0 {
				hooks[event] = kept
			} else {
				delete(hooks, event)
			}
		}
	}
	if len(hooks) > 0 {
		doc["hooks"] = hooks
	} else {
		delete(doc, "hooks")
	}
	if !changed {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return false, err
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(settingsPath, append(out, '\n'), 0o600)
}

// wiringRows answers GET /api/terminals/wiring: one row per supported
// CLI, honestly marked when the wiring is manual.
func installedOnPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func wiringRows() []wiringRow {
	rows := []wiringRow{}
	if p, err := claudeSettingsPath(); err == nil {
		rows = append(rows, wiringRow{
			ID: "claude-code", Label: "Claude Code",
			Installed:  installedOnPath("claude"),
			Wired:      claudeWired(p),
			ConfigPath: p,
		})
	}
	if p, err := codexConfigPath(); err == nil {
		raw, _ := os.ReadFile(p)
		rows = append(rows, wiringRow{
			ID: "codex", Label: "Codex",
			Installed:  installedOnPath("codex"),
			Wired:      strings.Contains(string(raw), wiringMarker),
			ConfigPath: p,
			Manual:     true,
			Note:       "One line in config.toml — see the guide.",
		})
	}
	return rows
}

func handleWiringStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"clis": wiringRows()})
	}
}

// handleWiringEnable installs the reporter and merges the Claude hooks.
// Unknown or manual CLIs refuse with the visible reason.
func handleWiringEnable(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("cli") != "claude-code" {
			writeErr(w, http.StatusBadRequest, "One-click wiring covers Claude Code; for this CLI, follow the guide.")
			return
		}
		script, err := ensureHookScript(deps.DataDir)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		p, err := claudeSettingsPath()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := claudeSetWiring(p, script, true); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"clis": wiringRows()})
	}
}

func handleWiringDisable(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("cli") != "claude-code" {
			writeErr(w, http.StatusBadRequest, "One-click wiring covers Claude Code; for this CLI, follow the guide.")
			return
		}
		p, err := claudeSettingsPath()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := claudeSetWiring(p, "", false); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"clis": wiringRows()})
	}
}

func registerTerminalWiringRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/terminals/wiring", handleWiringStatus(deps))
	mux.HandleFunc("POST /api/terminals/wiring/{cli}/enable", handleWiringEnable(deps))
	mux.HandleFunc("POST /api/terminals/wiring/{cli}/disable", handleWiringDisable(deps))
}
