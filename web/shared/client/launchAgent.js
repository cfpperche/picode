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
  const body = { cli, name: name || "", overrides: overrides || {} };
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
