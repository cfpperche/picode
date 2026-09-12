// Native settings capabilities are independent of launch/install capabilities.
// Each CLI keeps its own editor, schema, and native persistence.
export const CLI_SETTINGS = [{ id: "pi", name: "Pi" }];
export const supportsCliSettings = id => CLI_SETTINGS.some(cli => cli.id === id);

// The pane edits one layer at a time and keeps the keyboard map on its own
// sub-tab (docs/plans/cli-settings-ux.md). Both travel on the route so a
// reload, a bookmark or a Palette command lands on the same view. The sub-tab
// is `tab`, not `view` — `view` already means the page in a route.
export const SETTINGS_LAYERS = ["global", "project", "agent"];
export const SETTINGS_TABS = ["settings", "keys"];

export function cliSettingsHash(cli = "pi", { agentId = "", focus = "", layer = "", tab = "" } = {}) {
  const query = new URLSearchParams();
  if (agentId) query.set("agentId", agentId);
  if (focus === "scoped-models") query.set("focus", focus);
  if (SETTINGS_LAYERS.includes(layer)) query.set("layer", layer);
  if (SETTINGS_TABS.includes(tab) && tab !== "settings") query.set("tab", tab);
  return "#/clis/" + encodeURIComponent(cli || "pi") + "/settings" + (query.size ? "?" + query : "");
}

export function cliSettingsLocation(hash = "", legacyAgentId = "") {
  const [path, query = ""] = hash.replace(/^#/, "").split("?");
  const extra = /^\/clis\/([^/]+)\/settings\//.exec(path);
  const nested = /^\/clis\/([^/]+)\/settings$/.exec(path);
  const strip = path === "/clis/settings" || path.startsWith("/clis/settings/");
  const legacy = path === "/settings" || path === "/more/settings";
  if (!legacy && !strip && !nested && !extra) return null;
  const params = new URLSearchParams(query);
  let cli = "pi";
  try {
    if (extra) cli = decodeURIComponent(extra[1]);
    else if (nested) cli = decodeURIComponent(nested[1]);
    else if (strip && path.startsWith("/clis/settings/")) cli = decodeURIComponent(path.slice("/clis/settings/".length));
  } catch { cli = ""; }
  if (extra) return { view: "clis", pane: "settings", id: cli, agentId: "", focus: "", invalid: true, legacy: false, redirect: "" };
  const adoptPane = !!(legacy && !params.has("agentId"));
  const agentId = params.has("agentId") ? params.get("agentId") : adoptPane ? legacyAgentId : "";
  const focus = params.get("focus") === "scoped-models" ? "scoped-models" : "";
  // An unknown layer or view is dropped, not adopted: the pane falls back to
  // the default layer and never writes to a layer the URL only guessed at.
  const layer = SETTINGS_LAYERS.includes(params.get("layer")) ? params.get("layer") : "";
  const tab = SETTINGS_TABS.includes(params.get("tab")) ? params.get("tab") : "";
  const canonical = cliSettingsHash(cli, { agentId, focus, layer, tab });
  return { view: "clis", pane: "settings", id: cli, agentId, focus, layer, tab, legacy: legacy || strip,
    ...(adoptPane ? { adoptPane: true } : {}),
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
