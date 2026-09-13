// The rows of the "…" menu on an Agent CLI terminal row — one menu the
// sidebar row (WorkspaceRows) and the Agent CLIs list row (AgentClis) both
// render. Pure on purpose, in the shape of termMenu.js: what the menu
// offers follows what the terminal can actually do, the components stay
// renderers, and this module is the contract that keeps the two surfaces
// from drifting apart again.
//
// A row that cannot act is dropped, not greyed: a stopped terminal gets
// Start, a running one gets Restart and Stop — the same states the Agent
// CLIs list's own buttons answer to. Continue in… appears only when the
// pin names a conversation another CLI can receive (ADR-0088). Remove
// stays dangerous; the calling surface owns its confirm dialog. Icons
// are the renderer's business, so this module stays importable by the
// node test runner.

import { terminalHandoffMenu } from "@picode/shared/domain/sessionHandoff.js";

export function termRowMenu(t = {}, { clis } = {}) {
  const running = !!t.running;
  const lifecycle = running
    ? [
        { id: "restart", label: "Restart terminal", title: "Stop and relaunch with the saved settings." },
        { id: "stop", label: "Stop terminal", title: "End the processes in this terminal; the launch stays saved." },
      ]
    : [
        { id: "start", label: "Start terminal", title: "Launch this terminal with its saved settings." },
      ];
  const handoff = terminalHandoffMenu(t, clis);
  return [
    { id: "rename", label: "Rename…", title: "Change this terminal's name." },
    { id: "launch", label: "Launch settings", title: "CLI, profile and environment this terminal launches with." },
    { id: "settings", label: "Terminal settings", title: "Appearance and behavior of this terminal." },
    { sep: true },
    ...(handoff ? [handoff, { sep: true }] : []),
    ...lifecycle,
    { sep: true },
    { id: "remove", label: "Remove terminal", title: "Stop the terminal and delete it with its launch settings.", danger: true },
  ];
}
