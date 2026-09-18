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
    status: "configured",
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
    status: "configured",
    auth: ["oauth"],
    toggle: "entry", // entry's enabled flag in the opencode.json mcp block
    signIn: { command: "opencode mcp auth {name}" },
  },
  grok: {
    id: "grok",
    name: "Grok",
    status: "configured",
    auth: ["oauth"],
    toggle: "none", // Grok's config has no per-server switch
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
    status: "configured",
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

// Import standard MCP configuration, not a second runtime/package format.
// Reviewing one server at a time makes command/token permissions explicit.
// Catalog tabs for the Add-connector card: the fixed catalog first, then one
// tab per agent CLI whose configuration actually contains MCP servers.
export function connectorTabs(found) {
  const hosts = (found || [])
    .filter((h) => h && h.kind && h.label && Array.isArray(h.servers) && h.servers.length)
    .map((h) => ({ id: h.kind, label: h.label, servers: h.servers }));
  hosts.sort((a, b) => a.label.localeCompare(b.label));
  return [{ id: "catalog", label: "Catalog", fixed: true }, ...hosts];
}

export function readConnectorDefinition(text) {
  if (text.length > 65536) throw new Error("Choose a connector file smaller than 64 KB.");
  let data;
  try { data = JSON.parse(text); } catch { throw new Error("Choose a JSON connector definition."); }
  const entries = Object.entries(data?.mcpServers || {});
  if (entries.length !== 1) throw new Error("Choose a definition containing one MCP server.");
  const [name, entry] = entries[0];
  if (!entry || typeof entry !== "object" || Array.isArray(entry) || !/^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/.test(name)) throw new Error("The connector needs a valid name and server configuration.");
  if (entry.disabled || entry.cwd || entry.socket || entry.transport || entry.type && !["http", "sse", "stdio"].includes(entry.type)) throw new Error("This definition has options that cannot be imported here. Use MCP configuration instead.");
  const allowed = ["command", "args", "url", "env", "headers", "auth", "bearerToken", "type", "disabled"];
  if (Object.keys(entry).some(key => !allowed.includes(key))) throw new Error("This definition has unsupported options. Use MCP configuration instead.");
  if (!!entry.url === !!entry.command) throw new Error("Choose either a server URL or a local command.");
  if (entry.url && (typeof entry.url !== "string" || !/^https?:\/\//.test(entry.url))) throw new Error("Use an HTTP or HTTPS server URL.");
  if (entry.command && typeof entry.command !== "string") throw new Error("Use a valid local command.");
  if (entry.args && (!Array.isArray(entry.args) || entry.args.some(arg => typeof arg !== "string"))) throw new Error("Arguments must be a list of strings.");
  for (const key of ["env", "headers"]) {
    if (entry[key] && (typeof entry[key] !== "object" || Array.isArray(entry[key]) || Object.values(entry[key]).some(v => typeof v !== "string"))) throw new Error("Connector settings must contain text values.");
  }
  if (entry.url && entry.env || entry.command && (entry.headers || entry.auth || entry.bearerToken)) throw new Error("Use environment variables for commands or headers for URLs.");
  if (entry.auth && !["oauth", "bearer"].includes(entry.auth) || entry.bearerToken && typeof entry.bearerToken !== "string") throw new Error("Use supported sign-in settings.");
  if (entry.url && [entry.bearerToken, ...Object.values(entry.headers || {})].some(v => typeof v === "string" && v.startsWith("!") && !v.startsWith("!!"))) throw new Error("Remote connector definitions cannot run credential commands. Configure those explicitly in MCP settings.");
  return { name, ...entry };
}
