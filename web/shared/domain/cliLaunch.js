import { cliProvidersLocation } from "./cliProviders.js";
import { cliPackagesLocation } from "./cliPackages.js";
import { cliKeysLocation, cliSettingsLocation } from "./cliSettings.js";
import { cliConnectorsLocation } from "./integrations.js";
import { supportsCliModels } from "./cliModels.js";

const CLI_PANES = new Set(["launch", "terminals", "sessions", "providers", "models", "settings", "keyboard", "memory", "packages", "connectors"]);

// cliCapabilities reads what the server says one catalog row can do.
// launch: New terminal exists. integration: activity, launch settings and
// the setup panes exist (an adapter is behind the CLI). sessions: a session
// source answers for it. Muse Code and Antigravity are launch without an
// adapter, so their surface carries no settings and no sessions.
export function cliCapabilities(cli) {
  if (!cli) return { launch: false, integration: false, sessions: false };
  return {
    launch: cli.launchable !== false,
    integration: cli.integrationCapable === true,
    sessions: !!(cli.sessions && cli.sessions.list),
  };
}

// cliPanes is the pane list a CLI's surface may show. A row with no
// integration keeps Launch and Terminals only: no Sessions (no source
// answers), and none of the setup panes, which all edit Pi-side things.
export function cliPanes(cli) {
  const cap = cliCapabilities(cli);
  // Sessions follows the session source the server reports, not the adapter:
  // Muse Code and Antigravity have history on disk before they have
  // activity reporting (their Sessions tab lists and resumes, nothing more).
  const sessions = cap.sessions ? ["sessions"] : [];
  // The setup tabs ride along for every CLI: the ones without a native
  // integration render them as in-development placeholders instead of
  // pretending the feature exists (owner, 2026-09-15).
  // Models only where PiCode can ask the CLI what it reaches (ADR-0181): a
  // pane that could only say "in development" for eight CLIs would be a
  // placeholder, and the setup group already carries those.
  const models = cli && supportsCliModels(cli.id) ? ["models"] : [];
  return ["launch", "terminals", ...sessions, "providers", ...models, "settings", "keyboard", "memory", "packages", "connectors"];
}

// Setup panes (Settings / Packages / Connectors) read identity from the
// hash. When the hash has none, the selected sidebar pane is the fallback
// so Agent CLIs → Packages still offers This workspace / This agent.
export function cliPaneSetupContext(route = {}, legacy = {}) {
  return {
    workspaceId: route.workspaceId || legacy.workspaceId || "",
    agentId: route.agentId || legacy.agentId || "",
    scope: route.scope || "user",
    focus: route.focus || "",
    // The settings layer rides along: the pane rewrites the hash to carry the
    // selected agent, and a rewrite that dropped the layer looked like a pill
    // click that did nothing (2026-09-12).
    layer: route.layer || "",
  };
}

// The Models pane's address: its workspace and the layer being edited.
export function cliModelsHash(cli, { workspaceId = "", layer = "" } = {}) {
  const q = new URLSearchParams();
  if (workspaceId) q.set("workspaceId", workspaceId);
  if (layer === "global" || layer === "project") q.set("layer", layer);
  const qs = q.toString();
  return "#/clis/" + encodeURIComponent(cli) + "/models" + (qs ? "?" + qs : "");
}

export function cliPaneHash(cli = "", pane = "launch", workspace = "") {
  if (!cli) return "#/clis";
  const id = encodeURIComponent(cli);
  if (!pane || pane === "launch") return "#/clis/" + id;
  if (pane === "sessions" && workspace) return "#/clis/" + id + "/sessions/" + encodeURIComponent(workspace);
  if (pane === "sessions") return "#/clis/" + id + "/sessions";
  if (pane === "terminals") return "#/clis/" + id + "/terminals";
  if (pane === "providers") return "#/clis/" + id + "/providers";
  if (pane === "models") return "#/clis/" + id + "/models";
  if (pane === "settings") return "#/clis/" + id + "/settings";
  if (pane === "keyboard") return "#/clis/" + id + "/keyboard";
  if (pane === "memory") return "#/clis/" + id + "/memory";
  if (pane === "packages") return "#/clis/" + id + "/packages";
  if (pane === "connectors") return "#/clis/" + id + "/connectors";
  return "#/clis/" + id;
}

function sessionsLocation(cli, workspace) {
  const id = cli || "pi";
  return { view: "clis", id, pane: "sessions", ...(workspace ? { workspace } : {}), redirect: cliPaneHash(id, "sessions", workspace) };
}

export function cliLocation(hash = "", legacy = {}) {
  const providers = cliProvidersLocation(hash);
  if (providers) return providers;
  const packages = cliPackagesLocation(hash, legacy.packageContext || {});
  if (packages) return packages;
  const keys = cliKeysLocation(hash);
  if (keys) return keys;
  const settings = cliSettingsLocation(hash, legacy.agentId || "");
  if (settings) {
    // The keyboard map left the settings sub-tab row (2026-09-12): a bookmark
    // from that day, or a link still carrying `?tab=keys`, lands on the pane.
    if (settings.keysTab) return { ...settings, keysTab: false, redirect: cliPaneHash(settings.id, "keyboard") };
    return settings;
  }
  const connectors = cliConnectorsLocation(hash, legacy.packageContext || {});
  if (connectors) return connectors;
  const [path, query] = hash.split("?");
  const parts = path.replace(/^#\//, "").split("/");
  const params = new URLSearchParams(query);
  const decode = (v) => { try { return decodeURIComponent(v || ""); } catch { return ""; } };
  // ADR-0079: sessions are a capability of a CLI. The 2026-09-11 amendment
  // nests them in that CLI's pane; old strip addresses rewrite.
  if (parts[0] === "sessions") return sessionsLocation(params.get("cli") || "pi", decode(parts[1]));
  if (parts[0] !== "clis") return { view: "clis", id: "", pane: "launch" };
  if (parts[1] === "profile") return { view: "profile", id: decode(parts[2]), cli: decode(parts[3]) };
  if (parts[1] === "new") return { view: "new", id: decode(parts[2]), ...(params.get("profile") ? { profile: params.get("profile") } : {}), ...(params.get("workspace") ? { workspace: params.get("workspace") } : {}) };
  if (parts[1] === "terminal") return { view: "terminal", id: decode(parts[2]) };
  if (parts[1] === "messages") return { view: "messages", id: decode(parts[2]) };
  // The general Terminals tab left the Agent CLIs view on 2026-09-11: a CLI's
  // own Terminals pane is the one list. The old address resolves to the
  // catalog at once and the view rewrites the hash to #/clis.
  if (parts[1] === "terminals") return { view: "clis", id: "", pane: "launch", redirect: "#/clis" };
  if (parts[1] === "sessions") return sessionsLocation(params.get("cli") || "pi", decode(parts[2]));
  const cli = decode(parts[1]);
  const panePart = parts[2] || "launch";
  const pane = CLI_PANES.has(panePart) ? panePart : "launch";
  const workspace = pane === "sessions" ? decode(parts[3]) : "";
  const loc = { view: "clis", id: cli, pane, ...(workspace ? { workspace } : {}) };
  if (pane === "providers") {
    const rest = decode(parts[3]);
    const rest2 = decode(parts[4]);
    if (rest === "new") loc.add = true;
    // The custom endpoint form is a page, not a dialog step (benchmarks.md
    // refuses modals for flows longer than 2 fields): /custom starts one,
    // /custom/<id> edits it. Anything deeper is an invalid link.
    else if (rest === "custom" && !rest2 && !parts[5]) loc.custom = "new";
    else if (rest === "custom" && rest2 && !parts[5]) { loc.custom = "edit"; loc.customId = rest2; }
    else if (rest) loc.invalid = true;
    if (["agentId", "workspaceId", "scope"].some((key) => params.has(key))) loc.scoped = true;
  }
  if (pane === "models") {
    // The catalog is read in a workspace and the lists are written to a layer,
    // so both ride the address; a path after the pane is not a link we made.
    loc.workspaceId = params.get("workspaceId") || "";
    loc.layer = params.get("layer") === "project" ? "project" : params.get("layer") === "global" ? "global" : "";
    if (parts[3]) loc.invalid = true;
  }
  if (pane === "memory") {
    loc.workspaceId = params.get("workspaceId") || "";
    loc.scope = params.get("scope") === "workspace" || params.get("scope") === "global" ? params.get("scope") : "";
  }
  if (pane === "settings") {
    loc.agentId = params.get("agentId") || "";
    loc.focus = params.get("focus") === "scoped-models" ? "scoped-models" : "";
    if (parts[3]) loc.invalid = true;
  }
  // `cliKeysLocation` handles the well-formed address; anything with a path
  // after the pane is as invalid as it is for settings.
  if (pane === "keyboard" && parts[3]) loc.invalid = true;
  if (pane === "packages") {
    const rest = parts.slice(3).map(decode);
    loc.pkg = rest[0] === "config" && rest[1] ? rest[1] : "";
    loc.workspaceId = params.get("workspaceId") || "";
    loc.agentId = params.get("agentId") || "";
    loc.scope = params.get("scope") || "user";
    if ((rest[0] && rest[0] !== "config") || rest[0] === "config" && !rest[1] || rest.length > 2) loc.invalid = true;
    if (!["user", "project", "agent"].includes(loc.scope)) loc.invalid = true;
  }
  if (pane === "connectors") {
    loc.workspaceId = params.get("workspaceId") || "";
    loc.agentId = params.get("agentId") || "";
    loc.scope = params.get("scope") || "user";
    if (parts[3]) loc.invalid = true;
    if (!["user", "project", "agent"].includes(loc.scope)) loc.invalid = true;
  }
  if (panePart === "launch" && parts[2]) loc.redirect = cliPaneHash(cli, "launch");
  return loc;
}

// PICODE_TOOL_FAMILIES are PiCode's own tool families a CLI agent can
// receive at launch as MCP servers (ADR-0154): the CLI agent's per-agent scope.
// The server validates the names; this list is what the form offers.
export const PICODE_TOOL_FAMILIES = [
  { id: "computer", label: "Computer", hint: "Use the Windows desktop through the desktop app. Off until you switch this terminal on in Settings ▸ Computer." },
  { id: "browser", label: "Browser", hint: "Read the page open in the work browser. Acting needs a grant in Settings ▸ Browser." },
  { id: "inbox", label: "Inbox", hint: "File notes and questions into your Inbox; a question waits for your answer there." },
  { id: "checklist", label: "Checklist", hint: "The agent's plan for the task, shown as the current step on this terminal's card." },
  { id: "delivery", label: "Delivery", hint: "Declare the change and ask for review in the project the CLI opens in; Git ▸ Delivery follows it. No merge, queue or deploy." },
	{ id: "mission", label: "Missions", hint: "Follow the assigned mission and report progress and evidence." },
];

const toolList = (tools) => Array.isArray(tools) ? tools.filter((t) => typeof t === "string" && t) : [];

export function launchDraft(c = {}) {
  return { executable: c.executable || "", argsText: (c.args || []).map((a) => !a.trim() || a.startsWith('"') ? JSON.stringify(a) : a).join("\n"), pathText: (c.path || []).join("\n"),
    envText: Object.entries(c.env || {}).map(([k, v]) => `${k}=${v}`).join("\n"), integration: !!c.integration, tools: toolList(c.tools) };
}

export const launchLines = (text) => String(text || "").split("\n").filter((line) => line.trim());

// launchArgs parses the args textarea into argv entries; a line that starts
// with a quote is JSON-decoded (the same encoding argLine writes back).
export const launchArgs = (text) => launchLines(text).map((a) => { if (a.startsWith('"')) { try { const s = JSON.parse(a); if (typeof s === "string") return s; } catch { /* literal argument */ } } return a; });

export function launchConfig(v) {
  const env = {};
  for (const line of launchLines(v.envText)) {
    const i = line.indexOf("=");
    env[line.slice(0, i).trim()] = line.slice(i + 1);
  }
  return { executable: v.executable.trim(), args: launchArgs(v.argsText), path: launchLines(v.pathText).map((p) => p.trim()), env, integration: v.integration, tools: toolList(v.tools) };
}

export function resolveLaunch(base, patch = {}) {
  const env = { ...base.env };
  for (const [k, v] of Object.entries(patch.env || {})) { if (v === null) delete env[k]; else env[k] = v; }
  return { ...base, ...patch, env };
}

export function launchOverrides(base, next) {
  const patch = {};
  for (const key of ["executable", "args", "path", "integration"]) {
    if (JSON.stringify(base[key]) !== JSON.stringify(next[key])) patch[key] = next[key];
  }
  // A config saved before ADR-0154 has no tools key: read it as none.
  if (JSON.stringify(toolList(base.tools)) !== JSON.stringify(toolList(next.tools))) patch.tools = toolList(next.tools);
  const env = {};
  for (const key of new Set([...Object.keys(base.env || {}), ...Object.keys(next.env || {})])) {
    if (base.env?.[key] !== next.env?.[key]) env[key] = next.env?.[key] ?? null;
  }
  if (Object.keys(env).length) patch.env = env;
  return patch;
}

export function cliTerminals(terminals, cli = "") {
  return terminals.filter((t) => {
    const observed = t.tui ? t.tui.cli : t.cli;
    return cli ? observed === cli || t.launchCli === cli : !!(observed || t.launchCli);
  });
}

export function terminalLaunchCLI(terminal, requested = "") {
  return terminal?.launchCli || terminal?.tui?.cli || terminal?.cli || requested;
}

export const defaultLaunchConfig = (integration) => ({ executable: "", args: [], path: [], env: {}, integration: !!integration, tools: [] });
export const cliWorkspaceList = (response) => Array.isArray(response) ? response : response?.workspaces || [];

// Resolve by the persisted binding, never the selected agent or route context.
export function terminalLaunchAgent(terminalId, workspaces = [], freeAgents = []) {
  if (!terminalId) return null;
  return [...workspaces.flatMap((w) => w.agents || []), ...freeAgents]
    .find((agent) => agent.terminalId === terminalId) || null;
}

export function launchChanged(applied, next) {
  return applied.cli !== next.cli || applied.fingerprint !== next.fingerprint || applied.executable !== next.executable || !!(applied.identity && applied.identity !== next.identity);
}

// Presets are copied, not linked. Explicit values (even empty) remain pinned.
export function profileOverrides(base, config) {
  const env = Object.fromEntries(Object.keys(base.env || {}).map((k) => [k, null]));
  return { ...config, args: [...config.args || []], path: [...config.path || []], tools: toolList(config.tools), env: { ...env, ...config.env } };
}

export function editLaunchOverrides(base, previous, next) {
  // Keep explicit pins even when they currently equal the CLI defaults.
  const before = resolveLaunch(base, previous);
  const changes = launchOverrides(before, next);
  return { ...previous, ...changes, ...(previous.env || changes.env ? { env: { ...previous.env, ...changes.env } } : {}) };
}

// Make agent (ADR-0184): a PiCode shell running a catalog CLI — detected, not
// launched, not a sign-in and not already an agent — offers to become one.
// Answers the CLI id to name, or "" when there is nothing to offer.
export function adoptOffer(term, boundAgent) {
  if (!term || boundAgent || term.kind || term.launchCli) return "";
  return (term.tui && term.tui.cli) || "";
}
