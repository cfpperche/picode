// The rows of the "…" menu on an agent row — one menu the desktop sidebar
// renders. Pure on purpose, in the shape of termRowMenu.js: what the menu
// offers follows what the agent can actually do, the component stays a
// renderer, and this module is the contract that keeps the sidebar and the
// node tests from drifting apart.
//
// Lifecycle labels describe the agent (ADR-0160), whatever runs it.
// Pi reads its run mode; other CLIs read their bound terminal. Settings
// use the existing scoped Pi editor or the bound terminal launch editor.
// Only Pi offers chat. Remove stays last, separated and dangerous.
//
// Continue in… (ADR-0088): a pinned conversation — pi's own session pin,
// or the CLI agent's bound-terminal pin — can continue in another CLI. The
// submenu rides first, separated off; with no pin the row stays flat.
//
// Icons are the renderer's business, so this module stays importable by
// the node test runner.

import { agentIsPi } from "./managedPrincipal.js";
import { terminalHandoffMenu } from "./sessionHandoff.js";
import { terminalStatus } from "./terminalCli.js";

// agentHandoffTerm is the shim the renderer and this menu hand the handoff
// flow for a Pi agent: its session pin rides as a terminal's lastSession
// would. Null when the agent has no pinned session.
export function agentHandoffTerm(ag = {}) {
  const path = String((ag && ag.sessionPath) || "").trim();
  if (!path) return null;
  return { lastSession: { cli: "pi", path, cwd: String((ag && ag.workPath) || "").trim() } };
}

export function agentRowMenu(ag = {}, { clis, term } = {}) {
  if (agentIsPi(ag)) {
    const running = ((ag && ag.mode) || "stopped") !== "stopped";
    const handoffTerm = agentHandoffTerm(ag);
    const handoff = handoffTerm ? terminalHandoffMenu(handoffTerm, clis) : null;
    return [
      ...(handoff ? [handoff, { sep: true }] : []),
      running
        ? { id: "restart", label: "Restart agent", title: "Restart this agent in its current mode." }
        : { id: "start", label: "Start agent", title: "Start this agent." },
      ...(running ? [{ id: "stop", label: "Stop agent", title: "Stop this agent's current run." }] : []),
      { id: "launch", label: "Launch settings", title: "Configure this agent's settings.", href: "#/clis/pi/settings?agentId=" + encodeURIComponent(ag.id) },
      ...(ag.terminalId ? [{ id: "terminal-settings", label: "Terminal settings", title: "Configure this agent's terminal launch.", href: "#/clis/terminal/" + encodeURIComponent(ag.terminalId) }] : []),
      { id: "chat", label: "Open chat", title: "Open this agent's chat panel." },
      { id: "term", label: "Open terminal", title: "Open this agent's terminal pane." },
      { id: "rename", label: "Rename", title: "Change this agent's name." },
      { id: "remove", label: "Remove agent", title: "Delete this agent.", danger: true },
    ];
  }
  const running = !!term && terminalStatus(term) !== "stopped";
  const cli = (clis || []).find((c) => c && ag.cli && c.id === ag.cli);
  // A CLI with no adapter has no launch settings to edit (Muse Code,
  // Antigravity): the row drops that item rather than opening an empty form.
  const launch = !!ag.terminalId && (!cli || cli.integrationCapable !== false);
  // A conversation pinned on the bound terminal can continue in another
  // CLI (ADR-0088) — the same submenu the terminal row offers.
  const handoff = term ? terminalHandoffMenu(term, clis) : null;
  return [
    ...(handoff ? [handoff, { sep: true }] : []),
    ...(running
      ? [
          { id: "restart", label: "Restart agent", title: "Stop and relaunch this agent." },
          { id: "stop", label: "Stop agent", title: "Stop this agent's current run; its launch stays saved." },
        ]
      : [{ id: "start", label: "Start agent", title: "Launch this agent in its terminal." }]),
    ...(launch
      ? [{ id: "launch", label: "Launch settings", title: "CLI, profile and environment this agent launches with.", href: "#/clis/terminal/" + encodeURIComponent(ag.terminalId) }]
      : []),
    { id: "term", label: "Open terminal", title: "Open this agent's terminal pane." },
    { id: "rename", label: "Rename", title: "Change this agent's name." },
    { id: "remove", label: "Remove agent", title: "Delete this agent and its terminal.", danger: true },
  ];
}
