// Headless contracts only; desktop and mobile own their presentation.
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
