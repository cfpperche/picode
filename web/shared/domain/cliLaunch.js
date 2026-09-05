export function cliLocation(hash = "") {
  const [path, query] = hash.split("?");
  const parts = path.replace(/^#\//, "").split("/");
  const params = new URLSearchParams(query);
  if (parts[0] !== "clis") return { view: "clis", id: "" };
  const decode = (v) => { try { return decodeURIComponent(v || ""); } catch { return ""; } };
  if (parts[1] === "profile") return { view: "profile", id: decode(parts[2]), cli: decode(parts[3]) };
  if (parts[1] === "new") return { view: "new", id: decode(parts[2]), ...(params.get("profile") ? { profile: params.get("profile") } : {}), ...(params.get("workspace") ? { workspace: params.get("workspace") } : {}) };
  if (parts[1] === "terminal") return { view: "terminal", id: decode(parts[2]) };
  if (parts[1] === "terminals") return { view: "terminals", id: "" };
  return { view: "clis", id: decode(parts[1]) };
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
