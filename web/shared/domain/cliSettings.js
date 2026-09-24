// Native settings capabilities are independent of launch/install capabilities.
// Each CLI keeps its own editor, schema, and native persistence.
export const CLI_SETTINGS = [{ id: "pi", name: "Pi" }];
export const supportsCliSettings = id => CLI_SETTINGS.some(cli => cli.id === id);

// The pane edits one layer at a time (docs/plans/cli-settings-ux.md); the layer
// travels on the route so a reload, a bookmark or a Palette command lands on
// the same view. The keyboard map is a sibling pane — `#/clis/pi/keyboard` —
// not a sub-tab: two tab rows inside one pane read as nesting (owner,
// 2026-09-12), and the map is machine-wide, so it has no layer of its own.
export const SETTINGS_LAYERS = ["global", "project", "agent"];

// The query of a settings view. The keyboard pane builds its own path and
// carries the same context through this, so a round trip to the map comes back
// to the workspace, the agent and the layer it left (2026-09-12).
export function cliSettingsQuery({ workspaceId = "", agentId = "", focus = "", layer = "" } = {}) {
  const query = new URLSearchParams();
  if (workspaceId) query.set("workspaceId", workspaceId);
  if (agentId) query.set("agentId", agentId);
  if (focus === "scoped-models") query.set("focus", focus);
  if (SETTINGS_LAYERS.includes(layer)) query.set("layer", layer);
  return query.size ? "?" + query : "";
}

export function cliSettingsHash(cli = "pi", { workspaceId = "", agentId = "", focus = "", layer = "" } = {}) {
  return "#/clis/" + encodeURIComponent(cli || "pi") + "/settings" + cliSettingsQuery({ workspaceId, agentId, focus, layer });
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
  if (extra) return { view: "clis", pane: "settings", id: cli, workspaceId: "", agentId: "", focus: "", invalid: true, legacy: false, redirect: "" };
  const adoptPane = !!(legacy && !params.has("agentId"));
  const workspaceId = params.get("workspaceId") || "";
  const agentId = params.has("agentId") ? params.get("agentId") : adoptPane ? legacyAgentId : "";
  const focus = params.get("focus") === "scoped-models" ? "scoped-models" : "";
  // An unknown layer or view is dropped, not adopted: the pane falls back to
  // the default layer and never writes to a layer the URL only guessed at.
  const layer = SETTINGS_LAYERS.includes(params.get("layer")) ? params.get("layer") : "";
  // `?tab=keys` was the sub-tab of the day before: the caller turns this flag
  // into a redirect to the keyboard pane, so an old bookmark still lands there.
  const keysTab = params.get("tab") === "keys";
  const canonical = cliSettingsHash(cli, { workspaceId, agentId, focus, layer });
  return { view: "clis", pane: "settings", id: cli, workspaceId, agentId, focus, layer, keysTab, legacy: legacy || strip,
    ...(adoptPane ? { adoptPane: true } : {}),
    redirect: !nested || path === "/clis/settings" ? canonical : "" };
}

// The layers a route may name; the keyboard pane keeps the settings context
// (agent and layer) so returning to Settings lands where the reader left.
export function cliKeysLocation(hash = "") {
  const [path, query = ""] = hash.replace(/^#/, "").split("?");
  const m = /^\/clis\/([^/]+)\/keyboard$/.exec(path);
  if (!m) return null;
  const params = new URLSearchParams(query);
  let cli = "pi";
  try { cli = decodeURIComponent(m[1]); } catch { cli = ""; }
  const layer = SETTINGS_LAYERS.includes(params.get("layer")) ? params.get("layer") : "";
  return { view: "clis", pane: "keyboard", id: cli, workspaceId: params.get("workspaceId") || "", agentId: params.get("agentId") || "", layer, invalid: false };
}

function unavailable(message) {
  return Object.assign(new Error(message), { status: 404 });
}

// Explicit ids are looked up as given. A workspace-only link never infers an
// agent, and a missing id never becomes another workspace or agent.
export async function loadPiSettingsContext(agentId, request, workspaceId = "") {
  if (workspaceId) {
    const rows = await request("/api/workspaces");
    const named = (Array.isArray(rows) ? rows : []).find(row => row.id === workspaceId);
    if (!named) throw unavailable("This workspace is no longer available.");
    if (!agentId) return { agent: null, workspace: named.id === "ws_free" ? null : named };
  }
  if (!agentId) return { agent: null, workspace: null };
  let report;
  try { report = await request("/api/pi-settings?agentId=" + encodeURIComponent(agentId)); }
  catch (error) {
    if (error.status === 404) throw unavailable("This agent is no longer available.");
    throw error;
  }
  if (!report.agent || report.agent.id !== agentId) throw unavailable("This agent is no longer available.");
  const free = !report.agent.workspaceId || report.agent.workspaceId === "ws_free";
  if (workspaceId && (free || workspaceId !== report.agent.workspaceId)) throw unavailable("This agent does not belong to this workspace.");
  const rows = await request(free ? "/api/agents?free=1" : "/api/workspaces");
  const workspace = free ? null : rows.find(row => row.id === report.agent.workspaceId);
  if (!free && !workspace) throw unavailable("This workspace is no longer available.");
  const agent = (free ? rows : workspace.agents || []).find(row => row.id === agentId);
  if (!agent) throw unavailable("This agent is no longer available.");
  return { agent, workspace };
}
