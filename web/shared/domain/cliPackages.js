// Native package capabilities are independent of terminal launch support.
export const CLI_PACKAGES = [{ id: "pi", name: "Pi" }];
export const supportsCliPackages = id => CLI_PACKAGES.some(cli => cli.id === id);

export function cliPackagesHash(cli = "pi", { workspaceId = "", agentId = "", scope = "user", pkg = "" } = {}) {
  const query = new URLSearchParams();
  if (workspaceId) query.set("workspaceId", workspaceId);
  if (agentId) query.set("agentId", agentId);
  if (scope !== "user") query.set("scope", scope);
  return "#/clis/" + encodeURIComponent(cli || "pi") + "/packages" + (pkg ? "/config/" + encodeURIComponent(pkg) : "") + (query.size ? "?" + query : "");
}

export function cliPackagesLocation(hash = "", legacyContext = {}) {
  const [path, query = ""] = hash.replace(/^#/, "").split("?");
  const nested = /^\/clis\/([^/]+)\/packages(?:\/config\/([^/]+))?$/.exec(path);
  const strip = path === "/clis/packages" || path.startsWith("/clis/packages/");
  const legacy = /^\/(?:more\/)?packages(?:\/|$)/.test(path);
  if (!legacy && !strip && !nested) return null;
  const params = new URLSearchParams(query);
  const match = nested || (legacy ? /^\/(?:more\/)?packages(?:\/config\/([^/]+))?$/.exec(path)
    : /^\/clis\/packages(?:\/([^/]+)(?:\/config\/([^/]+))?)?$/.exec(path));
  let id = "pi", pkg = "", invalid = !match;
  try {
    if (nested) {
      id = decodeURIComponent(nested[1] || "pi");
      pkg = decodeURIComponent(nested[2] || "");
    } else if (match) {
      id = legacy ? "pi" : decodeURIComponent(match[1] || "pi");
      pkg = decodeURIComponent(match[legacy ? 1 : 2] || "");
    }
  } catch { invalid = true; }
  // An explicit context, including an intentionally empty one, never inherits
  // a different selected pane. Canonical links never consult the current pane.
  const explicit = params.has("workspaceId") || params.has("agentId");
  const fallback = legacy && !explicit ? legacyContext : {};
  const workspaceId = params.get("workspaceId") || fallback.workspaceId || "";
  const agentId = params.get("agentId") || fallback.agentId || "";
  const scope = params.get("scope") || "user";
  invalid ||= !["user", "project", "agent"].includes(scope);
  const route = { view: "clis", pane: "packages", id, pkg, workspaceId, agentId, scope, legacy: legacy || strip, invalid };
  return { ...route, redirect: !invalid && !nested ? cliPackagesHash(id, route) : "" };
}

const missing = message => Object.assign(new Error(message), { status: 404 });

// Names and runtime status may refresh; an edited draft must keep its file target.
export function packageContextKey({ workspace, agent }) {
  return JSON.stringify([workspace?.id || "", workspace?.path || "", agent?.id || "", agent?.workPath || ""]);
}

export async function loadPiPackagesContext({ workspaceId = "", agentId = "", scope = "user" }, request) {
  if (!["user", "project", "agent"].includes(scope)) throw missing("This package scope is not available.");
  if (!agentId && scope === "agent") throw missing("Choose an agent to use this package scope.");
  if (!agentId && !workspaceId) {
    if (scope === "project") throw missing("Choose a workspace to use this package scope.");
    return { workspace: null, agent: null };
  }
  const workspaces = await request("/api/workspaces");
  let workspace = workspaceId ? workspaces.find(row => row.id === workspaceId) : null;
  if (workspaceId && !workspace) throw missing("This workspace is no longer available.");
  let agent = null;
  if (agentId) {
    const owner = workspaces.find(row => row.agents?.some(agent => agent.id === agentId));
    if (owner) {
      if (workspaceId && workspaceId !== owner.id) throw missing("This agent does not belong to this workspace.");
      workspace = owner;
      agent = owner.agents.find(row => row.id === agentId);
    } else {
      const free = await request("/api/agents?free=1");
      agent = free.find(row => row.id === agentId);
      if (!agent) throw missing("This agent is no longer available.");
      if (workspaceId) throw missing("This agent does not belong to this workspace.");
    }
  }
  if (!workspace && scope === "project") throw missing("Choose a workspace to use this package scope.");
  return { workspace, agent };
}
