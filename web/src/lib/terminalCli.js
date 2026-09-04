const CLI_ALIASES = Object.freeze({
  claude: "claude-code",
  "claude-code": "claude-code",
  codex: "codex",
  grok: "grok",
  pi: "pi",
});

const CLI_LABELS = Object.freeze({
  "claude-code": "Claude Code",
  codex: "Codex",
  grok: "Grok",
  pi: "Pi",
});

const CLI_MARKS = Object.freeze({
  "claude-code": "Cl",
  codex: "Cx",
  grok: "G",
  pi: "π",
});

export function normalizeTerminalCli(id) {
  return CLI_ALIASES[String(id || "").trim().toLowerCase()] || "";
}

// `tui` is the authoritative runtime presence. The top-level cli/state pair
// remains a compatibility projection for older servers and sessions.
export function terminalCli(term) {
  const runtime = term && term.tui;
  // The mere presence of tui is authoritative: never revive a legacy CLI
  // projection after the current runtime says there is no supported CLI.
  return runtime ? normalizeTerminalCli(runtime.cli) : normalizeTerminalCli(term && term.cli);
}

export function terminalCliLabel(id) {
  return CLI_LABELS[normalizeTerminalCli(id)] || "Terminal";
}

export function terminalCliMark(id) {
  return CLI_MARKS[normalizeTerminalCli(id)] || ">_";
}

export function terminalStatus(term) {
  // A live-list response explicitly says false; do not let a delayed
  // ephemeral state or runtime event make a gone tmux session look alive.
  if (term && term.running === false) return "stopped";
  const runtime = term && term.tui;
  const stateMatchesRuntime = !runtime || !term.runId || !runtime.runId || term.runId === runtime.runId;
  const state = stateMatchesRuntime ? String((term && term.state) || "") : "";
  if (state === "needs-you") return "needs-you";
  if (state === "working") return "working";
  if (terminalCli(term)) return state === "idle" ? "ready" : "open";
  return "open";
}

export function terminalStatusLabel(term) {
  const status = terminalStatus(term);
  if (status === "needs-you") return "Needs you";
  if (status === "working") return "Working";
  if (status === "ready") return "Ready";
  if (status === "open") return terminalCli(term) ? "Open" : "Terminal open";
  return "Stopped";
}

export function terminalActivityStamp(term) {
  const runtime = term && term.tui;
  const stateMatchesRuntime = !runtime || !term.runId || !runtime.runId || term.runId === runtime.runId;
  return (stateMatchesRuntime && term && term.stateAt) || (runtime && runtime.startedAt) || "";
}
