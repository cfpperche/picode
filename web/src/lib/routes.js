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
  if (h.startsWith("/term/")) return "workspace";
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

export function termTabId(id) {
  return id ? "t:" + id : "";
}

export function isTermTab(id) {
  return String(id || "").startsWith("t:");
}

export function tabTermId(id) {
  return isTermTab(id) ? String(id).slice(2) : "";
}

export function pinRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  if (h === "/pins" || h === "/pins/new") return { mode: "new", id: "" };
  const m = /^\/pins\/([^/]+)$/.exec(h);
  if (m) return { mode: "edit", id: decodeURIComponent(m[1]) };
  return { mode: "", id: "" };
}

const PREF_SECTIONS = ["appearance", "notifications", "server", "backup"];

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
