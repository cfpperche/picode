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

// pi.dev's own favicon.svg art — the same three paths and the official
// light ink (#111111). Inlined because the served SVG recolours itself from
// the OS scheme (prefers-color-scheme), not the app's theme; the dark theme
// recolours it instead with the shared term-cli-face invert, which lands
// within a step of pi.dev's own dark ink (#eee vs #f6f6f6).
const PI_MARK = "data:image/svg+xml," + encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 560 560"><g fill="#111111">'
  + '<path d="M420 280H280V140H0V0H420V280Z"/><path d="M560 560H420V280H560V560Z"/>'
  + '<path d="M140 560H0V140H140V280H280V420H140V560Z"/></g></svg>',
);

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
  // pi has no lobehub mark and pi.dev's own favicon colours itself from the
  // OS scheme (prefers-color-scheme), not the app's theme: a light OS drew a
  // near-black mark on the dark app, a dark OS a near-white block on the
  // light one. pi.dev's art inlined at its official light ink; the dark
  // theme recolours it with the term-cli-face invert like every other
  // monochrome mark (both apps' styles/app.css).
  pi: Object.freeze([PI_MARK]),
});

export function normalizeTerminalCli(id) {
  return CLI_ALIASES[String(id || "").trim().toLowerCase()] || "";
}

// `tui` is the authoritative runtime presence. Without it, the top-level cli
// the server copies from the last hook report (term_state.go) answers.
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

// terminalIsIdleShellAt: a plain shell, sitting in root, doing nothing —
// the only kind of terminal a git command may be typed into (ADR-0096).
// A terminal launched for a CLI carries launchCli even while its TUI is
// idle and reports no runtime, so it never counts: its composer would eat
// the keystrokes, and the server refuses it anyway ("This is an Agent CLI").
// Every CLI launch is an agent (ADR-0184), so a terminal an agent owns is
// never a plain shell either: agentTerms is that set of terminal ids, and it
// holds even when a feed patch left the row without its live fields.
export function terminalIsIdleShellAt(term, root, agentTerms) {
  if (!term || !root || term.cwd !== root) return false;
  if (agentTerms && agentTerms.has(term.id)) return false;
  return !term.tui && !term.cli && !term.launchCli && !term.state;
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
  if (state === "compacting") return "compacting";
  if (state === "working") return "working";
  // A shell command the CLI runs ("!make deploy") fires no hook in most
  // CLIs; the server sees it in the process tree (ADR-0212). It counts as
  // work without ever being a turn.
  if (terminalCommand(term)) return "working";
  if (terminalCli(term)) return state === "idle" ? "ready" : "open";
  return "open";
}

// terminalCommand -> { name, since } of the shell command the terminal's CLI
// is running, or null. Observed beside the hook state, never instead of it.
export function terminalCommand(term) {
  const cmd = term && term.command;
  return cmd && cmd.name ? cmd : null;
}

// terminalCommandTitle: the pill's tooltip naming the command, "" when none.
export function terminalCommandTitle(term) {
  const cmd = terminalCommand(term);
  return cmd ? "Running " + cmd.name : "";
}

export function terminalStatusLabel(term) {
  const status = terminalStatus(term);
  if (status === "needs-you") return "Needs you";
  if (status === "working") return String(term.state || "") !== "working" && terminalCommand(term) ? "Running" : "Working";
  if (status === "compacting") return "Compacting";
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
  const cmd = terminalCommand(term);
  if (cmd && cmd.since && terminalStatusLabel(term) === "Running") return cmd.since;
  return (stateMatchesRuntime && term && term.stateAt) || (runtime && runtime.startedAt) || "";
}
