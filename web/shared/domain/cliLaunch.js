import { cliProvidersLocation } from "./cliProviders.js";
import { cliPackagesLocation } from "./cliPackages.js";
import { cliKeysLocation, cliSettingsLocation } from "./cliSettings.js";
import { cliConnectorsLocation } from "./integrations.js";

const CLI_PANES = new Set(["launch", "terminals", "sessions", "providers", "settings", "keyboard", "packages", "connectors"]);

export function cliDetectOnly(cli) {
  return !!(cli && cli.surface === "detect");
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

export function cliPaneHash(cli = "", pane = "launch", workspace = "") {
  if (!cli) return "#/clis";
  const id = encodeURIComponent(cli);
  if (!pane || pane === "launch") return "#/clis/" + id;
  if (pane === "sessions" && workspace) return "#/clis/" + id + "/sessions/" + encodeURIComponent(workspace);
  if (pane === "sessions") return "#/clis/" + id + "/sessions";
  if (pane === "terminals") return "#/clis/" + id + "/terminals";
  if (pane === "providers") return "#/clis/" + id + "/providers";
  if (pane === "settings") return "#/clis/" + id + "/settings";
  if (pane === "keyboard") return "#/clis/" + id + "/keyboard";
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
    if (rest === "new") loc.add = true;
    else if (rest) loc.invalid = true;
    if (["agentId", "workspaceId", "scope"].some((key) => params.has(key))) loc.scoped = true;
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

export function launchDraft(c = {}) {
  return { executable: c.executable || "", argsText: (c.args || []).map((a) => !a.trim() || a.startsWith('"') ? JSON.stringify(a) : a).join("\n"), pathText: (c.path || []).join("\n"),
    envText: Object.entries(c.env || {}).map(([k, v]) => `${k}=${v}`).join("\n"), integration: !!c.integration };
}

export const launchLines = (text) => String(text || "").split("\n").filter((line) => line.trim());

export function launchConfig(v) {
  const env = {};
  for (const line of launchLines(v.envText)) {
    const i = line.indexOf("=");
    env[line.slice(0, i).trim()] = line.slice(i + 1);
  }
  const args = launchLines(v.argsText).map((a) => { if (a.startsWith('"')) { try { const s = JSON.parse(a); if (typeof s === "string") return s; } catch { /* literal argument */ } } return a; });
  return { executable: v.executable.trim(), args, path: launchLines(v.pathText).map((p) => p.trim()), env, integration: v.integration };
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

export const defaultLaunchConfig = (integration) => ({ executable: "", args: [], path: [], env: {}, integration: !!integration });
export const cliWorkspaceList = (response) => Array.isArray(response) ? response : response?.workspaces || [];

export function launchChanged(applied, next) {
  return applied.cli !== next.cli || applied.fingerprint !== next.fingerprint || applied.executable !== next.executable || !!(applied.identity && applied.identity !== next.identity);
}

// Presets are copied, not linked. Explicit values (even empty) remain pinned.
export function profileOverrides(base, config) {
  const env = Object.fromEntries(Object.keys(base.env || {}).map((k) => [k, null]));
  return { ...config, args: [...config.args || []], path: [...config.path || []], env: { ...env, ...config.env } };
}

export function editLaunchOverrides(base, previous, next) {
  // Keep explicit pins even when they currently equal the CLI defaults.
  const before = resolveLaunch(base, previous);
  const changes = launchOverrides(before, next);
  return { ...previous, ...changes, ...(previous.env || changes.env ? { env: { ...previous.env, ...changes.env } } : {}) };
}
