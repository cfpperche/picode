// Headless contracts only; desktop and mobile own their presentation.
// A CLI is connector-capable only when a driver declares its capabilities
// here (ADR-0150). The pane renders what a driver declares and nothing more:
// status "configured" never claims live state, toggle "none" removes the
// switch entirely (Claude Code turns servers on/off in its own interface).
// signIn is the vendor's own sign-in path shown to guests: command renders
// as copyable code (a terminal command by default, or the TUI named in
// `where`); text renders as a plain sentence when no command exists.
export const CONNECTOR_DRIVERS = {
  pi: {
    id: "pi",
    name: "Pi",
    status: "live", // live | configured — what the pane may honestly claim
    auth: ["oauth", "bearer"],
    toggle: "entry", // per-entry enable/disable, not stub-overlay
  },
  // id matches the CLI catalog id so #/clis/<id>/connectors highlights the
  // right roster entry (AgentClis selected = find(c.id === route.id)).
  "claude-code": {
    id: "claude-code",
    name: "Claude Code",
    status: "live", // `claude mcp list` health-checks servers (✓/✘ tails)

    auth: ["oauth"],
    toggle: "none", // sign-in and on/off live inside Claude Code
    signIn: { command: "claude mcp login {name}" },
  },
  codex: {
    id: "codex",
    name: "Codex",
    status: "configured",
    auth: ["oauth", "bearer"],
    toggle: "entry", // [mcp_servers.<name>] enabled flag
    signIn: { command: "codex mcp login {name}" },
  },
  omp: {
    id: "omp",
    name: "Omp",
    status: "configured",
    auth: ["oauth"],
    toggle: "entry", // entry's enabled flag
    signIn: { command: "/mcp reauth {name}", where: "the Omp TUI" }, // OAuth is TUI-only
  },
  agy: {
    id: "agy",
    name: "Antigravity",
    status: "configured",
    auth: ["oauth"],
    toggle: "entry", // entry's disabled flag
    signIn: { text: "Authenticate in Antigravity (Agent Settings → Authenticate)" },
  },
  opencode: {
    id: "opencode",
    name: "OpenCode",
    status: "live", // `opencode mcp list` reports per-server status

    auth: ["oauth"],
    toggle: "entry", // entry's enabled flag in the opencode.json mcp block
    signIn: { command: "opencode mcp auth {name}" },
  },
  grok: {
    id: "grok",
    name: "Grok",
    status: "configured",
    auth: ["oauth"],
    toggle: "entry", // [mcp_servers.<name>] enabled flag + disabled_mcp_servers overlay
    signIn: { text: "Sign in happens on first use inside Grok" },
  },
  muse: {
    id: "muse",
    name: "Muse Code",
    status: "configured",
    auth: ["oauth"],
    toggle: "entry", // entry's enabled flag in settings.json mcp_servers
    signIn: { command: "muse mcp login {name}" },
  },
  hermes: {
    id: "hermes",
    name: "Hermes Agent",
    status: "live", // `hermes mcp list` reports per-server status

    auth: ["oauth"],
    toggle: "entry", // entry's enabled flag in config.yaml mcp_servers
    signIn: { command: "hermes mcp login {name}" },
  },
};
export const CLI_CONNECTORS = Object.values(CONNECTOR_DRIVERS);
export const connectorDriver = (id) => CONNECTOR_DRIVERS[id] || null;
export const supportsCliConnectors = (id) => !!connectorDriver(id);

export function cliConnectorsHash(cli = "pi", { workspaceId = "", agentId = "", scope = "user" } = {}) {
  const query = new URLSearchParams();
  if (workspaceId) query.set("workspaceId", workspaceId);
  if (agentId) query.set("agentId", agentId);
  if (scope && scope !== "user") query.set("scope", scope);
  return "#/clis/" + encodeURIComponent(cli || "pi") + "/connectors" + (query.size ? "?" + query : "");
}

export function cliConnectorsLocation(hash = "", legacyContext = {}) {
  const [path, query = ""] = hash.replace(/^#/, "").split("?");
  if (path === "/integrations/webhooks" || path === "/more/integrations/webhooks") return null;
  const extra = /^\/clis\/([^/]+)\/connectors\//.exec(path);
  const nested = /^\/clis\/([^/]+)\/connectors$/.exec(path);
  const strip = path === "/clis/connectors";
  const legacy = path === "/mcps" || path === "/more/mcps" || path === "/integrations" || path === "/integrations/connectors"
    || path === "/more/integrations" || path === "/more/integrations/connectors";
  if (!legacy && !strip && !nested && !extra) return null;
  const params = new URLSearchParams(query);
  let id = "pi";
  try {
    if (extra) id = decodeURIComponent(extra[1]);
    else if (nested) id = decodeURIComponent(nested[1]);
  } catch { id = ""; }
  if (extra) return { view: "clis", pane: "connectors", id, workspaceId: "", agentId: "", scope: "user", invalid: true, legacy: false, redirect: "" };
  const explicit = params.has("workspaceId") || params.has("agentId");
  const adoptPane = !!(legacy && !explicit);
  const fallback = adoptPane ? legacyContext : {};
  const workspaceId = params.get("workspaceId") || fallback.workspaceId || "";
  const agentId = params.get("agentId") || fallback.agentId || "";
  const scope = params.get("scope") || "user";
  const invalid = !["user", "project", "agent"].includes(scope);
  const canonical = cliConnectorsHash(id, { workspaceId, agentId, scope });
  return { view: "clis", pane: "connectors", id, workspaceId, agentId, scope, invalid, legacy: legacy || strip,
    ...(adoptPane ? { adoptPane: true } : {}),
    redirect: !invalid && !nested ? canonical : "" };
}

export function integrationSection(hash = "") {
  return hash.replace(/^#/, "").split("/").includes("webhooks") ? "webhooks" : "connectors";
}

export const webhookPresets = [
  { label: "Agents and inbox", types: "agent., inbox." },
  { label: "Tasks and automations", types: "task., automation." },
  { label: "Docker jobs", types: "docker.job, docker.operation" },
];

export function destinationLabel(raw) {
  try { const u = new URL(raw); return u.host + (u.pathname === "/" ? "" : u.pathname); }
  catch { return "Webhook"; }
}

// Config files PiCode could not read (ADR-0150): one line per blocked layer
// names the file instead of a pane-wide error. `file` names the document,
// `path` keeps the full location for the Open tooltip.
export function blockedLayers(data) {
  const layers = (data && Array.isArray(data.layers)) ? data.layers : [];
  return layers
    .filter((l) => l && l.error)
    .map((l) => ({ scope: l.scope || "user", error: l.error, path: l.path || "", file: fileName(l.path) }));
}

function fileName(p) {
  const s = String(p || "");
  const i = Math.max(s.lastIndexOf("/"), s.lastIndexOf("\\"));
  return i >= 0 ? s.slice(i + 1) : s;
}

// Marketplace (ADR-0157): a gallery hit becomes the same POST /api/mcp entry
// the preset rows wrote — name is the catalog id, transport fields follow the
// hit's kind, auth passes through. Entries only: credentials never ride the
// catalog.
export function connectorAddBody(hit) {
  const h = hit || {};
  const stdio = h.kind === "stdio";
  return {
    name: String(h.id || ""),
    url: stdio ? "" : String(h.url || ""),
    command: stdio ? String(h.command || "") : "",
    args: stdio && Array.isArray(h.args) ? h.args.map(String) : [],
    auth: String(h.auth || ""),
  };
}

// A catalog entry is "Added" when the selected layer already has it; another
// layer's copy is not this one (docs/plans/connectors-ux.md).
export function connectorScopeNames(servers, scope) {
  const list = Array.isArray(servers) ? servers : [];
  return new Set(list.filter((s) => s && s.scope === scope).map((s) => s.name));
}
