package server

// Terminal CLI intercept (ADR-0056, owner 2026-09-03): never write the
// user's ~/.claude / ~/.codex / ~/.grok / ~/.hermes / ~/.config/opencode / ~/.pi. PiCode terminals prepend
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
# Hermes: interactive entry points (bare, chat, flags, a prompt) keep the lease.
# Named maintenance subcommands do not.
if [ "$name" = hermes ]; then
  case "${1-}" in
    ""|chat|-*) ;;
    HERMES_MAINT) picode_tui=0 ;;
  esac
fi
for picode_arg in "$@"; do
  case "$picode_arg" in
    -p) [ "$name" = hermes ] || picode_tui=0 ;;
    --print|--json|--headless|--non-interactive|--version|-V|--help|-h) picode_tui=0 ;;
  esac
done
picode_run_id="${PICODE_TERM_ID}-$$-$(date +%%s%%N 2>/dev/null || date +%%s)"
picode_hook=%q
if [ "$picode_tui" = 1 ] && [ -n "$PICODE_TERM_ID" ]; then
  export PICODE_TUI_RUN_ID="$picode_run_id" PICODE_TUI_PID="$$"
  "$picode_hook" runtime-start "$name" "$picode_run_id" "$$" >/dev/null 2>&1 || true
fi
`

func wrapperLifecycle(hook string) string {
	s := strings.Replace(wrapperLifecycleTmpl, "HERMES_MAINT", hermesMaintenanceCommands, 1)
	return fmt.Sprintf(s, hook)
}

// hermesMaintenanceCommands is the first positional after flags that means
// "not an interactive chat/TUI". Kept in one place so the presence lease and
// native integration passthrough skip the same names.
const hermesMaintenanceCommands = `setup|model|moa|fallback|secrets|migrate|gateway|proxy|lsp|postinstall|whatsapp|whatsapp-cloud|slack|send|login|logout|auth|status|cron|webhook|portal|kanban|project|hooks|doctor|security|dump|debug|backup|checkpoints|import|config|console|pairing|skills|bundles|plugins|curator|pets|journey|learning|memory-graph|memory|tools|computer-use|mcp|sessions|insights|claw|version|update|uninstall|acp|profile|completion|dashboard|serve|desktop|gui|logs|prompt-size`

// opencodeMaintenanceCommands is the first positional after flags that means
// "not the interactive TUI". A path positional is the TUI project argument,
// not a subcommand. Kept in one place so the presence lease skips the same
// names the OPENCODE_CONFIG plugin skip uses.
const opencodeMaintenanceCommands = `session|auth|providers|mcp|models|serve|web|acp|plugin|plug|db|upgrade|uninstall|github|stats|export|import|completion|agent|debug|run|attach|pr`

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
		"\"$real\"" + quotedCLIArgs(cliIntegrationPlan("claude-code", dataDir, hook).Branches[0].Args) + " \"$@\"\n" +
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

// Codex resume/fork owns its own -c parser. Root overrides before the
// subcommand are discarded there, including lifecycle hooks and their trust.
func codexInvoke(args []string) string {
	flags := quotedCLIArgs(args)
	return "case \"${1-}\" in\n" +
		" resume|fork) \"$real\" \"$@\"" + flags + " ;;\n" +
		" *) \"$real\"" + flags + " \"$@\" ;;\nesac\n"
}

func writeCodexIntercept(dataDir, hook string) error {
	branches := cliIntegrationPlan("codex", dataDir, hook).Branches
	// --dangerously-bypass-hook-trust is only a capability marker. PiCode
	// deliberately does not pass it: the injected hook hashes above trust
	// these exact commands while repository hooks keep their own trust rules.
	body := "#!/bin/sh\n# PiCode intercept — Codex. Session PATH only.\nname=codex\n" +
		wrapperFindReal +
		wrapperLifecycle(hook) +
		"if \"$real\" --help 2>&1 | grep -q -- '--dangerously-bypass-hook-trust'; then\n" +
		codexInvoke(branches[0].Args) +
		wrapperLifecycleEnd +
		"fi\n" +
		codexInvoke(branches[1].Args) +
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
      const child = spawn(reporter, [state, "pi"], { stdio: "ignore", env: { ...process.env, PICODE_NATIVE_SESSION_ID: ctx.sessionManager.getSessionId(), PICODE_NATIVE_SESSION_PATH: ctx.sessionManager.getSessionFile() || "" } });
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
	// The wrapper names the receiver too (cliIntegrationPlan), so it has to
	// exist before the first pi launches through it.
	if _, err := ensurePiReplyExtension(dataDir); err != nil {
		return err
	}
	piArgs := quotedCLIArgs(cliIntegrationPlan("pi", dataDir, hook).Branches[0].Args)
	wrapper := "#!/bin/sh\n# PiCode intercept — Pi TUI. Session PATH only.\nname=pi\n" +
		wrapperFindReal +
		"# Pi dispatches subcommands only when they are argv[1]. Do not move them.\n" +
		"case \"${1-}\" in\n" +
		"  " + strings.Join(piPassthrough, "|") + ") exec \"$real\" \"$@\" ;;\n" +
		"esac\n" +
		"# Managed rpc runs exec the real pi: a shell middleman between PiCode\n" +
		"# and the agent process is the orphaned-child class the runtime's\n" +
		"# process-group kill exists for — drop the middleman outright.\n" +
		"case \"pi:${1-}:${2-}\" in\n" +
		"  pi:--mode:rpc|pi:--mode:json) exec \"$real\"" + piArgs + " \"$@\" ;;\n" +
		"esac\n" +
		wrapperLifecycle(hook) +
		"\"$real\"" + piArgs + " \"$@\"\n" +
		wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "pi"), wrapper)
}

func writeGrokIntercept(dataDir, hook string) error {
	if err := writeNativeAssets(dataDir); err != nil {
		return err
	}
	body := "#!/bin/sh\n# PiCode native Grok integration.\nname=grok\n" + wrapperFindReal +
		nativeWrapperSetup(dataDir, hook, "grok") + wrapperLifecycle(hook) + "\"$real\" \"$@\"\n" + wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "grok"), body)
}

func writeHermesIntercept(dataDir, hook string) error {
	if err := writeNativeAssets(dataDir); err != nil {
		return err
	}
	passthrough := strings.Replace(`# Named maintenance subcommands skip the session patch, even when flags
# precede them (hermes -p NAME setup). Session ids after --resume/-r/-c
# and extra launch arguments are not subcommands.
picode_take=
for picode_arg in "$@"; do
  if [ -n "$picode_take" ]; then
    picode_take=
    continue
  fi
  case "$picode_arg" in
    -p|--profile|--resume|-r|-c|--provider|--model|-m|--skills|-t|--toolsets|--session)
      picode_take=1
      continue
      ;;
    --profile=*|--resume=*|--provider=*|--model=*|--skills=*|--toolsets=*|--session=*)
      continue
      ;;
    --|-*) continue ;;
    chat) break ;;
    HERMES_MAINT)
      exec "$real" "$@"
      ;;
    *) break ;;
  esac
done
`, "HERMES_MAINT", hermesMaintenanceCommands, 1)
	body := "#!/bin/sh\n# PiCode native Hermes plugin.\nname=hermes\n" + wrapperFindReal + passthrough + nativeWrapperSetup(dataDir, hook, "hermes") + wrapperLifecycle(hook) + "\"$real\" \"$@\"\n" + wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "hermes"), body)
}

func opencodeInterceptDir(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "opencode")
}

func opencodeConfigFile(dataDir string) string {
	return filepath.Join(opencodeInterceptDir(dataDir), "opencode.json")
}

func opencodePluginFile(dataDir string) string {
	return filepath.Join(opencodeInterceptDir(dataDir), "picode-activity.js")
}

// opencodeMapEventJS is the event → terminal-state table. Kept as a
// standalone function so tests can execute it without loading OpenCode.
const opencodeMapEventJS = `function mapEvent(type, statusType) {
  switch (type) {
    case "session.status":
      if (statusType === "busy" || statusType === "retry") return "working"
      if (statusType === "idle") return "idle"
      return ""
    case "session.idle":
      return "idle"
    case "permission.asked":
    case "permission.v2.asked":
    case "question.asked":
    case "question.v2.asked":
      return "needs-you"
    case "permission.replied":
    case "permission.v2.replied":
    case "question.replied":
    case "question.v2.replied":
      return "working"
    case "question.rejected":
    case "question.v2.rejected":
      return "idle"
    default:
      return ""
  }
}
`

// opencodeActivityJS is a session-only OpenCode plugin. It exports one
// default function — extra function exports would be loaded as plugins.
const opencodeActivityJS = `import { spawn } from "node:child_process"

const HOOK = process.env.PICODE_OPENCODE_HOOK || ""
let sequence = 0n

function report(state, sessionId) {
  if (!HOOK || !process.env.PICODE_TERM_ID) return
  if (state !== "working" && state !== "idle" && state !== "needs-you") return
  try {
    const child = spawn(HOOK, [state, "opencode"], { stdio: "ignore", detached: true, env: { ...process.env, PICODE_NATIVE_SESSION_ID: sessionId || "", PICODE_NATIVE_SESSION_SEQ: String(sequence = (BigInt(Date.now()) * 1000000n > sequence ? BigInt(Date.now()) * 1000000n : sequence + 1n)) } })
    child.unref()
  } catch {
    // never throw into the TUI
  }
}

` + opencodeMapEventJS + `
// Only a root receiving a native user message becomes the selected conversation.
// Other roots and child status events on the local server cannot steal it.
export default async function picodeActivity({ client }) {
  let selected = ""
  let queue = Promise.resolve()
  const roots = new Map()
  async function root(id) {
    if (typeof id !== "string" || !id) return false
    if (roots.has(id)) return roots.get(id)
    try {
      const result = await client.session.get({ path: { id } })
      const session = result?.data
      if (!session || session.id !== id) return false
      const value = !session.parentID
      if (roots.size >= 128) roots.clear()
      roots.set(id, value)
      return value
    } catch { return false }
  }
  const resumed = process.env.PICODE_OPENCODE_SESSION_ID
  if (await root(resumed)) selected = resumed
  function ordered(fn) {
    queue = queue.then(fn).catch(() => {})
    return queue
  }
  return {
    "chat.message": async ({ sessionID }) => ordered(async () => {
      if (await root(sessionID)) {
        selected = sessionID
        report("working", selected)
      }
    }),
    event: async ({ event }) => ordered(async () => {
      const id = event?.properties?.sessionID
      if (!selected || id !== selected || !(await root(id))) return
      const state = mapEvent(event?.type, event?.properties?.status?.type)
      if (state) report(state, id)
    }),
  }
}

`

const opencodeConfigJSON = `{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["./picode-activity.js"]
}
`

func writeOpencodeIntercept(dataDir, hook string) error {
	if err := writeInterceptFile(opencodePluginFile(dataDir), []byte(opencodeActivityJS), 0o600); err != nil {
		return err
	}
	if err := writeInterceptFile(opencodeConfigFile(dataDir), []byte(opencodeConfigJSON), 0o600); err != nil {
		return err
	}
	plan := cliIntegrationPlan("opencode", dataDir, hook)
	passthrough := strings.Replace(`# Named maintenance subcommands skip the presence lease and the session
# plugin, even when flags precede them (opencode --log-level DEBUG session
# list). A path positional is the TUI project argument, not a subcommand.
# Session ids after --session/-s are not subcommands.
picode_take=
for picode_arg in "$@"; do
  if [ -n "$picode_take" ]; then
    picode_take=
    continue
  fi
  case "$picode_arg" in
    -s|--session|-m|--model|--agent|--prompt|--port|--hostname|--mdns-domain|--cors|--log-level|--replay-limit)
      picode_take=1
      continue
      ;;
    --session=*|--model=*|--agent=*|--prompt=*|--port=*|--hostname=*|--mdns-domain=*|--cors=*|--log-level=*|--replay-limit=*)
      continue
      ;;
    --|-*) continue ;;
    OPENCODE_MAINT)
      exec "$real" "$@"
      ;;
    *) break ;;
  esac
done
`, "OPENCODE_MAINT", opencodeMaintenanceCommands, 1)
	body := "#!/bin/sh\n# PiCode intercept — OpenCode. Session PATH only. OPENCODE_CONFIG plugin; no data-dir overlay.\nname=opencode\n" +
		wrapperFindReal +
		passthrough +
		"export OPENCODE_CONFIG=" + shellQuote(plan.Environment["OPENCODE_CONFIG"]) + "\n" +
		"export PICODE_OPENCODE_HOOK=" + shellQuote(plan.Environment["PICODE_OPENCODE_HOOK"]) + "\n" +
		wrapperLifecycle(hook) +
		"printf 'Starting OpenCode...\\n'\n" +
		"\"$real\" \"$@\"\n" +
		wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "opencode"), body)
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
	case "hermes":
		if err := writeHermesIntercept(dataDir, hook); err != nil {
			return err
		}
	case "opencode":
		if err := writeOpencodeIntercept(dataDir, hook); err != nil {
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
	case "hermes":
		removeWrapper(dataDir, "hermes")
	case "opencode":
		removeWrapper(dataDir, "opencode")
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
