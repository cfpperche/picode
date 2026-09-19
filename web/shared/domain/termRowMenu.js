// The rows of the "…" menu on an Agent CLI terminal row — one menu the
// desktop sidebar, the Agent CLIs list, and the phone Work list all
// render. Pure on purpose, in the shape of termMenu.js: what the menu
// offers follows what the terminal can actually do, the components stay
// renderers, and this module is the contract that keeps the surfaces
// from drifting apart again.
//
// A row that cannot act is dropped, not greyed: a stopped terminal gets
// Start, a running one gets Restart and Stop. Continue in… appears only
// when the pin names a conversation another CLI can receive (ADR-0088).
// Remove stays dangerous; the calling surface owns its confirm dialog.
// Icons are the renderer's business, so this module stays importable by
// the node test runner.
//
// surface: "phone" hides Launch settings and Terminal settings — those
// are forms that live under Agent CLIs and inside the open pane.

import { terminalHandoffMenu } from "./sessionHandoff.js";

const PHONE_HIDDEN = new Set(["launch", "settings"]);

export function termRowMenu(t = {}, { clis, surface } = {}) {
  const running = !!t.running;
  // A CLI with no adapter has no launch settings to edit (Muse Code,
  // Antigravity): the row drops that item rather than opening an empty form.
  const cli = (clis || []).find((c) => c.id === (t.launchCli || t.cli));
  const launchSettings = !cli || cli.integrationCapable !== false;
  const lifecycle = running
    ? [
        { id: "restart", label: "Restart terminal", title: "Stop and relaunch this CLI in the same conversation." },
        { id: "stop", label: "Stop terminal", title: "End the processes in this terminal; the launch stays saved." },
      ]
    : [
        { id: "start", label: "Start terminal", title: "Launch this terminal with its saved settings." },
      ];
  const handoff = terminalHandoffMenu(t, clis);
  const rows = [
    { id: "rename", label: "Rename…", title: "Change this terminal's name." },
    ...(launchSettings ? [{ id: "launch", label: "Launch settings", title: "CLI, profile and environment this terminal launches with." }] : []),
    { id: "settings", label: "Terminal settings", title: "Appearance and behavior of this terminal." },
    { sep: true },
    ...(handoff ? [handoff, { sep: true }] : []),
    ...lifecycle,
    { sep: true },
    { id: "remove", label: "Remove terminal", title: "Stop the terminal and delete it with its launch settings.", danger: true },
  ];
  if (surface === "phone") {
    return compactMenu(rows.filter((r) => r.sep || !PHONE_HIDDEN.has(r.id)));
  }
  return rows;
}

function compactMenu(rows) {
  const out = [];
  for (const r of rows) {
    if (r.sep) {
      if (!out.length || out[out.length - 1].sep) continue;
      out.push(r);
      continue;
    }
    out.push(r);
  }
  if (out.length && out[out.length - 1].sep) out.pop();
  return out;
}
