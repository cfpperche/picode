package server

// Terminal CLI intercept (ADR-0056, owner 2026-09-03): never write the
// user's ~/.claude / ~/.codex / ~/.grok / ~/.pi. PiCode terminals prepend
// <dataDir>/bin to PATH at tmux session creation; wrappers there exec
// the real binary with launch-time injection (args, extension, or overlay).
// Lifecycle-aware wrappers remain a small shell parent so they can announce
// the process's end after the real CLI exits; maintenance bypasses still exec
// directly to preserve the CLI's dispatch semantics.
// Outside those sessions the wrappers are not on PATH.

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func interceptBinDir(dataDir string) string { return filepath.Join(dataDir, "bin") }
func interceptDir(dataDir string) string    { return filepath.Join(dataDir, "intercept") }

func interceptEnabledPath(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "enabled.json")
}

func loadInterceptEnabled(dataDir string) map[string]bool {
	out := map[string]bool{}
	raw, err := os.ReadFile(interceptEnabledPath(dataDir))
	if err != nil {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func saveInterceptEnabled(dataDir string, m map[string]bool) error {
	if err := os.MkdirAll(interceptDir(dataDir), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return writeInterceptFile(interceptEnabledPath(dataDir), append(raw, '\n'), 0o600)
}

func wrapperPath(dataDir, binName string) string {
	return filepath.Join(interceptBinDir(dataDir), binName)
}

func interceptOn(dataDir, cliID string) bool {
	return loadInterceptEnabled(dataDir)[cliID]
}

// interceptSessionPath is the PATH=… entry for new-session -e, or empty
// when nothing is intercepting (so we don't shadow the user's PATH).
func interceptBashrcPath(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "bashrc")
}

const interceptBashrc = `# PiCode terminal rc (not your login rc).
# Sources the usual bashrc, then puts intercept first so a PATH reset
# in ~/.bashrc cannot hide the wrappers.
[ -f /etc/bash.bashrc ] && . /etc/bash.bashrc
[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"
if [ -n "$PICODE_INTERCEPT_BIN" ] && [ -d "$PICODE_INTERCEPT_BIN" ]; then
  PATH="$PICODE_INTERCEPT_BIN:$PATH"
  export PATH
fi
`

func ensureInterceptBashrc(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", fmt.Errorf("data directory unknown")
	}
	return interceptBashrcPath(dataDir), writeExecutable(interceptBashrcPath(dataDir), interceptBashrc)
}

func interceptBinEnv(dataDir string) string {
	bin := interceptBinDir(dataDir)
	ents, err := os.ReadDir(bin)
	if err != nil {
		return ""
	}
	for _, e := range ents {
		if !e.IsDir() {
			return "PICODE_INTERCEPT_BIN=" + bin
		}
	}
	return ""
}

func interceptSessionPath(dataDir string) string {
	bin := interceptBinDir(dataDir)
	ents, err := os.ReadDir(bin)
	if err != nil || len(ents) == 0 {
		return ""
	}
	has := false
	for _, e := range ents {
		if !e.IsDir() {
			has = true
			break
		}
	}
	if !has {
		return ""
	}
	return "PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH")
}

const wrapperFindReal = `here=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
real=
IFS=:
for d in $PATH; do
  [ "$d" = "$here" ] && continue
  if [ -x "$d/$name" ]; then real="$d/$name"; break; fi
done
unset IFS
if [ -z "$real" ]; then
  printf '%s\n' "picode: $name is not installed outside this terminal." >&2
  exit 127
fi
`

const wrapperLifecycleTmpl = `# Runtime presence is separate from hook activity (ADR-0062).
picode_tui=1
case "$name:${1-}:${2-}" in
  codex:exec:*|codex:app-server:*|codex:mcp-server:*) picode_tui=0 ;;
  pi:--mode:rpc|pi:--mode:json) picode_tui=0 ;;
esac
for picode_arg in "$@"; do
  case "$picode_arg" in
    -p|--print|--json|--headless|--non-interactive|--version|-V|--help|-h) picode_tui=0 ;;
  esac
done
picode_run_id="${PICODE_TERM_ID}-$$-$(date +%%s%%N 2>/dev/null || date +%%s)"
picode_hook=%q
if [ "$picode_tui" = 1 ] && [ -n "$PICODE_TERM_ID" ]; then
  export PICODE_TUI_RUN_ID="$picode_run_id"
  "$picode_hook" runtime-start "$name" "$picode_run_id" "$$" >/dev/null 2>&1 || true
fi
`

func wrapperLifecycle(hook string) string {
	return fmt.Sprintf(wrapperLifecycleTmpl, hook)
}

const wrapperLifecycleEnd = `rc=$?
if [ "$picode_tui" = 1 ] && [ -n "${PICODE_TUI_RUN_ID-}" ]; then
  "$picode_hook" runtime-end "$name" "$PICODE_TUI_RUN_ID" "$$" >/dev/null 2>&1 || true
fi
exit $rc
`

func writeExecutable(path, body string) error {
	return writeInterceptFile(path, []byte(body), 0o755)
}

// Replace complete files atomically: running wrappers and hook reporters must
// never read a half-written script while another terminal is being launched.
func writeInterceptFile(path string, body []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".picode-write-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(body); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func claudeSettingsFile(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "claude-settings.json")
}

func writeClaudeIntercept(dataDir, hook string) error {
	doc := map[string]any{"hooks": map[string]any{}}
	hooks := doc["hooks"].(map[string]any)
	for _, event := range claudeHookEvents {
		hooks[event] = []any{map[string]any{
			"hooks": []any{map[string]any{
				"type":    "command",
				"command": hook + " auto claude-code",
			}},
		}}
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(interceptDir(dataDir), 0o755); err != nil {
		return err
	}
	if err := writeInterceptFile(claudeSettingsFile(dataDir), append(raw, '\n'), 0o600); err != nil {
		return err
	}
	body := "#!/bin/sh\n# PiCode intercept — Claude Code. Session PATH only.\nname=claude\n" +
		wrapperFindReal +
		wrapperLifecycle(hook) +
		fmt.Sprintf("\"$real\" --settings %q \"$@\"\n", claudeSettingsFile(dataDir)) +
		wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "claude"), body)
}

type codexHookSpec struct {
	event      string
	key        string
	timeoutSec int
}

var codexHookSpecs = []codexHookSpec{
	{event: "UserPromptSubmit", key: "user_prompt_submit", timeoutSec: 5},
	{event: "PermissionRequest", key: "permission_request", timeoutSec: 5},
	{event: "Stop", key: "stop", timeoutSec: 5},
	// Codex caps Interrupt and SessionEnd hooks at three seconds.
	{event: "Interrupt", key: "interrupt", timeoutSec: 3},
	{event: "SessionEnd", key: "session_end", timeoutSec: 3},
}

func tomlString(s string) string {
	raw, _ := json.Marshal(s)
	return string(raw)
}

// codexHookHash mirrors Codex's public hook trust fingerprint: canonical
// JSON (sorted by encoding/json) of the normalized event/group/handler,
// SHA-256 prefixed with "sha256:". This lets the wrapper trust only the
// PiCode-owned session hooks, never every hook in the current repository.
func codexHookHash(spec codexHookSpec, command string) string {
	identity := map[string]any{
		"event_name": spec.key,
		"hooks": []any{map[string]any{
			"async":   false,
			"command": command,
			"timeout": spec.timeoutSec,
			"type":    "command",
		}},
	}
	var canonical bytes.Buffer
	enc := json.NewEncoder(&canonical)
	enc.SetEscapeHTML(false) // match serde_json: '<', '>' and '&' stay literal
	_ = enc.Encode(identity)
	raw := bytes.TrimSuffix(canonical.Bytes(), []byte{'\n'})
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum)
}

func codexHookOverrides(hook string) []string {
	command := hook + " auto codex"
	out := make([]string, 0, len(codexHookSpecs)+2)
	state := make([]string, 0, len(codexHookSpecs))
	for _, spec := range codexHookSpecs {
		out = append(out, fmt.Sprintf(
			"hooks.%s=[{hooks=[{type=\"command\",command=%s,timeout=%d}]}]",
			spec.event, tomlString(command), spec.timeoutSec,
		))
		key := "/<session-flags>/config.toml:" + spec.key + ":0:0"
		state = append(state, fmt.Sprintf("%s={trusted_hash=%s}",
			tomlString(key), tomlString(codexHookHash(spec, command))))
	}
	out = append(out, "hooks.state={"+strings.Join(state, ",")+"}")
	// Legacy notify remains an idle fallback for Codex builds predating hooks.
	out = append(out, fmt.Sprintf("notify=[%s,%s,%s]",
		tomlString(hook), tomlString("auto"), tomlString("codex")))
	return out
}

func writeCodexIntercept(dataDir, hook string) error {
	overrides := codexHookOverrides(hook)
	var hookArgs strings.Builder
	for _, override := range overrides {
		fmt.Fprintf(&hookArgs, " -c %q", override)
	}
	// --dangerously-bypass-hook-trust is only a capability marker. PiCode
	// deliberately does not pass it: the injected hook hashes above trust
	// these exact commands while repository hooks keep their own trust rules.
	body := "#!/bin/sh\n# PiCode intercept — Codex. Session PATH only.\nname=codex\n" +
		wrapperFindReal +
		wrapperLifecycle(hook) +
		"if \"$real\" --help 2>&1 | grep -q -- '--dangerously-bypass-hook-trust'; then\n" +
		fmt.Sprintf("  \"$real\"%s \"$@\"\n", hookArgs.String()) +
		wrapperLifecycleEnd +
		"fi\n" +
		fmt.Sprintf("\"$real\" -c %q \"$@\"\n", overrides[len(overrides)-1]) +
		wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "codex"), body)
}

func piTerminalStateExtensionFile(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "pi-terminal-state.ts")
}

// PiCode's Inbox reply receiver (ADR-0060): consumes one-shot reply files and
// submits them through the TUI's own message path. Injected into every agent
// TUI PiCode spawns, independent of the opt-in terminal-status roster.
//
//go:embed intercept/pi-inbox-reply.ts
var piInboxReplyExtensionTS string

func piReplyExtensionFile(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "pi-inbox-reply.ts")
}

func ensurePiReplyExtension(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", fmt.Errorf("data directory unknown")
	}
	if err := os.MkdirAll(interceptDir(dataDir), 0o755); err != nil {
		return "", err
	}
	path := piReplyExtensionFile(dataDir)
	if err := os.WriteFile(path, []byte(piInboxReplyExtensionTS), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

const piTerminalStateExtensionTmpl = `import { spawn } from "node:child_process";

const reporter = %s;
let pending = Promise.resolve();

function report(state, ctx) {
  if (!process.env.PICODE_TERM_ID || ctx.mode !== "tui") return Promise.resolve();
  pending = pending.then(() => new Promise((resolve) => {
    try {
      const child = spawn(reporter, [state, "pi"], { stdio: "ignore" });
      child.once("error", resolve);
      child.once("close", resolve);
    } catch {
      resolve();
    }
  }));
  return pending;
}

export default function (pi) {
  pi.on("session_start", async (_event, ctx) => report("idle", ctx));
  pi.on("agent_start", async (_event, ctx) => report("working", ctx));
  pi.on("ui_prompt_start", async (_event, ctx) => report("needs-you", ctx));
  pi.on("ui_prompt_end", async (_event, ctx) => report(ctx.isIdle() ? "idle" : "working", ctx));
  pi.on("agent_settled", async (_event, ctx) => report("idle", ctx));
  pi.on("session_shutdown", async (_event, ctx) => report("idle", ctx));
}
`

func writePiIntercept(dataDir, hook string) error {
	if err := os.MkdirAll(interceptDir(dataDir), 0o755); err != nil {
		return err
	}
	extension := piTerminalStateExtensionFile(dataDir)
	body := fmt.Sprintf(piTerminalStateExtensionTmpl, tomlString(hook))
	if err := writeInterceptFile(extension, []byte(body), 0o600); err != nil {
		return err
	}
	wrapper := "#!/bin/sh\n# PiCode intercept — Pi TUI. Session PATH only.\nname=pi\n" +
		wrapperFindReal +
		"# Pi dispatches subcommands only when they are argv[1]. Do not move them.\n" +
		"case \"${1-}\" in\n" +
		"  auth|config|install|list|remove|uninstall|update|--help|-h|--version|-v) exec \"$real\" \"$@\" ;;\n" +
		"esac\n" +
		wrapperLifecycle(hook) +
		fmt.Sprintf("\"$real\" -e %q \"$@\"\n", extension) +
		wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "pi"), wrapper)
}

func grokHomeDir(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "grok-home")
}

func writeGrokIntercept(dataDir, hook string) error {
	home := grokHomeDir(dataDir)
	if err := os.MkdirAll(filepath.Join(home, "hooks"), 0o755); err != nil {
		return err
	}
	doc := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{map[string]any{
				"hooks": []any{map[string]any{"type": "command", "command": hook + " auto grok"}},
			}},
			"UserPromptSubmit": []any{map[string]any{
				"hooks": []any{map[string]any{"type": "command", "command": hook + " auto grok"}},
			}},
			"Notification": []any{map[string]any{
				"hooks": []any{map[string]any{"type": "command", "command": hook + " auto grok"}},
			}},
			"Stop": []any{map[string]any{
				"hooks": []any{map[string]any{"type": "command", "command": hook + " auto grok"}},
			}},
			"SessionEnd": []any{map[string]any{
				"hooks": []any{map[string]any{"type": "command", "command": hook + " auto grok"}},
			}},
		},
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := writeInterceptFile(filepath.Join(home, "hooks", "picode.json"), append(raw, '\n'), 0o600); err != nil {
		return err
	}
	refresh := `user_grok="${HOME}/.grok"
mkdir -p "$GROK_HOME/hooks"
if [ -e "$user_grok/auth.json" ]; then ln -sfn "$user_grok/auth.json" "$GROK_HOME/auth.json"; fi
if [ -e "$user_grok/config.toml" ]; then ln -sfn "$user_grok/config.toml" "$GROK_HOME/config.toml"; fi
if [ -d "$user_grok/sessions" ]; then ln -sfn "$user_grok/sessions" "$GROK_HOME/sessions"; fi
`
	body := "#!/bin/sh\n# PiCode intercept — Grok. Session PATH only. GROK_HOME overlay.\nname=grok\n" +
		wrapperFindReal +
		fmt.Sprintf("export GROK_HOME=%q\n", home) +
		refresh +
		wrapperLifecycle(hook) +
		"\"$real\" \"$@\"\n" +
		wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "grok"), body)
}

func removeWrapper(dataDir, binName string) {
	_ = os.Remove(wrapperPath(dataDir, binName))
}

// stripLegacyUserClaudeHooks undoes the 2026-09-03 file-wiring if it
// ever landed in the user's ~/.claude/settings.json. Best-effort: a
// missing or foreign file is not an error.
func stripLegacyUserClaudeHooks() {
	p, err := claudeSettingsPath()
	if err != nil {
		return
	}
	_, _ = claudeSetWiring(p, "", false)
}

func installIntercept(dataDir, cliID string) error {
	hook, err := ensureHookScript(dataDir)
	if err != nil {
		return err
	}
	switch cliID {
	case "claude-code":
		stripLegacyUserClaudeHooks()
		if err := writeClaudeIntercept(dataDir, hook); err != nil {
			return err
		}
	case "codex":
		if err := writeCodexIntercept(dataDir, hook); err != nil {
			return err
		}
	case "grok":
		if err := writeGrokIntercept(dataDir, hook); err != nil {
			return err
		}
	case "pi":
		if err := writePiIntercept(dataDir, hook); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown CLI %q", cliID)
	}
	m := loadInterceptEnabled(dataDir)
	m[cliID] = true
	return saveInterceptEnabled(dataDir, m)
}

func uninstallIntercept(dataDir, cliID string) error {
	switch cliID {
	case "claude-code":
		stripLegacyUserClaudeHooks()
		removeWrapper(dataDir, "claude")
	case "codex":
		removeWrapper(dataDir, "codex")
	case "grok":
		removeWrapper(dataDir, "grok")
	case "pi":
		removeWrapper(dataDir, "pi")
		_ = os.Remove(piTerminalStateExtensionFile(dataDir))
	default:
		return fmt.Errorf("unknown CLI %q", cliID)
	}
	m := loadInterceptEnabled(dataDir)
	delete(m, cliID)
	return saveInterceptEnabled(dataDir, m)
}

func interceptWired(dataDir, cliID, binName string) bool {
	if !interceptOn(dataDir, cliID) {
		return false
	}
	st, err := os.Stat(wrapperPath(dataDir, binName))
	return err == nil && !st.IsDir()
}

func looksLikeInterceptPATH(path string) bool {
	return strings.Contains(path, string(os.PathListSeparator)) || strings.HasPrefix(path, "PATH=")
}
