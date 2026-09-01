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
  termset: "/termset",
  automations: "/automations",
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
  if (h === "/termset" || h.startsWith("/termset/")) return "termset";
  if (h === "/automations" || h.startsWith("/automations/")) return "automations";
  if (h.startsWith("/sessions") || h.startsWith("/sessions/")) return "sessions";
  if (h.startsWith("/term/")) return "workspace";
  if (h.startsWith("/file/")) return "workspace";
  if (h.startsWith("/git/")) return "workspace";
  if (h.startsWith("/tree/")) return "workspace";
  if (h.startsWith("/app/")) return "workspace";
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

// Owner kinds encode as one letter: t = terminal, a = agent, and since
// ADR-0030 w = workspace (a folder can be read with nobody in it, ADR-0027).
function ownerLetter(kind) {
  if (kind === "term") return "t";
  if (kind === "workspace") return "w";
  return "a";
}

function ownerKind(letter) {
  if (letter === "t") return "term";
  if (letter === "w") return "workspace";
  return "agent";
}

export function fileTabId(kind, id, path) {
  return "f:" + ownerLetter(kind) + ":" + String(id || "") + ":" + encodeURIComponent(path || "");
}

export function isFileTab(id) {
  return String(id || "").startsWith("f:");
}

export function parseFileTab(id) {
  const m = /^f:(t|a|w):([^:]+):(.+)$/.exec(String(id || ""));
  if (!m) return null;
  try {
    return { kind: ownerKind(m[1]), id: m[2], path: decodeURIComponent(m[3]) };
  } catch {
    return null;
  }
}

export function fileHash(kind, id, path) {
  return "#/file/" + ownerLetter(kind) + "/" + encodeURIComponent(id || "") + "/" + encodeURIComponent(path || "");
}

export function fileRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/file\/(t|a|w)\/([^/]+)\/(.+)$/.exec(h);
  if (!m) return null;
  try {
    return { kind: ownerKind(m[1]), id: decodeURIComponent(m[2]), path: decodeURIComponent(m[3]) };
  } catch {
    return null;
  }
}

// termsetRoute: null on the global page, the terminal id on a terminal's.
export function termsetRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  const m = /^\/termset\/([^/]+)$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function pinRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  if (h === "/pins" || h === "/pins/new") return { mode: "new", id: "" };
  const m = /^\/pins\/([^/]+)$/.exec(h);
  if (m) return { mode: "edit", id: decodeURIComponent(m[1]) };
  return { mode: "", id: "" };
}

// "terminal" left this list on 2026-08-30: terminal appearance lives on the
// terminal settings page now (#/termset), beside the behaviour it belongs
// with. An old #/preferences/terminal link falls back to Appearance.
const PREF_SECTIONS = ["appearance", "shortcuts", "notifications", "server", "backup"];

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
  if (name === "sessions") {
    // Machine-wide view; the per-workspace one is #/sessions/<id>.
    location.hash = "#/sessions";
    return;
  }
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

// File tree (ADR-0030). Same two identities as the git graph: the hash names
// the owner that authorises the read, the tab id names the canonical root
// folder, so every owner confined to one folder shares one tab.
export function treeHash(kind, id) {
  return "#/tree/" + ownerLetter(kind) + "/" + encodeURIComponent(id || "");
}

export function treeRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/tree\/(t|a|w)\/([^/]+)$/.exec(h);
  if (!m) return null;
  try {
    return { kind: ownerKind(m[1]), id: decodeURIComponent(m[2]) };
  } catch {
    return null;
  }
}

export function treeTabId(root) {
  return root ? "d:" + root : "";
}

export function isTreeTab(id) {
  return String(id || "").startsWith("d:");
}

export function treeTabRoot(id) {
  return isTreeTab(id) ? String(id).slice(2) : "";
}

// Apps host (ADR-0036). One identity only: the tab id and the hash both
// name the app — no owner sidecar, an app tab is self-describing.
export function appTabId(id) {
  return id ? "x:" + id : "";
}

export function isAppTab(id) {
  return String(id || "").startsWith("x:");
}

export function tabAppId(id) {
  return isAppTab(id) ? String(id).slice(2) : "";
}

export function appHash(id) {
  return id ? "#/app/" + encodeURIComponent(id) : "#/";
}

export function appRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/app\/([^/]+)$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

// Automations (ADR-0045): "#/automations" is the list, "#/automations/new"
// the editor, "#/automations/<id>" one automation. null = not ours.
export function automationRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/automations(?:\/([^/]+))?$/.exec(h);
  if (!m) return null;
  if (!m[1]) return "";
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function automationsHash(sub) {
  return sub ? "#/automations/" + encodeURIComponent(sub) : "#/automations";
}
