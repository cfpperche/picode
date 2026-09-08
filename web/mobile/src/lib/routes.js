import { cliSettingsLocation, cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
// Mobile route helpers. Tool links live in mobileRoutes; desktop tab identities are absent.
const PREF_SECTIONS = ["appearance", "layout", "shortcuts", "notifications", "server", "backup"];

export function agentRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/agent\/([^/]+)$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function workspaceHash(agentId) {
  return agentId ? "#/agent/" + encodeURIComponent(agentId) : "#/";
}

export function termRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/term\/([^/]+)$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function termHash(id) {
  return id ? "#/term/" + encodeURIComponent(id) : "#/";
}

export function appPath(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  const match = /^\/app\/[^/]+\/(.*)$/.exec(h);
  if (!match) return "";
  return match[1].split("/").map((part) => { try { return decodeURIComponent(part); } catch { return part; } }).join("/");
}

export function prefSection(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  const m = /^\/preferences\/([a-z]+)$/.exec(h);
  if (m && PREF_SECTIONS.includes(m[1])) return m[1];
  return "appearance";
}

export function go(section) {
  location.hash = "#/more/" + encodeURIComponent(section || "providers");
}

export function automationRoute(hash) {
  const value = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  const match = /^\/(?:automations|more\/automations)(?:\/([^/]+))?$/.exec(value);
  if (!match) return null;
  if (!match[1]) return "";
  try { return decodeURIComponent(match[1]); } catch { return match[1]; }
}

export function automationsHash(id) {
  return id ? "#/automations/" + encodeURIComponent(id) : "#/automations";
}
