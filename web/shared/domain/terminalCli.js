const CLI_ALIASES = Object.freeze({
  claude: "claude-code",
  "claude-code": "claude-code",
  codex: "codex",
  grok: "grok",
  hermes: "hermes",
  opencode: "opencode",
  pi: "pi",
  muse: "muse",
  agy: "agy",
  omp: "omp",
});

const CLI_LABELS = Object.freeze({
  "claude-code": "Claude Code",
  codex: "Codex",
  grok: "Grok",
  hermes: "Hermes Agent",
  opencode: "OpenCode",
  pi: "Pi",
  muse: "Muse Code",
  agy: "Antigravity",
  omp: "Omp",
});

const CLI_MARKS = Object.freeze({
  "claude-code": "Cl",
  codex: "Cx",
  grok: "G",
  hermes: "H",
  opencode: "Oc",
  pi: "π",
  muse: "Mu",
  agy: "Ag",
  omp: "Om",
});

// Keep runtime identity on the vendor's own mark rather than a home-made
// glyph. Preference-ordered official assets; the badge walks the list when
// one fails to load. The first links are the same transparent SVG marks the
// provider faces use, so a loaded favicon never paints its own background
// card (vendor .ico/.png files are opaque white).
const CLI_ICON_BASE = "https://unpkg.com/@lobehub/icons-static-svg@1.73.0/icons/";

const CLI_FAVICONS = Object.freeze({
  "claude-code": Object.freeze([
    CLI_ICON_BASE + "claude.svg",
    // Anthropic's own raster icons as fallbacks — claude.ai's favicon hangs
    // or 403s for browser subresources.
    "https://www.anthropic.com/images/icons/favicon-32x32.png",
    "https://claude.ai/favicon.ico",
  ]),
  codex: Object.freeze([
    CLI_ICON_BASE + "openai.svg",
    "https://openai.com/favicon.ico",
    // ChatGPT's official static CDN touch icon; openai.com challenges some
    // browsers, and this hash may rotate when OpenAI rebuilds the site.
    "https://cdn.oaistatic.com/assets/apple-touch-icon-mz9nytnj.webp",
  ]),
  grok: Object.freeze([
    CLI_ICON_BASE + "grok.svg",
    "https://grok.com/images/favicon.svg",
  ]),
  hermes: Object.freeze([
    // hermes-agent.svg is not in the 1.73.0 pin the other marks use.
    "https://unpkg.com/@lobehub/icons-static-svg@1.95.0/icons/hermes-agent.svg",
    CLI_ICON_BASE + "nousresearch.svg",
    "https://hermes-agent.nousresearch.com/favicon.ico",
  ]),
  opencode: Object.freeze([
    // opencode.svg is not in the 1.73.0 pin the other marks use.
    "https://unpkg.com/@lobehub/icons-static-svg@1.95.0/icons/opencode.svg",
    "https://opencode.ai/favicon.svg",
    "https://opencode.ai/favicon.ico",
  ]),
  // Muse Code is Meta's product; its mark is the vendor's, like Codex
  // wearing OpenAI's.
  muse: Object.freeze([
    CLI_ICON_BASE + "meta.svg",
    "https://www.meta.com/favicon.ico",
  ]),
  agy: Object.freeze([
    // antigravity.svg is not in the 1.73.0 pin the other marks use.
    "https://unpkg.com/@lobehub/icons-static-svg@1.95.0/icons/antigravity.svg",
    "https://antigravity.google/favicon.ico",
  ]),
  // Omp ships no transparent glyph: its own mark is a rounded dark tile
  // that reads as an app icon on the dark UI. The vendor's SVG first, its
  // raster fallbacks after.
  omp: Object.freeze([
    "https://omp.sh/favicon.svg",
    "https://omp.sh/favicon.ico",
  ]),
  // pi has no lobehub mark; pi.dev serves a transparent SVG.
  pi: Object.freeze(["https://pi.dev/favicon.svg"]),
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

// Display identity for a row, tab or face: the runtime CLI when the adapter
// reported one, else the CLI the terminal was launched with. A CLI with no
// adapter (Muse Code, Antigravity) never reports a runtime, so its terminal
// would otherwise wear the plain-shell icon and be called "Shell session".
// A present `tui` stays authoritative even with an empty cli — a Pi terminal
// whose CLI exited is a shell again, not a Pi terminal.
export function terminalDisplayCli(term) {
  if (term && term.tui) return terminalCli(term);
  return normalizeTerminalCli(term && term.cli) || normalizeTerminalCli(term && term.launchCli);
}

export function terminalCliFaviconUrls(id) {
  return CLI_FAVICONS[normalizeTerminalCli(id)] || [];
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
  // "Open" is the CLI's terminal with no activity to report (a CLI without
  // an adapter never reports); "Terminal open" is a plain shell. Both are
  // the same status, and only the identity tells them apart.
  if (status === "open") return terminalDisplayCli(term) ? "Open" : "Terminal open";
  return "Stopped";
}

export function terminalActivityStamp(term) {
  const runtime = term && term.tui;
  const stateMatchesRuntime = !runtime || !term.runId || !runtime.runId || term.runId === runtime.runId;
  return (stateMatchesRuntime && term && term.stateAt) || (runtime && runtime.startedAt) || "";
}
