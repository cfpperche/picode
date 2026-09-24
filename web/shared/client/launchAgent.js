import { api } from "./api.js";

// ADR-0184: every user-facing CLI launch — New agent from Agent CLIs, a
// launch profile, resuming a session — creates an agent whose terminal
// carries the launch, then starts it. One door for both apps.
//   { cli, name, workspaceId, folder, overrides }
// workspaceId empty (or the free sentinel) makes a free agent; folder is
// where the CLI starts (empty = the workspace's, or a private one).
// Resolves { agent, terminalId, launchError }; rejects only when no agent
// was created.
export async function createLaunchAgent({ cli, name, workspaceId, folder, overrides }, request = api) {
  const free = !workspaceId || workspaceId === "ws_free";
  const body = { cli, name: name || "" };
  // A Pi agent has chat and terminal both; with nothing to change in its
  // launch it is the same agent the palette's New agent makes, and it gets
  // its terminal when one is opened. Any other CLI runs in its terminal
  // from the start, so its launch always travels.
  const plain = !overrides || Object.keys(overrides).length === 0;
  if (!(cli === "pi" && plain)) body.overrides = overrides || {};
  if (folder) body[free ? "path" : "workPath"] = folder;
  const url = free ? "/api/agents" : "/api/workspaces/" + encodeURIComponent(workspaceId) + "/agents";
  const agent = await request(url, post(body));
  const terminalId = (agent && agent.terminalId) || "";
  let launchError = "";
  if (terminalId) {
    try {
      await request("/api/terminals/" + encodeURIComponent(terminalId) + "/launch/start", post({ confirm: false }));
    } catch (e) {
      launchError = (e && e.message) || String(e);
    }
  }
  return { agent, terminalId, launchError };
}

function post(body) {
  return { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) };
}

// ADR-0205/0211: bring a removed agent back from its exit and start it, the
// same way from the history page and from the removal toast's Undo. A CLI
// agent's terminal starts (resuming its pinned conversation when there is
// one); a Pi agent without a terminal starts its managed run, which is what
// shows its conversation. Resolves { agent, terminalId, envKeys, startError };
// rejects only when nothing came back.
export async function restoreAgent(exitId, { workspaceId, undo } = {}, request = api) {
  const res = await request("/api/agent-history/" + encodeURIComponent(exitId) + "/restore", post({ workspaceId: workspaceId || "", undo: !!undo }));
  const agent = res.agent || null;
  const terminalId = res.terminalId || "";
  let startError = "";
  try {
    if (terminalId) {
      await request("/api/terminals/" + encodeURIComponent(terminalId) + "/launch/start", post({ confirm: false, resume: !!res.resume }));
    } else if (agent) {
      await request("/api/agents/" + encodeURIComponent(agent.id) + "/managed/start", { method: "POST" });
    }
  } catch (e) {
    startError = (e && e.message) || String(e);
  }
  return { agent, terminalId, envKeys: res.envKeys || [], startError };
}
