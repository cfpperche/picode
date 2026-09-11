// Native settings capabilities are independent of launch/install capabilities.
// Each CLI keeps its own editor, schema, and native persistence.
export const CLI_SETTINGS = [{ id: "pi", name: "Pi" }];
export const supportsCliSettings = id => CLI_SETTINGS.some(cli => cli.id === id);

export function cliSettingsHash(cli = "pi", { agentId = "", focus = "" } = {}) {
  const query = new URLSearchParams();
  if (agentId) query.set("agentId", agentId);
  if (focus === "scoped-models") query.set("focus", focus);
  return "#/clis/" + encodeURIComponent(cli || "pi") + "/settings" + (query.size ? "?" + query : "");
}

export function cliSettingsLocation(hash = "", legacyAgentId = "") {
  const [path, query = ""] = hash.replace(/^#/, "").split("?");
  const nested = /^\/clis\/([^/]+)\/settings$/.exec(path);
  const strip = path === "/clis/settings" || path.startsWith("/clis/settings/");
  const legacy = path === "/settings" || path === "/more/settings";
  if (!legacy && !strip && !nested) return null;
  const params = new URLSearchParams(query);
  let cli = "pi";
  try {
    if (nested) cli = decodeURIComponent(nested[1]);
    else if (strip && path.startsWith("/clis/settings/")) cli = decodeURIComponent(path.slice("/clis/settings/".length));
  } catch { cli = ""; }
  const agentId = params.has("agentId") ? params.get("agentId") : legacy ? legacyAgentId : "";
  const focus = params.get("focus") === "scoped-models" ? "scoped-models" : "";
  const canonical = cliSettingsHash(cli, { agentId, focus });
  return { view: "clis", pane: "settings", id: cli, agentId, focus, legacy: legacy || strip,
    redirect: !nested || path === "/clis/settings" ? canonical : "" };
}

function unavailable(message) {
  return Object.assign(new Error(message), { status: 404 });
}

// Pi's report validates the explicit identity before any workspace lookup.
export async function loadPiSettingsContext(agentId, request) {
  let report;
  try { report = await request("/api/pi-settings?agentId=" + encodeURIComponent(agentId)); }
  catch (error) {
    if (error.status === 404) throw unavailable("This agent is no longer available.");
    throw error;
  }
  if (!report.agent || report.agent.id !== agentId) throw unavailable("This agent is no longer available.");
  const free = !report.agent.workspaceId || report.agent.workspaceId === "ws_free";
  const rows = await request(free ? "/api/agents?free=1" : "/api/workspaces");
  const workspace = free ? null : rows.find(row => row.id === report.agent.workspaceId);
  if (!free && !workspace) throw unavailable("This workspace is no longer available.");
  const agent = (free ? rows : workspace.agents || []).find(row => row.id === agentId);
  if (!agent) throw unavailable("This agent is no longer available.");
  return { agent, workspace };
}
