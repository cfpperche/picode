export function cliLocation(hash = "") {
  const parts = hash.replace(/^#\//, "").split("/");
  if (parts[0] !== "clis") return { view: "clis", id: "" };
  const decode = (v) => { try { return decodeURIComponent(v || ""); } catch { return ""; } };
  if (parts[1] === "new") return { view: "new", id: decode(parts[2]) };
  if (parts[1] === "terminal") return { view: "terminal", id: decode(parts[2]) };
  if (parts[1] === "terminals") return { view: "terminals", id: "" };
  return { view: "clis", id: decode(parts[1]) };
}

export function launchDraft(c = {}) {
  return { executable: c.executable || "", argsText: (c.args || []).join("\n"), pathText: (c.path || []).join("\n"),
    envText: Object.entries(c.env || {}).map(([k, v]) => `${k}=${v}`).join("\n"), integration: !!c.integration };
}

export const launchLines = (text) => String(text || "").split("\n").filter((line) => line.trim());

export function launchConfig(v) {
  const env = {};
  for (const line of launchLines(v.envText)) {
    const i = line.indexOf("=");
    env[line.slice(0, i).trim()] = line.slice(i + 1);
  }
  return { executable: v.executable.trim(), args: launchLines(v.argsText), path: launchLines(v.pathText).map((p) => p.trim()), env, integration: v.integration };
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
