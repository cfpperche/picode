// The PiCode right-click menu for a terminal pane.
// Study: docs/benchmarks/2026-09-07-terminal-context-menu.md.
//
// Pure on purpose: what the menu offers is decided from what this pane can
// actually do, so the rows are testable without a browser and the component
// stays a renderer. A row that cannot act is dropped, not greyed — the two
// exceptions are Copy and Paste, which every terminal keeps in place (VS
// Code, Ghostty, Windows Terminal) so the muscle memory never misses. Copy
// then carries the reason it is idle, as the app's generic menu already does.
//
// Icons are names, not components: this module must stay importable by the
// node test runner.

import { focusRowLabel } from "@picode/shared/domain/focusMode.js";
import { agentPin, terminalHandoffMenu } from "@picode/shared/domain/sessionHandoff.js";
import { terminalCli, terminalCliLabel } from "@picode/shared/domain/terminalCli.js";

const SEP = { sep: true };

// Shortcuts are the ones the terminal really binds (web/shared/domain/
// termKeys.js and TermSurface's own handler). Never advertise a chord the
// app does not answer — a menu that lies about its keys is worse than one
// that shows none.
export const TERM_MENU_KEYS = {
  copy: "Ctrl+Shift+C",
  find: "Ctrl+Shift+F",
  paste: "Ctrl+Shift+V",
  "text-bigger": "Ctrl+=",
  "text-smaller": "Ctrl+-",
  "text-reset": "Ctrl+0",
  "close-tab": "Alt+W",
};

// paneCapabilities turns a live .term-pane + fleet row into the bag
// buildTermMenu reads. host is the dispatch address (dataset.termKind);
// promptDoor is the HTTP door, never a fake cli/running on an agent pane.
export function paneCapabilities({
  host, termRecord, agent, workspace, paneCwd, tabs, clis,
  selection, link, focus, focusable, focusKey, findKey, splitOn,
} = {}) {
  const interactive = host === "agent" && agent && agent.mode === "interactive";
  const cliRunning = host === "term" && termRecord && termRecord.launchCli && termRecord.running;
  const found = host === "term" ? !!termRecord : !!agent;
  const tabId = host === "agent" ? (agent && agent.id) : (termRecord && ("t:" + termRecord.id));
  return {
    host,
    kind: host, // live handlers still read ctx.kind (open-link, open-browser)
    id: host === "agent" ? (agent && agent.id) : (termRecord && termRecord.id),
    selection, link, focus, focusable, focusKey, findKey, splitOn, clis,
    promptDoor: cliRunning ? "cli" : interactive ? "tui" : null,
    cliLabel: cliRunning ? terminalCliLabel(termRecord.launchCli) : (interactive ? "Pi" : ""),
    running: !!(cliRunning || interactive),
    shell: host === "term" && !!termRecord && !termRecord.launchCli && !terminalCli(termRecord),
    lifecycle: found ? host : null,
    filesOwner: found ? host : null,
    settings: !found ? null : (host === "term" ? "per-terminal" : "global"),
    tabOpen: !!(tabId && (tabs || []).includes(tabId)),
    record: host === "term" ? termRecord : agentPin(agent, workspace, paneCwd),
  };
}

// ctx is either a paneCapabilities bag or the older {kind, cli, running, …}
// shape the tests still pass. promptDoor / lifecycle win when present.
export function buildTermMenu(ctx = {}) {
  const selection = (ctx.selection || "").trim();
  const host = ctx.host || ctx.kind || "term";
  const promptDoor = ctx.promptDoor !== undefined
    ? ctx.promptDoor
    : (host === "agent" ? null : ((ctx.cli && ctx.running) ? "cli" : null));
  const cliLabel = ctx.cliLabel || ctx.cli || "";
  const lifecycle = ctx.lifecycle !== undefined
    ? ctx.lifecycle
    : (host === "agent" ? null : "term");
  const settings = ctx.settings !== undefined
    ? ctx.settings
    : (lifecycle === "term" ? "per-terminal" : lifecycle === "agent" ? "global" : null);
  const filesOwner = ctx.filesOwner !== undefined ? ctx.filesOwner : lifecycle;
  const tabOpen = ctx.tabOpen !== undefined ? !!ctx.tabOpen : !!lifecycle;
  const noun = lifecycle === "agent" ? "agent" : "terminal";
  const rows = [
    { id: "copy", label: "Copy", icon: "copy", key: TERM_MENU_KEYS.copy, disabled: !selection, reason: "Select text in the terminal first." },
    { id: "paste", label: "Paste", icon: "paste", key: TERM_MENU_KEYS.paste },
    { id: "select-all", label: "Select all", icon: "select" },
  ];

  // The section that makes this menu PiCode's rather than an emulator's:
  // the pane's own output becomes the agent's context (Cursor's "Add to
  // Chat", Warp's block copy), and the token under the cursor opens where
  // Ctrl+click already opens it.
  const send = [];
  if (promptDoor) {
    if (selection) send.push({ id: "ask", label: "Ask " + (cliLabel || "Pi") + " about this", icon: "ask" });
    send.push({ id: "attach", label: "Attach files…", icon: "clip" });
    send.push({ id: "snippet", label: "Send to terminal…", icon: "file" });
  }
  // A command snippet runs in the pane's own shell (ADR-0130): the row
  // follows the same bare-shell flag Clear does, not the prompt door.
  if (ctx.shell && host !== "agent") send.push({ id: "snippet-cmd", label: "Run command…", icon: "file" });
  if (ctx.link) send.push({ id: "open-link", label: "Open " + ctx.link.label, icon: ctx.link.kind === "http" ? "external" : "file" });
  if (send.length) rows.push(SEP, ...send);

  const view = [
    { id: "find", label: "Find…", icon: "search", key: ctx.findKey || TERM_MENU_KEYS.find },
    { id: "scroll-end", label: "Go to the end", icon: "end" },
    {
      id: "text-size",
      label: "Text size",
      icon: "text",
      sub: [
        { id: "text-bigger", label: "Bigger", key: TERM_MENU_KEYS["text-bigger"] },
        { id: "text-smaller", label: "Smaller", key: TERM_MENU_KEYS["text-smaller"] },
        { id: "text-reset", label: "Reset", key: TERM_MENU_KEYS["text-reset"] },
      ],
    },
  ];
  // Clear is a shell courtesy, not a screen wipe: it sends the same Ctrl-L
  // the user would type. A TUI owns its screen — launched by us or started
  // by hand in a plain terminal — so the row stays away from it rather than
  // pretending to clear something it does not control.
  if (ctx.shell && host !== "agent") view.push({ id: "clear", label: "Clear", icon: "clear" });
  // Fullscreen belongs to the shell, not to this pane, but the owner asked
  // for it on every tab's menu — and a terminal pane never shows the
  // generic one, so this is the only place it can appear over a terminal.
  if (ctx.focusable !== false) {
    view.push({ id: "fullscreen", label: focusRowLabel(!!ctx.focus), icon: ctx.focus ? "collapse" : "expand", key: ctx.focusKey || "" });
  }
  rows.push(SEP, ...view);

  // The owner's split (ADR-0135): a pane that hosts an agent — an agent's
  // TUI, or a terminal launched with an agent CLI (pi, Claude Code, …) —
  // can host a work-browser pane beside it, bound to that agent. A bare
  // shell has no agent to bind, so the row — open or close, one toggle —
  // stays away from it.
  if (host === "agent" || !!cliLabel) {
    rows.push(SEP, ctx.splitOn
      ? { id: "close-browser", label: "Close browser split", icon: "x" }
      : { id: "open-browser", label: "Open browser", icon: "globe" });
  }

  if (lifecycle) {
    const ownRows = [];
    ownRows.push({ id: "rename", label: "Rename " + noun + "…", icon: "pencil" });
    if (settings) ownRows.push({ id: "settings", label: "Terminal settings", icon: "settings" });
    if (filesOwner) ownRows.push({ id: "files", label: "Open folder in Files", icon: "folders" });
    const handoff = terminalHandoffMenu(ctx.record || { lastSession: ctx.lastSession }, ctx.clis);
    if (handoff) ownRows.push(handoff);
    const closeHint = lifecycle === "agent" ? "The session keeps running." : "The terminal keeps running.";
    rows.push(SEP, ...ownRows, SEP);
    if (tabOpen) {
      rows.push({ id: "close-tab", label: "Close tab", icon: "x", key: TERM_MENU_KEYS["close-tab"], hint: closeHint });
    }
    rows.push({ id: "remove", label: "Remove " + noun + "…", icon: "trash", danger: true });
  }
  return rows;
}

// A selection handed to the CLI takes one of two shapes, because a prompt
// input is one line and terminal output rarely is. Anything that reads
// cleanly inline is pre-filled as the message; the rest is staged as a text
// attachment, so the CLI reads a whole file instead of a mangled paste
// (ADR-0089 already delivers files by path). Nothing is ever truncated.
export const ASK_INLINE_MAX = 400;

export function planAsk(selection) {
  const raw = String(selection || "");
  const trimmed = raw.trim();
  if (!trimmed) return { mode: "none" };
  if (!/[\r\n]/.test(trimmed) && trimmed.length <= ASK_INLINE_MAX) return { mode: "text", text: trimmed };
  return { mode: "file", name: "selection.txt", body: raw.replace(/\s+$/, "") + "\n" };
}
