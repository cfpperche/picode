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

	"github.com/cfpperche/picode/internal/pimission"
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
	m := loadInterceptEnabled(dataDir)
	switch cliID {
	case TmuxGuardID:
		// ADR-0138: the tmux guard defaults on — three measured incidents are
		// the context. Only an explicit opt-out (wiring disable) turns it off.
		on, seen := m[cliID]
		return !seen || on
	case OpenURLID:
		// ADR-0180: the browser hand-off defaults on for the same reason.
		on, seen := m[cliID]
		return !seen || on
	}
	return m[cliID]
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

const wrapperFindReal = `# Another PiCode wrapper is never the real binary: two instances' bin dirs
# on one PATH (a scratch terminal inside PiCode) made each wrapper exec the
# other forever, one pid at full CPU (2026-09-25). Same test as the Go side's
# isCLIWrapper, in shell builtins only.
picode_wrapper() {
  { IFS= read -r picode_l1 && IFS= read -r picode_l2; } < "$1" 2>/dev/null || return 1
  [ "$picode_l1" = "#!/bin/sh" ] || return 1
  case "$picode_l2" in "# PiCode "*) return 0 ;; esac
  return 1
}
here=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
real=
IFS=:
for d in $PATH; do
  [ "$d" = "$here" ] && continue
  if [ -x "$d/$name" ] && ! picode_wrapper "$d/$name"; then real="$d/$name"; break; fi
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
  omp:--mode:rpc|omp:--mode:json|omp:--mode:acp|omp:--mode:rpc-ui|omp:--mode=rpc*|omp:--mode=json*|omp:--mode=acp*) picode_tui=0 ;;
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
	{event: "PreCompact", key: "pre_compact", timeoutSec: 5},
	{event: "PostCompact", key: "post_compact", timeoutSec: 5},
	{event: "SessionStart", key: "session_start", timeoutSec: 5},
	{event: "PermissionRequest", key: "permission_request", timeoutSec: 5},
	// PostToolUse is the permission-resume signal: Codex reports a waiting
	// permission UI as needs-you and has no "permission resolved" event, so
	// the approved tool's completion is what returns the terminal to working.
	{event: "PostToolUse", key: "post_tool_use", timeoutSec: 5},
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
	return out
}

func codexNotifyOverride(hook string) string {
	return fmt.Sprintf("notify=[%s,%s,%s]", tomlString(hook), tomlString("auto"), tomlString("codex"))
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
		"export PICODE_CODEX_HOOKS=1\n" +
		codexInvoke(branches[0].Args) +
		wrapperLifecycleEnd +
		"fi\n" +
		"unset PICODE_CODEX_HOOKS\n" +
		codexInvoke(branches[1].Args) +
		wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "codex"), body)
}

func piTerminalStateExtensionFile(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "pi-terminal-state.ts")
}

// ompTerminalStateExtensionFile is the omp reporter: a pi fork whose
// extension API accepts the same load mechanism, but a different event set
// (verified against the 18.2.6 bundle: agent_start/agent_end with
// willContinue, tool_approval_requested/resolved, tool_execution_start and
// tool_result — while the pi events agent_settled / ui_prompt_start /
// ui_prompt_end never fire, so pi's template left omp stuck on Working).
func ompTerminalStateExtensionFile(dataDir string) string {
	return filepath.Join(interceptDir(dataDir), "omp-terminal-state.ts")
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
      const child = spawn(reporter, [state, "%s"], { stdio: "ignore", env: { ...process.env, PICODE_NATIVE_SESSION_ID: ctx.sessionManager.getSessionId(), PICODE_NATIVE_SESSION_PATH: ctx.sessionManager.getSessionFile() || "" } });
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
  pi.on("session_before_compact", async (_event, ctx) => report("compacting", ctx));
  pi.on("session_compact", async (_event, ctx) => report(ctx.isIdle() ? "idle" : "working", ctx));
  pi.on("session_compact_failed", async (_event, ctx) => report(ctx.isIdle() ? "idle" : "working", ctx));
  pi.on("ui_prompt_start", async (_event, ctx) => report("needs-you", ctx));
  pi.on("ui_prompt_end", async (_event, ctx) => report(ctx.isIdle() ? "idle" : "working", ctx));
  pi.on("agent_settled", async (_event, ctx) => report("idle", ctx));
  pi.on("session_shutdown", async (_event, ctx) => report("idle", ctx));
}
`

// omp shares pi's extension load mechanism but not its event set — verified
// against the 18.2.6 bundle: agent_start is always emitted, agent_end
// carries willContinue (mid-run turns must not settle), and approvals and
// the ask question card arrive as tool_approval_requested/resolved and
// tool_execution_start{toolName:"ask"}/tool_result. The pi events
// agent_settled and ui_prompt_* never fire, so pi's template left omp stuck
// on Working and blind to every approval. turn_start/turn_end fire many
// times per run and carry no state.
const ompTerminalStateExtensionTmpl = `import { spawn } from "node:child_process";

const reporter = %s;
let pending = Promise.resolve();

function report(state, ctx) {
  if (!process.env.PICODE_TERM_ID || ctx.mode !== "tui") return Promise.resolve();
  pending = pending.then(() => new Promise((resolve) => {
    try {
      const child = spawn(reporter, [state, "omp"], { stdio: "ignore", env: { ...process.env, PICODE_NATIVE_SESSION_ID: ctx.sessionManager.getSessionId(), PICODE_NATIVE_SESSION_PATH: ctx.sessionManager.getSessionFile() || "" } });
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
  pi.on("session_before_compact", async (_event, ctx) => report("compacting", ctx));
  pi.on("session_compact", async (_event, ctx) => report(ctx.isIdle() ? "idle" : "working", ctx));
  pi.on("tool_approval_requested", async (_event, ctx) => report("needs-you", ctx));
  pi.on("tool_approval_resolved", async (_event, ctx) => report("working", ctx));
  pi.on("tool_execution_start", async (_event, ctx) => {
    if (_event && _event.toolName === "ask") return report("needs-you", ctx);
  });
  pi.on("tool_result", async (_event, ctx) => report("working", ctx));
  pi.on("agent_end", async (_event, ctx) => {
    if (_event && _event.willContinue) return;
    return report("idle", ctx);
  });
  pi.on("session_shutdown", async (_event, ctx) => report("idle", ctx));
}
`

func writePiIntercept(dataDir, hook string) error {
	if err := os.MkdirAll(interceptDir(dataDir), 0o755); err != nil {
		return err
	}
	extension := piTerminalStateExtensionFile(dataDir)
	body := fmt.Sprintf(piTerminalStateExtensionTmpl, tomlString(hook), "pi")
	if err := writeInterceptFile(extension, []byte(body), 0o600); err != nil {
		return err
	}
	// The wrapper names the receiver too (cliIntegrationPlan), so it has to
	// exist before the first pi launches through it.
	if _, err := ensurePiReplyExtension(dataDir); err != nil {
		return err
	}
	if _, err := pimission.Ensure(dataDir); err != nil {
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

// museMaintenanceCommands is the first positional that means "not the
// interactive TUI". Bare runs (with or without a prompt) and resume keep
// the presence lease; every vendor subcommand below is headless, auth,
// setup or inspection. Kept in one place so future subcommands land here,
// next to the hermes/opencode lists above.
const museMaintenanceCommands = `exec|export|trace|skills|sandbox|schema|session-message|mcp|auth|login|logout|init|config|serve`

// agyMaintenanceCommands is the first positional that means "not the
// interactive TUI". Bare runs and --prompt-interactive/--conversation
// continuations keep the lease; print mode is already caught by the
// generic --print loop below, subcommands here complete it.
const agyMaintenanceCommands = `agent|agents|changelog|help|install|mcp|mic-serve|models|plugin|plugins|remote-control|update`

// ompMaintenanceCommands is the first positional that means "not an omp
// agent session": every subcommand omp dispatches — auth, config, models,
// protocol servers, even its interactive tools (git, shell) — is not a
// session the presence lease should describe. Only bare runs (with or
// without a prompt), -c/--continue and -r/--resume keep it; protocol
// modes (--mode rpc/json/acp/rpc-ui) are caught by the wrapperLifecycle
// case above. Verified against `omp --help` on 18.2.4.
const ompMaintenanceCommands = `acp|agents|auth-broker|auth-gateway|bench|browser-relay|cleanse|collab|commit|completions|compress|config|dry-balance|gallery|gc|git|grep|grievances|if-bench|images|install|join|models|plugin|ps|read|render|say|search|setup|share|shell|ssh|stats|tiny-models|token|ttsr|update|usage|worktree`

func writeOmpIntercept(dataDir, hook string) error {
	if err := os.MkdirAll(interceptDir(dataDir), 0o755); err != nil {
		return err
	}
	extension := ompTerminalStateExtensionFile(dataDir)
	body := fmt.Sprintf(ompTerminalStateExtensionTmpl, tomlString(hook))
	if err := writeInterceptFile(extension, []byte(body), 0o600); err != nil {
		return err
	}
	// The wrapper names Pi's reply receiver too (cliIntegrationPlan).
	if _, err := ensurePiReplyExtension(dataDir); err != nil {
		return err
	}
	ompArgs := quotedCLIArgs(cliIntegrationPlan("omp", dataDir, hook).Branches[0].Args)
	passthrough := `# Named maintenance subcommands skip the presence lease and the extension.
# Bare runs (with or without a prompt), -c/--continue and -r/--resume keep both.
# --export renders a file and exits: exec straight past the lease (measured:
# it is a one-shot render, not a session).
if [ "$name" = omp ]; then
  case "${1-}" in
    "") ;;
    --export|--export=*)
      exec "$real" "$@"
      ;;
    -*) ;;
    OMP_MAINT)
      exec "$real" "$@"
      ;;
  esac
fi
`
	passthrough = strings.Replace(passthrough, "OMP_MAINT", ompMaintenanceCommands, 1)
	wrapper := "#!/bin/sh\n# PiCode Omp integration (omp is a Pi fork: pi-shaped extension).\nname=omp\n" +
		wrapperFindReal + passthrough + wrapperLifecycle(hook) +
		"\"$real\"" + ompArgs + " \"$@\"\n" +
		wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "omp"), wrapper)
}

func writeMuseIntercept(dataDir, hook string) error {
	// Maintenance subcommands exec straight past the lease below (the
	// hermes shape): wrapperLifecycle owns the picode_tui=1 default, so a
	// later assignment would only take effect after runtime-start already
	// ran. Bare runs (with or without a starting prompt) and resume keep it.
	passthrough := `# Named maintenance subcommands skip the presence lease.
# Bare runs (with or without a starting prompt) and resume keep it.
if [ "$name" = muse ]; then
  case "${1-}" in
    ""|resume|-*) ;;
    MUSE_MAINT)
      exec "$real" "$@"
      ;;
  esac
fi
`
	passthrough = strings.Replace(passthrough, "MUSE_MAINT", museMaintenanceCommands, 1)
	body := "#!/bin/sh\n# PiCode Muse Code integration.\nname=muse\n" + wrapperFindReal + passthrough + wrapperLifecycle(hook) + "\"$real\" \"$@\"\n" + wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "muse"), body)
}

func writeAgyIntercept(dataDir, hook string) error {
	passthrough := `# Named maintenance subcommands skip the presence lease.
# Bare runs, prompt-interactive and conversation continuations keep it.
if [ "$name" = agy ]; then
  case "${1-}" in
    ""|-*) ;;
    AGY_MAINT)
      exec "$real" "$@"
      ;;
  esac
fi
`
	passthrough = strings.Replace(passthrough, "AGY_MAINT", agyMaintenanceCommands, 1)
	body := "#!/bin/sh\n# PiCode Antigravity integration.\nname=agy\n" + wrapperFindReal + passthrough + wrapperLifecycle(hook) + "\"$real\" \"$@\"\n" + wrapperLifecycleEnd
	return writeExecutable(wrapperPath(dataDir, "agy"), body)
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
  if (state !== "working" && state !== "compacting" && state !== "idle" && state !== "needs-you") return
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
  // This is only a candidate until root() validates a native event below.
  // Calling the session API during plugin initialization deadlocks OpenCode:
  // that API waits for the instance, which is waiting for this plugin.
  let selected = process.env.PICODE_OPENCODE_SESSION_ID || ""
  let activity = 0
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
  function ordered(fn) {
    queue = queue.then(fn).catch(() => {})
    return queue
  }
  // Resume need not emit an idle event. Query native status after returning
  // the plugin, never on its initialization promise or the event queue.
  // A newer native event wins over this one-time snapshot.
  if (selected) setTimeout(() => {
    if (activity !== 0) return
    const id = selected, generation = activity
    void (async () => {
      if (!(await root(id))) return
      const result = await client.session.status()
      if (!result?.data || typeof result.data !== "object" || Array.isArray(result.data)) return
      const state = mapEvent("session.status", Object.hasOwn(result.data, id) ? result.data[id]?.type : "idle")
      await ordered(async () => {
        if (state && selected === id && activity === generation) report(state, id)
      })
    })().catch(() => {})
  }, 0)
  return {
    "experimental.session.compacting": async ({ sessionID }) => ordered(async () => {
      if (selected && sessionID === selected && await root(sessionID)) report("compacting", sessionID)
    }),
    "chat.message": async ({ sessionID }) => {
      activity++
      return ordered(async () => {
      if (await root(sessionID)) {
        selected = sessionID
        report("working", selected)
      }
      })
    },
    event: async ({ event }) => {
      const id = event?.properties?.sessionID
      if (id === selected && (mapEvent(event?.type, event?.properties?.status?.type) || event?.type === "session.compacted")) activity++
      return ordered(async () => {
      if (!selected || id !== selected || !(await root(id))) return
      if (event?.type === "session.compacted") {
        const result = await client.session.status()
        const state = mapEvent("session.status", result?.data?.[id]?.type || "idle")
        report(state || "idle", id)
        return
      }
      const state = mapEvent(event?.type, event?.properties?.status?.type)
      if (state) report(state, id)
      })
    },
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

// TmuxGuardID is the wiring row of the tmux guard (ADR-0138). It is not a
// clilaunch CLI: enable/disable go through installTmuxGuard/uninstallTmuxGuard,
// and the wrapper gates only the sessions PiCode creates (ADR-0056 PATH).
const TmuxGuardID = "tmux-guard"

func tmuxGuardLog(dataDir string) string { return filepath.Join(dataDir, "tmux-guard.log") }

// tmuxGuardWrapper is the policy of ADR-0138, one table, enforced in order.
// It stays a small POSIX sh so it loads before any user rc and can never
// outlive the real binary it execs. The log path is baked in at write time.
const tmuxGuardWrapper = `#!/bin/sh
# PiCode intercept — tmux guard (ADR-0138). Session PATH only.
#   kill-server / kill-window / kill-pane ............. refuse
#   kill-session -a (all but target) .................. refuse
#   kill-session -t PATTERN ........................... refuse
#   kill-session -t NAME (marker != this terminal) .... refuse
#   kill-session -t NAME (marker == this terminal) .... allow
#   send-keys carrying kill-server / pkill / killall .. refuse
#   new-session (simple form) ......................... stamp PICODE_TERM_ID
#   anything else (ls, capture-pane, attach, ...) ..... passthrough
# Outside a managed terminal (no PICODE_TERM_ID) this wrapper is not on
# PATH; invoked directly it passes straight through. This is a guardrail,
# not a security boundary: plain accidents must be safe, circumvention
# must be deliberate. The probes carry no -L/-S on purpose: inside a pane
# they inherit $TMUX — that pane's own server, the dedicated socket
# included (ADR-0139) — and outside one they ask the default server, where
# an exact-name kill aimed at another socket finds no marked session and is
# refused (fail-closed).
name=tmux
` + guardFindReal + `
picode_log='__PICODE_GUARD_LOG__'
picode_argv=$*

picode_refuse() {
  printf '%s\n' "picode tmux guard: refused. $1" >&2
  printf '%s\trefused\tterm=%s\t%s\n' \
    "$(date +%Y-%m-%dT%H:%M:%S 2>/dev/null)" "${PICODE_TERM_ID-}" "$picode_argv" \
    >>"$picode_log" 2>/dev/null || true
  exit 1
}

[ -n "${PICODE_TERM_ID-}" ] || exec "$real" "$@"

picode_cmd=
picode_skip=
picode_target=
picode_allbut=
picode_pending_t=
for picode_a in "$@"; do
  if [ -n "$picode_skip" ]; then picode_skip=; continue; fi
  if [ -z "$picode_cmd" ]; then
    case "$picode_a" in
      -c|-f|-L|-S) picode_skip=1 ;;
      -c*|-f*|-L*|-S*) ;;
      -*) ;;
      *) picode_cmd=$picode_a ;;
    esac
    continue
  fi
  case "$picode_a" in
    -t) picode_pending_t=1 ;;
    -t?*) picode_target=${picode_a#-t} ;;
    # An 'a' anywhere in a flag cluster is kill-session's all-but-target
    # (-a, -at X, -aC); fail-closed — only kill-session reads it, other
    # commands ignore it. Must sit after the -t rules: '-ta' is a target.
    -*a*) picode_allbut=1 ;;
    *) [ -n "$picode_pending_t" ] && { picode_target=$picode_a; picode_pending_t=; } ;;
  esac
done

case "$picode_cmd" in
  kill-server|kill-window|kill-pane)
    picode_refuse "'$picode_cmd' reaches beyond this terminal: it takes down every session on the server, PiCode's and yours. Close terminals from the app, or kill one exact session: tmux kill-session -t <name>."
    ;;
  kill-session)
    [ -n "$picode_allbut" ] && picode_refuse "'kill-session -a' kills every other session on the server; the guard allows only exact, owned kills."
    if [ -z "$picode_target" ]; then
      if [ -n "${TMUX-}" ]; then
        picode_target=$("$real" display-message -p '#S' 2>/dev/null)
      fi
      [ -n "$picode_target" ] || exec "$real" "$@"
    fi
    case "$picode_target" in
      *\**|*\?*|*\[*|*\]*|*\\*)
        picode_refuse "'$picode_target' is a pattern. Only exact session names may be killed — run 'tmux ls' and pick one name."
        ;;
    esac
    picode_owner=$("$real" show-environment -t "$picode_target" PICODE_TERM_ID 2>/dev/null)
    picode_owner=${picode_owner#PICODE_TERM_ID=}
    [ "$picode_owner" = "$PICODE_TERM_ID" ] || picode_refuse "'$picode_target' was not created by this terminal. Only sessions this terminal created may be killed."
    ;;
  send-keys)
    case "$picode_argv" in
      *kill-server*|*pkill*|*killall*)
        picode_refuse "send-keys would type a server or process killer into a pane. If that pane is yours, run the command there yourself."
        ;;
    esac
    ;;
  new-session)
    # A plain child session does not inherit the marker (measured, tmux 3.6),
    # so stamp the sessions this terminal creates — you own what you create.
    # Only the simple form (argv[1] is the command) is rewritten; exotic
    # server flags pass through unstamped and stay kill-refused.
    if [ "${1-}" = "new-session" ]; then
      case "$picode_argv" in
        *PICODE_TERM_ID=*) ;;
        *) shift; exec "$real" new-session -e "PICODE_TERM_ID=$PICODE_TERM_ID" "$@" ;;
      esac
    fi
    ;;
  mine)
    [ "${1-}" = "mine" ] && {
      "$real" list-sessions -F '#S' 2>/dev/null | while IFS= read -r picode_s; do
        picode_o=$("$real" show-environment -t "$picode_s" PICODE_TERM_ID 2>/dev/null)
        [ "${picode_o#PICODE_TERM_ID=}" = "$PICODE_TERM_ID" ] && printf '%s\n' "$picode_s"
      done
      exit 0
    }
    ;;
esac

exec "$real" "$@"
`

// guardFindReal resolves the real tmux without external binaries. The
// shared wrapperFindReal shells out to dirname(1); when the guard ran under
// a minimal PATH (a test fixture, a hardened launch), dirname was absent,
// the computed `here` was wrong, and the guard found *itself* in the bin
// dir and exec'd in an endless self-loop (caught 2026-09-15). A guard must
// not depend on the environment it polices: ${0%/*} is pure shell.
const guardFindReal = `# Another PiCode wrapper is never the real binary: two instances' bin dirs
# on one PATH (a scratch terminal inside PiCode) made each wrapper exec the
# other forever, one pid at full CPU (2026-09-25). Same test as the Go side's
# isCLIWrapper, in shell builtins only.
picode_wrapper() {
  { IFS= read -r picode_l1 && IFS= read -r picode_l2; } < "$1" 2>/dev/null || return 1
  [ "$picode_l1" = "#!/bin/sh" ] || return 1
  case "$picode_l2" in "# PiCode "*) return 0 ;; esac
  return 1
}
here=${0%/*}
[ "$here" = "$0" ] && here=.
real=
IFS=:
for d in $PATH; do
  [ "$d" = "$here" ] && continue
  if [ -x "$d/$name" ] && ! picode_wrapper "$d/$name"; then real="$d/$name"; break; fi
done
unset IFS
if [ -z "$real" ]; then
  printf '%s\n' "picode: $name is not installed outside this terminal." >&2
  exit 127
fi
`

func tmuxGuardBody(dataDir string) string {
	return strings.Replace(tmuxGuardWrapper, "__PICODE_GUARD_LOG__", tmuxGuardLog(dataDir), 1)
}

func writeTmuxGuard(dataDir string) error {
	return writeExecutable(wrapperPath(dataDir, "tmux"), tmuxGuardBody(dataDir))
}

func installTmuxGuard(dataDir string) error {
	if err := writeTmuxGuard(dataDir); err != nil {
		return err
	}
	m := loadInterceptEnabled(dataDir)
	if m[TmuxGuardID] {
		return nil
	}
	m[TmuxGuardID] = true
	return saveInterceptEnabled(dataDir, m)
}

func uninstallTmuxGuard(dataDir string) error {
	removeWrapper(dataDir, "tmux")
	// Persist an explicit false: the guard defaults on, so a deleted key
	// would read as "never configured" and silently re-arm the guard.
	m := loadInterceptEnabled(dataDir)
	m[TmuxGuardID] = false
	return saveInterceptEnabled(dataDir, m)
}

// ensureTmuxGuard (re)creates the wrapper when the guard is on and the file
// is missing — an upgrade or an operator's clean-up must not silently leave
// new sessions unguarded. Called at session creation, beside the rcfile.
func ensureTmuxGuard(dataDir string) {
	if !interceptOn(dataDir, TmuxGuardID) {
		return
	}
	// Rewrite when missing or stale: a guard written by an older binary kept
	// its old body across every deploy (2026-09-25: the two-instance exec
	// loop stayed on disk after its fix). Boot calls this too.
	if b, err := os.ReadFile(wrapperPath(dataDir, "tmux")); err == nil && string(b) == tmuxGuardBody(dataDir) {
		return
	}
	_ = writeTmuxGuard(dataDir)
}

func installIntercept(dataDir, cliID string) error {
	hook, err := ensureHookScript(dataDir)
	if err != nil {
		return err
	}
	switch cliID {
	case "claude-code":
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
	case "agy":
		if err := installAgyTitleReporter(dataDir); err != nil {
			return err
		}
		if err := writeAgyIntercept(dataDir, hook); err != nil {
			return err
		}
	case "muse":
		if err := writeMuseIntercept(dataDir, hook); err != nil {
			return err
		}
	case "omp":
		if err := writeOmpIntercept(dataDir, hook); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown CLI %q", cliID)
	}
	m := loadInterceptEnabled(dataDir)
	m[cliID] = true
	return saveInterceptEnabled(dataDir, m)
}

// agySettingsPath is the CLI's own settings file. There is no flag or
// env to point the CLI elsewhere, so the title reporter installs here —
// merged key by key, never overwritten, and only our own block removed.
func agySettingsPath() (string, error) {
	return homeFile(".gemini", "antigravity-cli", "settings.json")
}

func agyTitleBlock(reporter string) map[string]any {
	return map[string]any{"type": "command", "command": reporter, "enabled": true}
}

// installAgyTitleReporter merges our title block into the user's settings.
// A foreign title command is refused, never replaced: the file is theirs.
//
// Exception to the retired user-home writes (2026-09-03): agy offers
// no --settings flag and no config env (full flag list and binary strings
// checked, Fatia 5), so its own settings.json is the only install path.
// The merge is one key, foreign blocks refuse loudly, and removal deletes
// only what we installed.
func installAgyTitleReporter(dataDir string) error {
	reporter := agyTitleReporterPath(dataDir)
	body := strings.Replace(agyTitleSh, "__PICODE_HOOK_ABS__", hookScriptPath(dataDir), 1)
	if !strings.Contains(body, hookScriptPath(dataDir)) {
		return fmt.Errorf("reporter template lost its hook path placeholder")
	}
	if err := writeExecutable(reporter, body); err != nil {
		return err
	}
	settings, err := agySettingsPath()
	if err != nil {
		return err
	}
	doc := map[string]any{}
	mode := os.FileMode(0o600)
	if raw, err := os.ReadFile(settings); err == nil {
		if st, serr := os.Stat(settings); serr == nil {
			mode = st.Mode().Perm()
		}
		if uerr := json.Unmarshal(raw, &doc); uerr != nil {
			return fmt.Errorf("your Antigravity settings.json does not parse; PiCode will not touch it")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if cur, ok := doc["title"].(map[string]any); ok {
		if cmd, _ := cur["command"].(string); cmd == reporter {
			return nil
		}
		return fmt.Errorf("your Antigravity title command is custom; PiCode will not replace it")
	} else if _, present := doc["title"]; present {
		return fmt.Errorf("your Antigravity title setting has an unknown shape; PiCode will not touch it")
	}
	doc["title"] = agyTitleBlock(reporter)
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writeInterceptFile(settings, append(raw, '\n'), mode)
}

// removeAgyTitleReporter removes only our own block, and the file itself
// when we created it (nothing else left in it).
func removeAgyTitleReporter(dataDir string) {
	reporter := agyTitleReporterPath(dataDir)
	_ = os.Remove(reporter)
	settings, err := agySettingsPath()
	if err != nil {
		return
	}
	raw, err := os.ReadFile(settings)
	if err != nil {
		return
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return
	}
	cur, ok := doc["title"].(map[string]any)
	if !ok {
		return
	}
	if cmd, _ := cur["command"].(string); cmd != reporter {
		return
	}
	delete(doc, "title")
	if len(doc) == 0 {
		_ = os.Remove(settings)
		return
	}
	if out, err := json.MarshalIndent(doc, "", "  "); err == nil {
		if st, serr := os.Stat(settings); serr == nil {
			_ = writeInterceptFile(settings, append(out, '\n'), st.Mode().Perm())
		}
	}
}

func uninstallIntercept(dataDir, cliID string) error {
	switch cliID {
	case "claude-code":
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
	case "omp":
		removeWrapper(dataDir, "omp")
		_ = os.Remove(ompTerminalStateExtensionFile(dataDir))
	case "agy":
		removeAgyTitleReporter(dataDir)
		removeWrapper(dataDir, "agy")
	case "muse":
		removeWrapper(dataDir, "muse")
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
