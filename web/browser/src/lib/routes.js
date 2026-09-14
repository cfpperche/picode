import { cliProvidersLocation, cliProvidersHash } from "@picode/shared/domain/cliProviders.js";
import { cliPackagesLocation, cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import { cliSettingsLocation, cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
import { cliConnectorsHash } from "@picode/shared/domain/integrations.js";
import { cliLocation, cliPaneHash } from "@picode/shared/domain/cliLaunch.js";
// Hash routes. Preferences is PiCode-the-product. Native settings live under Agent CLIs (ADR-0101).
// Sessions live under Agent CLIs (ADR-0079); /clis/* views are parsed by
// cliLocation in @picode/shared/domain/cliLaunch.js.
export const ROUTES = {
  workspace: "/",
  preferences: "/preferences",
  clis: "/clis",
  settings: "/clis/pi/settings",
  system: "/system",
  providers: "/clis/pi/providers",
  llama: "/llama/models",
  mcps: "/clis/pi/connectors",
  integrations: "/integrations/webhooks",
  packages: "/clis/pi/packages",
  devices: "/devices",
  browser: "/browser",
  pins: "/pins",
  termset: "/termset",
  automations: "/automations",
  snippets: "/snippets",
};

export function parseRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  if (h === "/preferences/status" || h === "/clis" || h.startsWith("/clis/")) return "clis";
  if (h === "/preferences" || h.startsWith("/preferences/")) return "preferences";
  if (cliPackagesLocation(h)) return "clis";
  if (cliSettingsLocation(h)) return "clis";
  if (h === "/system") return "system";
  if (h === "/llama" || h.startsWith("/llama/") || ["/providers/llama", "/more/providers/llama"].includes(h.split("?")[0])) return "llama";
  if (cliProvidersLocation(h)) return "clis";
  if (h === "/integrations/webhooks" || h.startsWith("/integrations/webhooks")) return "integrations";
  if (h === "/mcps" || h === "/integrations" || h.startsWith("/integrations/")) return "clis";
  if (h === "/devices") return "devices";
  if (h === "/browser") return "browser";
  if (h === "/pins" || h.startsWith("/pins/")) return "pins";
  if (h === "/termset" || h.startsWith("/termset/")) return "termset";
  if (h === "/automations" || h.startsWith("/automations/")) return "automations";
  if (h === "/snippets" || h.startsWith("/snippets/")) return "snippets";
  // Legacy #/sessions* deep links render the Agent CLIs shell; AgentClis
  // redirects the hash to #/clis/<cli>/sessions* (ADR-0079).
  if (h.startsWith("/sessions") || h.startsWith("/sessions/")) return "clis";
  if (h.startsWith("/web/")) return "workspace";
  if (h.startsWith("/term/")) return "workspace";
  if (h.startsWith("/file/")) return "workspace";
  if (h.startsWith("/git/")) return "workspace";
  if (h.startsWith("/tree/")) return "workspace";
  if (h.startsWith("/app/")) return "workspace";
  return "workspace";
}

// Compatibility helpers; canonical package URLs carry their own context.
export function packagesConfigRoute(hash) {
  return cliPackagesLocation(hash || (typeof location !== "undefined" ? location.hash : ""))?.pkg || null;
}
export function packagesConfigHash(pkg, context = {}) {
  return cliPackagesHash("pi", { ...context, pkg });
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

// Work browser tabs are session-scoped native surfaces, but they own an
// address: without one the hash router kept re-resolving the previous tab's
// route and yanked the selection back (2026-09-14). The id is the sequence
// openWebTab assigned (tab "w:<id>", here without the prefix).
export function webHash(id) {
  return id ? "#/web/" + encodeURIComponent(id) : "#/";
}

export function webRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/web\/([^/]+)$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

// Sessions live on the selected CLI's pane (ADR-0079 amendment 2026-09-11):
// machine-wide is #/clis/<cli>/sessions, one folder is #/clis/<cli>/sessions/<workspaceId>.
export function sessionsHash(wsId, cli = "pi") {
  return cliPaneHash(cli || "pi", "sessions", wsId || "");
}

export function sessionsRoute(hash) {
  const loc = cliLocation(hash || (typeof location !== "undefined" ? location.hash : ""));
  return loc.pane === "sessions" && loc.workspace ? loc.workspace : null;
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
export function ownerLetter(kind) {
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

// Pin Studio (#/pins/<id>) — the one place that builds the studio's hash, so
// a caller never assembles it by hand.
export function pinHash(id) {
  return id ? "#/pins/" + encodeURIComponent(id) : "#/pins";
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
  return !!cliProvidersLocation(h)?.add;
}

export function providersLlama(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  return h === "/providers/llama";
}

export function go(name, agentId, extra = {}) {
  const ctx = { agentId: extra.agentId || agentId || "", workspaceId: extra.workspaceId || "" };
  if (name === "settings") { location.hash = cliSettingsHash("pi", ctx); return; }
  if (name === "packages") { location.hash = cliPackagesHash("pi", ctx); return; }
  if (name === "mcps" || name === "connectors") { location.hash = cliConnectorsHash("pi", ctx); return; }
  if (typeof name === "string" && name.startsWith("preferences")) {
    const sec = name === "preferences" ? "" : name.slice("preferences-".length);
    location.hash = sec ? "#/preferences/" + sec : "#/preferences";
    return;
  }
  if (name === "providers-new") {
    location.hash = cliProvidersHash("pi", { add: true });
    return;
  }
  if (name === "providers-llama") {
    location.hash = "#/llama/models";
    return;
  }
  if (name === "pins-new") {
    location.hash = "#/pins/new";
    return;
  }
  if (name === "snippets-new") {
    location.hash = "#/snippets/new";
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
  return "#/git/" + ownerLetter(kind) + "/" + encodeURIComponent(id || "");
}

export function gitRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/git\/(t|a|w)\/([^/]+)$/.exec(h);
  if (!m) return null;
  try {
    return { kind: ownerKind(m[1]), id: decodeURIComponent(m[2]) };
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

// Work browser tabs (Phase 3, docs/plans/desktop-v2.md). Session-scoped:
// they do not survive an app restart (the pages live in native webviews).
export function webTabId(seq) {
  return seq ? "w:" + seq : "";
}

export function isWebTab(id) {
  return String(id || "").startsWith("w:");
}

export function tabWebId(id) {
  return isWebTab(id) ? String(id).slice(2) : "";
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

// A tab id is a tagged surface (`t:` terminal, `f:` file, `d:` folder,
// `g:` repository, `x:` app, `w:` web) or a bare agent id — the one thing the
// strip does not tag (ADR-0012, ADR-0022). Callers that ask "is the reader
// looking at an agent?" must ask this, not "is it a terminal tab": the
// role-state and slash fetches did the latter and paid two 404s for every
// file, tree, git and app tab selected (a chat composer asking a repository
// its role).
export function isAgentTab(id) {
  const s = String(id || "");
  return !!s && !isTermTab(s) && !isFileTab(s) && !isGitTab(s) && !isTreeTab(s) && !isAppTab(s) && !isWebTab(s);
}

export function appHash(id, path = "") {
  return id ? "#/app/" + encodeURIComponent(id) + (path ? "/" + path.split("/").map(encodeURIComponent).join("/") : "") : "#/";
}

// ADR-0118: the Matrix app became Canvas — id, hash and tab id. One map,
// read by the hash redirect (App.jsx) and by the tab restore
// (lib/openTabs.js), so an old bookmark and an old tab strip land in the
// same place and no second mechanism can drift from this one.
const RENAMED_APPS = { matrix: "canvas" };

// renamedAppId(id) -> the id an app answers to now, or "" when nothing moved.
export function renamedAppId(id) {
  return Object.hasOwn(RENAMED_APPS, id) ? RENAMED_APPS[id] : "";
}

// renamedAppHash(hash) -> the hash to replace this one with, or "": the same
// deep link under the app's current id, path and all (#/app/matrix/<id> →
// #/app/canvas/<id>). Only #/app/* is answered; every other hash is not ours.
export function renamedAppHash(hash) {
  const h = hash || (typeof location !== "undefined" ? location.hash : "") || "";
  const was = appRoute(h);
  const now = was ? renamedAppId(was) : "";
  return now ? appHash(now, appPath(h)) : "";
}

// renamedTabId(id) -> the app tab id this one is now, or "": an `x:matrix`
// restored from localStorage opens the canvas app instead of a dead tab.
export function renamedTabId(id) {
  if (!isAppTab(id)) return "";
  const now = renamedAppId(tabAppId(id));
  return now ? appTabId(now) : "";
}

export function appRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/app\/([^/]+)(?:\/.*)?$/.exec(h);
  if (!m) return null;
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function appPath(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "");
  const match = /^\/app\/[^/]+\/(.*)$/.exec(h);
  if (!match) return "";
  return match[1].split("/").map((part) => { try { return decodeURIComponent(part); } catch { return part; } }).join("/");
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

// Snippets (ADR-0130): "#/snippets" list, "#/snippets/new" editor,
// "#/snippets/<id>" one snippet. null = not ours.
export function snippetRoute(hash) {
  const h = (hash || (typeof location !== "undefined" ? location.hash : "") || "").replace(/^#/, "") || "/";
  const m = /^\/snippets(?:\/([^/]+))?$/.exec(h);
  if (!m) return null;
  if (!m[1]) return "";
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

export function snippetsHash(sub) {
  return sub ? "#/snippets/" + encodeURIComponent(sub) : "#/snippets";
}
