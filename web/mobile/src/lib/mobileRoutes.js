import { missionLocation } from "@picode/shared/domain/missions.js";
import { cliPackagesLocation } from "@picode/shared/domain/cliPackages.js";
import { cliSettingsLocation } from "@picode/shared/domain/cliSettings.js";
import { cliConnectorsLocation } from "@picode/shared/domain/integrations.js";
import { agentRoute, workspaceHash, termRoute, termHash, appPath } from "./routes.js";

// Mobile hash routes (ADR-0044/0095). Four tabs plus focused tools. The
// agent and terminal screens share the desktop's `#/agent/<id>` and
// `#/term/<id>` so a QR scan or a link pasted from the desktop lands on
// the same thing; every other desktop hash maps to the closest mobile
// section instead of a dead end. Work mirrors the desktop sidebar's rail:
// workspaces (agents + terminals per folder), free agents, terminals.
//   route := { screen: now|inbox|work|agent|term|inspector|app|more, id, section }
export const MORE_SECTIONS = ["pins", "snippets", "outcomes", "history", "llama", "devices", "preferences", "system", "notifications", "apps", "clis", "integrations", "automations"];
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
  clis: "clis",
  llama: "llama",
  preferences: "preferences",
  devices: "devices",
  system: "system",
  integrations: "integrations",
  termset: "preferences",
  pins: "pins",
  snippets: "snippets",
  outcomes: "outcomes",
  history: "history",
};

function strip(hash) {
  return (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
}

function dec(s) {
  try { return decodeURIComponent(s); } catch { return s; }
}

export function mobileRoute(hash) {
  const h = strip(hash);
  const mission = missionLocation(h);
  if (mission) return { screen: "missions", ...mission, section: "" };
  if (h === "/more/missions") return { screen: "missions", id: "", workspace: "", section: "" };
  if (cliPackagesLocation(h)) return { screen: "more", id: "", section: "clis" };
  if (cliSettingsLocation(h)) return { screen: "more", id: "", section: "clis" };
  if (cliConnectorsLocation(h)) return { screen: "more", id: "", section: "clis" };
  if (h.split("?")[0] === "/providers/llama" || h.split("?")[0] === "/more/providers/llama") return { screen: "more", id: "", section: "llama" };
  const agentId = agentRoute("#" + h);
  if (agentId) {
    const view = new URLSearchParams(h.split("?")[1] || "").get("view");
    return { screen: "agent", id: agentId, section: "", ...(view === "terminal" || view === "chat" ? { view } : {}) };
  }
  const termId = termRoute("#" + h);
  if (termId) return { screen: "term", id: termId, section: "" };
  const [toolPath, toolQuery = ""] = h.split("?");
  const toolMatch = /^\/(file|tree|files|git|inspector)\/(a|t|w)\/([^/]+)(?:\/(.+))?$/.exec(toolPath);
  if (toolMatch && (toolMatch[1] === "file" ? !!toolMatch[4] : !toolMatch[4])) {
    const params = new URLSearchParams(toolQuery);
    const view = params.get("view");
    return { screen: toolMatch[1] === "git" ? "git" : toolMatch[1] === "inspector" ? "inspector" : "files", id: dec(toolMatch[3]), section: { a: "agent", t: "term", w: "workspace" }[toolMatch[2]],
      ...(toolMatch[4] ? { path: dec(toolMatch[4]) } : {}),
      ...(params.get("root") ? { root: params.get("root") } : {}),
      ...(toolMatch[1] === "git" && params.get("commit") ? { commit: params.get("commit") } : {}),
      ...(toolMatch[1] === "inspector" && (view === "files" || view === "pr") ? { view } : {}) };
  }
  const parts = h.split("/").filter(Boolean);
  const head = parts[0] || "";
  if (!head) return { screen: "now", id: "", section: "" };
  if (head === "inbox") return { screen: "inbox", id: parts[1] ? dec(parts[1]) : "", section: "" };
  if (head === "work") {
    const sec = parts[1] ? dec(parts[1]) : "";
    // `#/work/workspaces/<id>` is the Back landing for a workspace's
    // agent or terminal: the Workspaces view focused on that group.
    const focus = sec === "workspaces" && parts[2] ? dec(parts[2]) : "";
    return { screen: "work", id: focus, section: WORK_SECTIONS.includes(sec) ? sec : "" };
  }
  if (head === "agents") return { screen: "work", id: "", section: "agents" };
  if (head === "terminals") return { screen: "work", id: "", section: "terminals" };
  if (head === "more") {
    const sec = parts[1] ? dec(parts[1]) : "";
    return { screen: "more", id: "", section: MORE_SECTIONS.includes(sec) ? sec : "" };
  }
  // Pins on the phone: the list lives under More; a pin opens read-only
  // (where a reminder's Open lands), `new` and `<id>/edit` are the form.
  if (head === "pins" && parts[1] === "new") return { screen: "pinEdit", id: "", section: "" };
  if (head === "pins" && parts[1] && parts[2] === "edit") return { screen: "pinEdit", id: dec(parts[1]), section: "" };
  if (head === "pins" && parts[1]) return { screen: "pin", id: dec(parts[1]), section: "" };
  // Snippets mirror Pins: the list lives under More; a snippet opens
  // read-only, `new` and `<id>/edit` are the form.
  if (head === "snippets" && parts[1] === "new") return { screen: "snipEdit", id: "", section: "" };
  if (head === "snippets" && parts[1] && parts[2] === "edit") return { screen: "snipEdit", id: dec(parts[1]), section: "" };
  if (head === "snippets" && parts[1]) return { screen: "snip", id: dec(parts[1]), section: "" };
  if (head === "app" && parts[1]) return { screen: "app", id: dec(parts[1]), section: "", ...(appPath("#" + h) ? { path: appPath("#" + h) } : {}) };
  if (head === "file" || head === "tree" || head === "git") {
    return { screen: "work", id: "", section: "" };
  }
  if (DESKTOP_TO_MORE[head]) return { screen: "more", id: "", section: DESKTOP_TO_MORE[head] };
  return { screen: "now", id: "", section: "" };
}

export function mobileHash(screen, id, section, view = "") {
  switch (screen) {
    case "missions": return id ? "#/mission/" + encodeURIComponent(id) : "#/missions";
    case "inbox": return id ? "#/inbox/" + encodeURIComponent(id) : "#/inbox";
    case "work": return id ? "#/work/" + encodeURIComponent(id) : "#/work";
    case "agent": return workspaceHash(id, view);
    case "term": return termHash(id);
    case "files": return toolHash("files", { kind: section, id });
    case "git": return toolHash("git", { kind: section, id });
    case "inspector": return toolHash("inspector", { kind: section, id }, { view });
    case "more": return id ? "#/more/" + encodeURIComponent(id) : "#/more";
    case "app": return "#/app/" + encodeURIComponent(id);
    case "pin": return "#/pins/" + encodeURIComponent(id);
    case "pinEdit": return id ? "#/pins/" + encodeURIComponent(id) + "/edit" : "#/pins/new";
    case "snip": return "#/snippets/" + encodeURIComponent(id);
    case "snipEdit": return id ? "#/snippets/" + encodeURIComponent(id) + "/edit" : "#/snippets/new";
    default: return "#/";
  }
}

// tabOf: which bottom tab a route lights up. A pushed screen keeps its
// parent tab lit so the user always knows where Back will land.
export function tabOf(route) {
  if (!route) return "now";
  if (route.screen === "agent" || route.screen === "term" || route.screen === "work" || ["inspector", "files", "git"].includes(route.screen)) return "work";
  if (route.screen === "inbox") return "inbox";
  if (route.screen === "missions") return "work";
  if (route.screen === "more" || route.screen === "app" || route.screen === "pin" || route.screen === "pinEdit" || route.screen === "snip" || route.screen === "snipEdit") return "more";
  return "now";
}

// parentHash: where Back goes when there is no history entry to pop —
// the tab a pushed screen belongs to, or the More menu for a section.
// An agent or terminal backs into the Work view that owns it: with a
// workspace (wsId) the Workspaces view focused on that group; a free one
// (wsId === null) its own flat list; an unknown owner (wsId undefined —
// the fleet has not answered yet) keeps the legacy parent.
export function parentHash(route, wsId) {
  if (!route) return "#/";
  if (route.screen === "missions") return route.id || route.create ? "#/missions" : "#/work";
  if (route.screen === "app") return "#/more/apps";
  if (route.screen === "agent") {
    if (wsId) return "#/work/workspaces/" + encodeURIComponent(wsId);
    return wsId === null ? "#/work/agents" : "#/work";
  }
  if (route.screen === "term") {
    if (wsId) return "#/work/workspaces/" + encodeURIComponent(wsId);
    return "#/work/terminals";
  }
  if (["inspector", "files", "git"].includes(route.screen)) {
    if (route.section === "agent") return workspaceHash(route.id);
    if (route.section === "term") return termHash(route.id);
    return "#/work";
  }
  if (route.screen === "inbox" && route.id) return "#/inbox";
  if (route.screen === "pin") return "#/more/pins";
  if (route.screen === "pinEdit") return route.id ? "#/pins/" + encodeURIComponent(route.id) : "#/more/pins";
  if (route.screen === "snip") return "#/more/snippets";
  if (route.screen === "snipEdit") return route.id ? "#/snippets/" + encodeURIComponent(route.id) : "#/more/snippets";
  if (route.screen === "more" && route.section) return "#/more";
  return "#/";
}

// Compatible desktop links, with the optional folder equality precondition.
export function toolHash(screen, owner, { path = "", root = "", commit = "", view = "" } = {}) {
  const kind = { agent: "a", term: "t", workspace: "w" }[owner.kind];
  if (!kind || !owner.id) return "#/work";
  const head = screen === "git" ? "git" : screen === "inspector" ? "inspector" : path ? "file" : "tree";
  const params = new URLSearchParams();
  if (root) params.set("root", root);
  if (screen === "git" && commit) params.set("commit", commit);
  if (screen === "inspector" && view) params.set("view", view);
  return "#/" + head + "/" + kind + "/" + encodeURIComponent(owner.id) + (head === "file" ? "/" + encodeURIComponent(path) : "") + (params.size ? "?" + params : "");
}
