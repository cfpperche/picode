// Hash routes. Preferences is PiCode-the-product. Settings is pi (ADR-0012).
export const ROUTES = {
  workspace: "/",
  preferences: "/preferences",
  settings: "/settings",
  system: "/system",
  providers: "/providers",
  mcps: "/mcps",
  packages: "/packages",
  devices: "/devices",
  pins: "/pins",
  sessions: "/sessions/:id",
};

export function parseRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  if (h === "/preferences" || h.startsWith("/preferences/")) return "preferences";
  if (h === "/settings") return "settings";
  if (h === "/system") return "system";
  if (h === "/providers" || h.startsWith("/providers/")) return "providers";
  if (h === "/mcps") return "mcps";
  if (h === "/packages") return "packages";
  if (h === "/devices") return "devices";
  if (h === "/pins" || h.startsWith("/pins/")) return "pins";
  if (h.startsWith("/sessions") || h.startsWith("/sessions/")) return "sessions";
  if (h.startsWith("/term/")) return "workspace";
  if (h.startsWith("/file/")) return "workspace";
  if (h.startsWith("/git/")) return "workspace";
  return "workspace";
}

export function agentRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/agent\/([^/]+)$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function termRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/term\/([^/]+)$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function workspaceHash(agentId) {
  return agentId ? "#/agent/" + encodeURIComponent(agentId) : "#/";
}

export function termHash(id) {
  return id ? "#/term/" + encodeURIComponent(id) : "#/";
}

export function sessionsHash(wsId) {
  return wsId ? "#/sessions/" + encodeURIComponent(wsId) : "#/";
}

export function sessionsRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  const m = /^\/sessions\/([^/]+)$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function termTabId(id) {
  return id ? "t:" + id : "";
}

export function isTermTab(id) {
  return String(id || "").startsWith("t:");
}

export function tabTermId(id) {
  return isTermTab(id) ? String(id).slice(2) : "";
}

export function fileTabId(kind, id, path) {
  const k = kind === "term" ? "t" : "a";
  return "f:" + k + ":" + String(id || "") + ":" + encodeURIComponent(path || "");
}

export function isFileTab(id) {
  return String(id || "").startsWith("f:");
}

export function parseFileTab(id) {
  const m = /^f:(t|a):([^:]+):(.+)$/.exec(String(id || ""));
  if (!m) return null;
  try {
    return { kind: m[1] === "t" ? "term" : "agent", id: m[2], path: decodeURIComponent(m[3]) };
  } catch {
    return null;
  }
}

export function fileHash(kind, id, path) {
  const k = kind === "term" ? "t" : "a";
  return "#/file/" + k + "/" + encodeURIComponent(id || "") + "/" + encodeURIComponent(path || "");
}

export function fileRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/file\/(t|a)\/([^/]+)\/(.+)$/.exec(h);
  if (!m) return null;
  try {
    return { kind: m[1] === "t" ? "term" : "agent", id: decodeURIComponent(m[2]), path: decodeURIComponent(m[3]) };
  } catch {
    return null;
  }
}

export function pinRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  if (h === "/pins" || h === "/pins/new") return { mode: "new", id: "" };
  const m = /^\/pins\/([^/]+)$/.exec(h);
  if (m) return { mode: "edit", id: decodeURIComponent(m[1]) };
  return { mode: "", id: "" };
}

const PREF_SECTIONS = ["appearance", "terminal", "notifications", "server", "backup"];

export function prefSection(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  const m = /^\/preferences\/([a-z]+)$/.exec(h);
  if (m && PREF_SECTIONS.includes(m[1])) return m[1];
  return "appearance";
}

export function providersNew(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  return h === "/providers/new";
}

export function providersLlama(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  return h === "/providers/llama";
}

export function go(name, agentId) {
  if (typeof name === "string" && name.startsWith("preferences")) {
    const sec = name === "preferences" ? "" : name.slice("preferences-".length);
    location.hash = sec ? "#/preferences/" + sec : "#/preferences";
    return;
  }
  if (name === "providers-new") {
    location.hash = "#/providers/new";
    return;
  }
  if (name === "providers-llama") {
    location.hash = "#/providers/llama";
    return;
  }
  if (name === "pins-new") {
    location.hash = "#/pins/new";
    return;
  }
  if (typeof name === "string" && name.startsWith("pin:")) {
    location.hash = "#/pins/" + encodeURIComponent(name.slice(4));
    return;
  }
  if (!name || name === "workspace") {
    location.hash = workspaceHash(agentId);
    return;
  }
  const path = ROUTES[name] || "/";
  location.hash = path === "/" ? "#/" : "#" + path;
}

// Git graph (ADR-0022). The hash names the *owner* that asked, because the
// owner is what authorises the read; the tab id names the *repository*, so two
// agents in two worktrees of one repo land on the same tab.
export function gitHash(kind, id) {
  const k = kind === "term" ? "t" : "a";
  return "#/git/" + k + "/" + encodeURIComponent(id || "");
}

export function gitRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/git\/(t|a)\/([^/]+)$/.exec(h);
  if (!m) return null;
  try {
    return { kind: m[1] === "t" ? "term" : "agent", id: decodeURIComponent(m[2]) };
  } catch {
    return null;
  }
}

export function gitTabId(key) {
  return key ? "g:" + key : "";
}

export function isGitTab(id) {
  return String(id || "").startsWith("g:");
}

export function gitTabKey(id) {
  return isGitTab(id) ? String(id).slice(2) : "";
}
