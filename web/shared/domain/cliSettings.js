// Native settings capabilities are independent of launch/install capabilities.
// Each CLI keeps its own editor, schema, and native persistence.
export const CLI_SETTINGS = [{ id: "pi", name: "Pi" }];
export const supportsCliSettings = id => CLI_SETTINGS.some(cli => cli.id === id);

export function cliSettingsHash(cli = "pi", { agentId = "", focus = "" } = {}) {
  const query = new URLSearchParams();
  if (agentId) query.set("agentId", agentId);
  if (focus === "scoped-models") query.set("focus", focus);
  return "#/clis/settings/" + encodeURIComponent(cli) + (query.size ? "?" + query : "");
}

export function cliSettingsLocation(hash = "", legacyAgentId = "") {
  const [path, query = ""] = hash.replace(/^#/, "").split("?");
  const legacy = path === "/settings" || path === "/more/settings";
  if (!legacy && path !== "/clis/settings" && !path.startsWith("/clis/settings/")) return null;
  const params = new URLSearchParams(query);
  let cli = "pi";
  if (!legacy && path.startsWith("/clis/settings/")) {
    try { cli = decodeURIComponent(path.slice("/clis/settings/".length)); } catch { cli = ""; }
  }
  const agentId = params.has("agentId") ? params.get("agentId") : legacy ? legacyAgentId : "";
  const focus = params.get("focus") === "scoped-models" ? "scoped-models" : "";
  return { view: "settings", id: cli, agentId, focus, legacy,
    redirect: legacy || path === "/clis/settings" ? cliSettingsHash(cli, { agentId, focus }) : "" };
}

// Pi's report validates the explicit identity before any workspace lookup.
export async function loadPiSettingsContext(agentId, request) {
  let report;
  try { report = await request("/api/pi-settings?agentId=" + encodeURIComponent(agentId)); }
  catch (error) {
    if (error.status === 404) throw new Error("This agent is no longer available.");
    throw error;
  }
  if (!report.agent || report.agent.id !== agentId) throw new Error("This agent is no longer available.");
  const free = !report.agent.workspaceId || report.agent.workspaceId === "ws_free";
  const rows = await request(free ? "/api/agents?free=1" : "/api/workspaces");
  const workspace = free ? null : rows.find(row => row.id === report.agent.workspaceId);
  if (!free && !workspace) throw new Error("This workspace is no longer available.");
  const agent = (free ? rows : workspace.agents || []).find(row => row.id === agentId);
  if (!agent) throw new Error("This agent is no longer available.");
  return { agent, workspace };
}
