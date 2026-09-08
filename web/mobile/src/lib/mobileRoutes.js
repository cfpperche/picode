import { cliSettingsLocation } from "@picode/shared/domain/cliSettings.js";
import { agentRoute, workspaceHash, termRoute, termHash, appPath } from "./routes.js";

// Mobile hash routes (ADR-0044/0095). Four tabs plus focused tools. The
// agent and terminal screens share the desktop's `#/agent/<id>` and
// `#/term/<id>` so a QR scan or a link pasted from the desktop lands on
// the same thing; every other desktop hash maps to the closest mobile
// section instead of a dead end. Work mirrors the desktop sidebar's rail:
// workspaces (agents + terminals per folder), free agents, terminals.
//   route := { screen: now|inbox|work|agent|term|changes|app|more, id, section }
//   changes: `#/changes/<a|t|w>/<id>` — the owner's uncommitted working tree,
//   read-only (ADR-0044 phase 3); section carries the owner kind.
export const MORE_SECTIONS = ["llama", "devices", "preferences", "settings", "system", "providers", "mcps", "packages", "notifications", "apps", "clis", "integrations", "automations"];
export const WORK_SECTIONS = ["workspaces", "agents", "terminals"];
const WORK_KEY = "picode-mobile-work";

export function readWorkSection() {
  try { const v = localStorage.getItem(WORK_KEY); return WORK_SECTIONS.includes(v) ? v : "workspaces"; } catch { return "workspaces"; }
}
export function writeWorkSection(v) {
  try { if (WORK_SECTIONS.includes(v)) localStorage.setItem(WORK_KEY, v); } catch { /* per-viewer nicety */ }
}

const DESKTOP_TO_MORE = {
  automations: "automations",
  sessions: "clis",
  clis: "clis",
  llama: "llama",
  preferences: "preferences",
  providers: "providers",
  devices: "devices",
  settings: "settings",
  system: "system",
  mcps: "mcps",
  integrations: "integrations",
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
  if (cliSettingsLocation(h)) return { screen: "more", id: "", section: "clis" };
  if (h === "/providers/llama") return { screen: "more", id: "", section: "llama" };
  if (h === "/preferences/status") return { screen: "more", id: "", section: "clis" };
  const agentId = agentRoute("#" + h);
  if (agentId) return { screen: "agent", id: agentId, section: "" };
  const termId = termRoute("#" + h);
  if (termId) return { screen: "term", id: termId, section: "" };
  const [toolPath, toolQuery = ""] = h.split("?");
  const toolMatch = /^\/(file|tree|files|git)\/(a|t|w)\/([^/]+)(?:\/(.+))?$/.exec(toolPath);
  if (toolMatch && (toolMatch[1] === "file" ? !!toolMatch[4] : !toolMatch[4])) {
    const params = new URLSearchParams(toolQuery);
    return { screen: toolMatch[1] === "git" ? "git" : "files", id: dec(toolMatch[3]), section: { a: "agent", t: "term", w: "workspace" }[toolMatch[2]],
      ...(toolMatch[4] ? { path: dec(toolMatch[4]) } : {}),
      ...(params.get("root") ? { root: params.get("root") } : {}),
      ...(toolMatch[1] === "git" && params.get("commit") ? { commit: params.get("commit") } : {}) };
  }
  const parts = h.split("/").filter(Boolean);
  const head = parts[0] || "";
  if (!head) return { screen: "now", id: "", section: "" };
  if (head === "inbox") return { screen: "inbox", id: parts[1] ? dec(parts[1]) : "", section: "" };
  if (head === "changes" && parts[1] && parts[2]) {
    const kind = { a: "agent", t: "term", w: "workspace" }[parts[1]];
    if (kind) return { screen: "changes", id: dec(parts[2]), section: kind };
  }
  if (head === "work") {
    const sec = parts[1] ? dec(parts[1]) : "";
    return { screen: "work", id: "", section: WORK_SECTIONS.includes(sec) ? sec : "" };
  }
  if (head === "agents") return { screen: "work", id: "", section: "agents" };
  if (head === "terminals") return { screen: "work", id: "", section: "terminals" };
  if (head === "more") {
    const sec = parts[1] ? dec(parts[1]) : "";
    return { screen: "more", id: "", section: MORE_SECTIONS.includes(sec) ? sec : "" };
  }
  if (head === "app" && parts[1] === "inbox") return { screen: "inbox", id: "", section: "" };
  if (head === "app" && parts[1]) return { screen: "app", id: dec(parts[1]), section: "", ...(appPath("#" + h) ? { path: appPath("#" + h) } : {}) };
  if (head === "file" || head === "tree" || head === "git") {
    return { screen: "work", id: "", section: "" };
  }
  if (DESKTOP_TO_MORE[head]) return { screen: "more", id: "", section: DESKTOP_TO_MORE[head] };
  return { screen: "now", id: "", section: "" };
}

export function mobileHash(screen, id, section) {
  switch (screen) {
    case "inbox": return id ? "#/inbox/" + encodeURIComponent(id) : "#/inbox";
    case "work": return id ? "#/work/" + encodeURIComponent(id) : "#/work";
    case "agent": return workspaceHash(id);
    case "term": return termHash(id);
    case "files": return toolHash("files", { kind: section, id });
    case "git": return toolHash("git", { kind: section, id });
    case "changes": return "#/changes/" + ({ agent: "a", term: "t", workspace: "w" }[section] || "a") + "/" + encodeURIComponent(id);
    case "more": return id ? "#/more/" + encodeURIComponent(id) : "#/more";
    case "app": return "#/app/" + encodeURIComponent(id);
    default: return "#/";
  }
}

// tabOf: which bottom tab a route lights up. A pushed screen keeps its
// parent tab lit so the user always knows where Back will land.
export function tabOf(route) {
  if (!route) return "now";
  if (route.screen === "agent" || route.screen === "term" || route.screen === "work" || ["changes", "files", "git"].includes(route.screen)) return "work";
  if (route.screen === "inbox") return "inbox";
  if (route.screen === "more" || route.screen === "app") return "more";
  return "now";
}

// parentHash: where Back goes when there is no history entry to pop —
// the tab a pushed screen belongs to, or the More menu for a section.
export function parentHash(route) {
  if (!route) return "#/";
  if (route.screen === "app") return "#/more/apps";
  if (route.screen === "agent") return "#/work";
  if (route.screen === "term") return "#/work/terminals";
  if (["changes", "files", "git"].includes(route.screen)) {
    if (route.section === "agent") return workspaceHash(route.id);
    if (route.section === "term") return termHash(route.id);
    return "#/work";
  }
  if (route.screen === "inbox" && route.id) return "#/inbox";
  if (route.screen === "more" && route.section) return "#/more";
  return "#/";
}

// Compatible desktop links, with the optional folder equality precondition.
export function toolHash(screen, owner, { path = "", root = "", commit = "" } = {}) {
  const kind = { agent: "a", term: "t", workspace: "w" }[owner.kind];
  if (!kind || !owner.id) return "#/work";
  const head = screen === "git" ? "git" : path ? "file" : "tree";
  const params = new URLSearchParams();
  if (root) params.set("root", root);
  if (screen === "git" && commit) params.set("commit", commit);
  return "#/" + head + "/" + kind + "/" + encodeURIComponent(owner.id) + (head === "file" ? "/" + encodeURIComponent(path) : "") + (params.size ? "?" + params : "");
}
