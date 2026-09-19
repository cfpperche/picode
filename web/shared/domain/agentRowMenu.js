// The rows of the "…" menu on an agent row — one menu the desktop sidebar
// renders. Pure on purpose, in the shape of termRowMenu.js: what the menu
// offers follows what the agent can actually do, the component stays a
// renderer, and this module is the contract that keeps the sidebar and the
// node tests from drifting apart.
//
// A Pi agent keeps the managed lifecycle: Start/Stop agent and Open chat.
// A guest (ADR-0160 — the CLI is the agent) has no managed runtime: its
// process is the bound terminal, so the lifecycle rows act on that
// terminal — Start / Restart / Stop terminal — and Launch settings opens
// the same screens the Agent CLIs hub offers for the terminal
// (`#/clis/terminal/<id>`). A guest has no chat panel: the row drops it,
// not greys it. Remove stays dangerous; the calling surface owns its
// confirm dialog.
//
// Icons are the renderer's business, so this module stays importable by
// the node test runner.

import { agentIsPi } from "./managedPrincipal.js";
import { terminalStatus } from "./terminalCli.js";

export function agentRowMenu(ag = {}, { clis, term } = {}) {
  if (agentIsPi(ag)) {
    const running = ((ag && ag.mode) || "stopped") !== "stopped";
    return [
      running
        ? { id: "stop", label: "Stop agent", title: "Stop the managed run." }
        : { id: "start", label: "Start agent", title: "Start a managed run." },
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
  return [
    ...(running
      ? [
          { id: "restart", label: "Restart terminal", title: "Stop and relaunch this CLI in the same conversation." },
          { id: "stop", label: "Stop terminal", title: "End the processes in this terminal; the launch stays saved." },
        ]
      : [{ id: "start", label: "Start terminal", title: "Launch this agent's CLI in its terminal." }]),
    ...(launch
      ? [{ id: "launch", label: "Launch settings", title: "CLI, profile and environment this agent launches with." }]
      : []),
    { id: "term", label: "Open terminal", title: "Open this agent's terminal pane." },
    { id: "rename", label: "Rename", title: "Change this agent's name." },
    { id: "remove", label: "Remove agent", title: "Delete this agent and its terminal.", danger: true },
  ];
}
