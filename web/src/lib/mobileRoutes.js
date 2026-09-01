import { agentRoute, workspaceHash } from "./routes.js";

// Mobile hash routes (ADR-0044). Four tabs plus two pushed screens. The
// agent screen shares the desktop's `#/agent/<id>` so a QR scan or a link
// pasted from the desktop lands on the same agent; every other desktop
// hash maps to the closest mobile section instead of a dead end.
//   route := { screen: now|inbox|agents|agent|more, id, section }
export const MORE_SECTIONS = ["devices", "preferences", "settings", "system", "providers", "mcps", "packages", "notifications"];

const DESKTOP_TO_MORE = {
  preferences: "preferences",
  providers: "providers",
  devices: "devices",
  settings: "settings",
  system: "system",
  mcps: "mcps",
  packages: "packages",
  termset: "preferences",
  pins: "settings",
};

function strip(hash) {
  return (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
}

function dec(s) {
  try { return decodeURIComponent(s); } catch { return s; }
}

export function mobileRoute(hash) {
  const h = strip(hash);
  const agentId = agentRoute("#" + h);
  if (agentId) return { screen: "agent", id: agentId, section: "" };
  const parts = h.split("/").filter(Boolean);
  const head = parts[0] || "";
  if (!head) return { screen: "now", id: "", section: "" };
  if (head === "inbox") return { screen: "inbox", id: parts[1] ? dec(parts[1]) : "", section: "" };
  if (head === "agents") return { screen: "agents", id: "", section: "" };
  if (head === "more") {
    const sec = parts[1] ? dec(parts[1]) : "";
    return { screen: "more", id: "", section: MORE_SECTIONS.includes(sec) ? sec : "" };
  }
  if (head === "app" && parts[1] === "inbox") return { screen: "inbox", id: "", section: "" };
  if (head === "sessions" || head === "term" || head === "file" || head === "tree" || head === "git") {
    return { screen: "agents", id: "", section: "" };
  }
  if (DESKTOP_TO_MORE[head]) return { screen: "more", id: "", section: DESKTOP_TO_MORE[head] };
  return { screen: "now", id: "", section: "" };
}

export function mobileHash(screen, id) {
  switch (screen) {
    case "inbox": return id ? "#/inbox/" + encodeURIComponent(id) : "#/inbox";
    case "agents": return "#/agents";
    case "agent": return workspaceHash(id);
    case "more": return id ? "#/more/" + encodeURIComponent(id) : "#/more";
    default: return "#/";
  }
}

// tabOf: which bottom tab a route lights up. A pushed screen keeps its
// parent tab lit so the user always knows where Back will land.
export function tabOf(route) {
  if (!route) return "now";
  if (route.screen === "agent") return "agents";
  if (route.screen === "inbox") return "inbox";
  if (route.screen === "agents") return "agents";
  if (route.screen === "more") return "more";
  return "now";
}

// parentHash: where Back goes when there is no history entry to pop —
// the tab a pushed screen belongs to, or the More menu for a section.
export function parentHash(route) {
  if (!route) return "#/";
  if (route.screen === "agent") return "#/agents";
  if (route.screen === "inbox" && route.id) return "#/inbox";
  if (route.screen === "more" && route.section) return "#/more";
  return "#/";
}
