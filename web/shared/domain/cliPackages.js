// One pane, every CLI (ADR-0176): the engine is the source of what a CLI is —
// its scopes, its capabilities, its catalog — and this module is the whole of
// the contract a browser owns: the request each control sends, the vocabulary
// the pane speaks for the surface a report declares, and the copy shown where a
// verb does not exist. Nothing here lists CLIs: a report is asked for and the
// answer decides (the two hardcoded lists this file used to carry are gone).
//
// Pi keeps its own API for its own mutations — the package pages, the config
// descriptors, the store write behind the agent scope — while a vendor's
// plugins go through the job lane (ADR-0087). Which one a pane uses is the
// driver's own declaration (`Caps.Async`), not a branch on which CLI it is.
//
// The surface a report declares (`Catalog`) also decides which vocabulary the
// pane speaks: a CLI whose installable list is PiCode's own npm gallery holds
// *packages*, and one whose list is the vendor's holds *plugins*. Both word
// sets were measured in the two panes this file served (ADR-0167/0102) and are
// kept verbatim, so the merge changes which component draws a control and not
// what it says.

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
  const adoptPane = !!(legacy && !explicit);
  const fallback = adoptPane ? legacyContext : {};
  const workspaceId = params.get("workspaceId") || fallback.workspaceId || "";
  const agentId = params.get("agentId") || fallback.agentId || "";
  const scope = params.get("scope") || "user";
  invalid ||= !["user", "project", "agent"].includes(scope);
  const route = { view: "clis", pane: "packages", id, pkg, workspaceId, agentId, scope, legacy: legacy || strip, invalid, ...(adoptPane ? { adoptPane: true } : {}) };
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

// --- the surface a report declares -------------------------------------------

// packagesSurface is where a CLI's installable rows come from: PiCode's own npm
// gallery ("gallery", Pi's) or the vendor's own store ("vendor", everyone
// else's). A report with no catalog at all still speaks the vendor's words —
// its rows are the CLI's own plugins, read from its files.
export function packagesSurface(report) {
  return report && report.catalog === "gallery" ? "gallery" : "vendor";
}

// PANE_WORDS is the copy that differs between the two surfaces, sentence by
// sentence, exactly as each pane said it before the merge. Anything the two
// surfaces say identically is not here.
export const PANE_WORDS = {
  gallery: {
    scopeLabel: "Install to",
    scopeGroup: "Install to",
    sourcePlaceholder: "npm:pkg  ·  git:github.com/user/repo  ·  ./path",
    sourceLabel: "Package source",
    access: "Packages run with full access. Only install what you review.",
    installedFilter: "Filter installed packages…",
    installedFilterLabel: "Filter installed packages",
    noMatch: q => "No installed package matches \u201c" + q + "\u201d.",
    emptyTitle: "Nothing installed yet",
    loading: "Loading packages",
    fallback: "Machine packages",
    isolation: "Only this agent's packages (skip machine and folder). Restart to apply.",
  },
  vendor: {
    scopeLabel: "Plugins go to",
    scopeGroup: "Plugin scope",
    sourcePlaceholder: "npm module, git URL, or name@marketplace",
    sourceLabel: "Plugin source",
    access: "Plugins run with full access. Only install what you review.",
    installedFilter: "Filter installed plugins…",
    installedFilterLabel: "Filter installed plugins",
    noMatch: q => "No installed plugin matches \u201c" + q + "\u201d.",
    emptyTitle: "Nothing installed.",
    loading: "Loading plugins",
    fallback: "Use global",
    isolation: "",
  },
};

// paneWords answers the copy one surface speaks, never null: a caller that
// renders before a report exists gets the vendor's words, which is what a CLI
// without a catalog reads as.
export function paneWords(surface) {
  return PANE_WORDS[surface] || PANE_WORDS.vendor;
}

// directMutation says whether a mutation is one of PiCode's own calls — the
// source, its layer, and the target the answer comes back for — rather than a
// job in the CLI's own lane. A CLI whose mutations are direct calls (`Caps.Async`
// false, Pi's own pipkg) always is; so is any write into the agent scope, and any
// row that lives there, because that layer is PiCode's own list on the agent row
// for every CLI — the CLI's launch is what passes the entries on (ADR-0176 slice
// 4). A vendor row in a vendor's layer is the lane's, and only its.
export function directMutation(caps, { scope = "user", row = null } = {}) {
  if (!caps || !caps.async) return true;
  return (row ? row.scope : scope) === "agent";
}

// paneTabs says whether the Installed/Marketplace pair is offered. A report with
// no catalog holds one list and has nothing to switch to; and a vendor's catalog
// does not exist for the agent layer — that list is PiCode's own on the agent
// row, so the Marketplace tab there would be a control that cannot work
// (ADR-0176 slice 4). Pi's gallery, which is PiCode's own, stays offered.
export function paneTabs(report, scope = "user") {
  if (!report || report.catalog === "") return false;
  return !(scope === "agent" && report.catalog !== "gallery");
}

// behindFor finds the catalog's answer for one installed row. The CLI's own
// update check names the source, the layer it lives in and the version pair; a
// row the check did not name is not behind, and nothing else may invent one — a
// blind Update button is a fabricated control (ADR-0167).
export function behindFor(updates, row) {
  const list = Array.isArray(updates) ? updates : [];
  const r = row || {};
  return list.find(entry => entry && entry.source === r.source && (!entry.scope || !r.vendor || entry.scope === r.vendor)) || null;
}

// --- the CLI's own plugins (ADR-0167) ----------------------------------------
//
// Every control comes from the report's `caps`, every absence from its `notes`.
// The pane never restates a CLI's capabilities.

// The verb names the contract uses, and the keys a CLI's `notes` may use to
// explain that verb's absence. `enable`/`disable` are two vendor words for the
// one toggle; `marketplace-add` is the source-management half of `marketplace`.
const GUEST_VERB_NOTES = {
  install: ["install"],
  remove: ["remove"],
  toggle: ["toggle", "enable", "disable"],
  update: ["update"],
  inspect: ["inspect"],
  marketplace: ["marketplace", "marketplace-add"],
};

// packagesNotes is the copy a capability-driven pane must not forget: one
// sentence per verb this CLI does not expose, then every note whose key is a
// concept rather than a verb (`capabilities`, `approve`, `local`, `app`) — so
// each blank cell is explained once, in the vendor's terms, and never by a
// control that could only fail.
export function packagesNotes(caps, notes) {
  // Both arguments arrive from a report that may not exist yet, and `null` is
  // not what a default parameter covers: a caller passing `report && report.notes`
  // hands over null before the first load. Coerce here, once, so a pane cannot
  // blank itself by rendering before its data (found in the browser, 2026-09-20).
  caps = caps || {};
  notes = notes || {};
  const out = [];
  const spoken = new Set();
  for (const [verb, keys] of Object.entries(GUEST_VERB_NOTES)) {
    for (const key of keys) if (notes[key]) spoken.add(key);
    if (caps[verb]) continue;
    const key = keys.find(candidate => notes[candidate]);
    if (key) out.push(notes[key]);
  }
  for (const [key, text] of Object.entries(notes)) {
    if (spoken.has(key) || !text || out.includes(text)) continue;
    out.push(text);
  }
  return out;
}

// catalogRowAction decides what a Marketplace row may offer. The vendor's own
// catalog has to name the spec a command takes: Omp's `omp plugin discover`
// prints a name and a version and never the marketplace an install needs
// (`omp plugin install <name>` resolves through npm instead), so its rows are
// information and the pane states the install form in the list note instead of
// offering a button that would run the wrong command.
export function catalogRowAction(caps, row) {
  const c = caps || {};
  const r = row || {};
  if (r.installed) return "installed";
  if (c.catalogInstall && r.source) return "install";
  return "none";
}

// matchParts splits text around every occurrence of the filter's needle, so a
// filtered card can show *why* it matched instead of just surviving the filter.
// The needle is compared literally (a `(` in a filter must not break the pane)
// and case-insensitively.
export function matchParts(text, needle) {
  const full = String(text == null ? "" : text);
  const want = String(needle == null ? "" : needle).trim();
  if (!want) return [{ text: full, hit: false }];
  const hay = full.toLowerCase();
  const pin = want.toLowerCase();
  const out = [];
  let at = 0;
  for (;;) {
    const found = hay.indexOf(pin, at);
    if (found < 0) break;
    if (found > at) out.push({ text: full.slice(at, found), hit: false });
    out.push({ text: full.slice(found, found + want.length), hit: true });
    at = found + want.length;
  }
  if (!out.length) return [{ text: full, hit: false }];
  if (at < full.length) out.push({ text: full.slice(at), hit: false });
  return out;
}

// SOURCE_GROUPS names where an installed plugin came from, in the order the
// list shows them. The keys are the vendors' own provenance words
// (`sourceKind`), so a list that mixes them — Hermes' bundled rows beside yours,
// Claude's claude.ai-managed ones beside the machine's — reads as groups instead
// of one undifferentiated column.
export const SOURCE_GROUPS = [
  { key: "picode", label: "Added by PiCode" },
  { key: "marketplace", label: "From a marketplace" },
  { key: "npm", label: "From npm" },
  { key: "git", label: "From a Git source" },
  { key: "path", label: "From a local path" },
  { key: "local", label: "Local plugin files" },
  { key: "user", label: "Installed by you" },
  { key: "bundled", label: "Ships with the CLI" },
  { key: "synced", label: "Managed by claude.ai" },
  { key: "imported", label: "Imported from another CLI" },
  { key: "other", label: "Other" },
];

// sourceGroupKey maps one row onto its group. PiCode's own integration wins (a
// reader should find it without knowing which vendor installed it), then the
// vendor's own word — the *specific* ones before the marketplace fallback, since
// Claude Code names a synced plugin's marketplace "synced" too, which filed it
// under marketplaces. Only a row whose vendor said nothing about the kind falls
// back on what the row shows.
export function sourceGroupKey(row) {
  const r = row || {};
  if (r.managedByPiCode) return "picode";
  const kind = String(r.sourceKind || "").toLowerCase();
  if (kind.startsWith("marketplace")) return "marketplace";
  if (kind === "bundled") return "bundled";
  if (kind === "synced") return "synced";
  if (kind === "import") return "imported";
  if (kind === "local-file") return "local";
  if (kind === "npm") return "npm";
  if (kind === "git") return "git";
  if (kind === "path" || kind === "native-local") return "path";
  if (kind === "user") return "user";
  if (kind) return "other";
  if (r.marketplace) return "marketplace";
  if (r.installPath && r.source) return "path";
  return "other";
}

// groupInstalledRows groups the visible rows, keeping the group order of
// SOURCE_GROUPS and the row order inside each group. One group is the whole
// list, so it gets no header: chrome that explains nothing is noise.
export function groupInstalledRows(rows) {
  const list = Array.isArray(rows) ? rows : [];
  const byKey = new Map();
  for (const row of list) {
    const key = sourceGroupKey(row);
    if (!byKey.has(key)) byKey.set(key, []);
    byKey.get(key).push(row);
  }
  if (byKey.size < 2) return [];
  return SOURCE_GROUPS.filter(group => byKey.has(group.key))
    .map(group => ({ key: group.key, label: group.label, rows: byKey.get(group.key) }));
}

// refusalCommand reads the command a refusal carries. The server attaches it
// (rendered by the same builder it executed) when the CLI refused and only a
// person in a terminal can answer — Grok's `--trust`, Claude's
// marketplace-declared command — so the pane shows the line to run instead of
// a dead end.
export function refusalCommand(error) {
  const body = error && error.body;
  const command = body && typeof body.command === "string" ? body.command.trim() : "";
  return { message: (error && error.message) || "", command };
}

// packagesApi(cli, {workspaceId, agentId, scope}) is the request builder for
// every route one pane needs (ADR-0176). The reads answer the unified report —
// the roster, its badge, and the catalog a CLI's mechanism keeps — and the
// verbs answer where that mechanism runs them: a vendor's own command through
// the job lane, PiCode's own package API for a CLI whose mutations are direct
// calls that answer the new list (`Caps.Async` false). A pane never assembles a
// URL, a query name or a body field inline: it asks for the route it needs, so
// one rename happens in one place and the shapes are testable without a browser.
//
// `ref` on a marketplace action is the scope the vendor's own add/update runs
// in (Claude's `--scope`, Muse's and Omp's working directory), so a machine
// action sends none — Codex's `--ref <git ref>` is never handed a scope word.
// `confirmTerminals` mirrors the clijob lane it already has: PiCode asks before
// a job that lands in a CLI whose terminals are live (plan §6).
export function packagesApi(cli, { workspaceId = "", agentId = "", scope = "user" } = {}) {
  const id = cli || "";
  const path = (route, { refresh = false, agent = false, vendor = false } = {}) => {
    const query = new URLSearchParams({ cli: id });
    if (workspaceId) query.set("workspace", workspaceId);
    if (agent && agentId) query.set("agent", agentId);
    // The layer being read. The unified reads take the CLI's own word for it
    // (`vendor` — the class alone cannot say Claude Code's uncommitted
    // `local`), the vendor routes take the same word under `scope`, which is
    // the query they have always answered.
    if (vendor) {
      if (scope && scope !== "user") query.set("vendor", scope);
    } else if (scope && scope !== "user") {
      query.set("scope", scope);
    }
    if (refresh) query.set("refresh", "1");
    return route + "?" + query;
  };
  const body = ({ name = "", source = "", requestKey = "", confirmTerminals = false, on } = {}) => {
    const out = { cli: id, scope };
    if (workspaceId) out.workspace = workspaceId;
    if (name) out.name = name;
    if (source) out.source = source;
    if (requestKey) out.requestKey = requestKey;
    if (confirmTerminals) out.confirmTerminals = true;
    if (on !== undefined) out.on = !!on;
    return out;
  };
  const post = (route, fields) => ({ method: "POST", path: route, body: body(fields) });
  // PiCode's own package API takes the source, its layer, and the workspace and
  // agent the fresh list is read back for — a project-scope write with no
  // workspace would land on the machine instead.
  const directBody = fields => {
    const out = { source: fields.source, scope: fields.scope };
    if (workspaceId) out.workspaceId = workspaceId;
    if (agentId) out.agentId = agentId;
    return out;
  };
  const get = (route, opts) => ({ method: "GET", path: path(route, opts), body: null });
  const ref = fields => (fields.ref ? { ref: fields.ref } : scope && scope !== "user" ? { ref: scope } : {});
  return {
    // The unified report: what this CLI is and what it holds, in one read.
    report: opts => get("/api/packages/report", { ...opts, agent: true, vendor: true }),
    // The badge read: the rows this CLI's own catalog has moved ahead of.
    updates: opts => get("/api/packages/updates", { ...opts, vendor: true }),
    // PiCode's own gallery (the catalog a `gallery` report declares).
    gallery: q => ({ method: "GET", path: "/api/packages/gallery?q=" + encodeURIComponent(String(q == null ? "" : q).trim()), body: null }),
    // The CLI's own configured sources. A CLI with no source-management verb
    // answers 400, which is the contract's "it has none", not a failure.
    available: () => get("/api/cli-packages/available"),
    marketplaces: () => get("/api/cli-packages/marketplaces"),
    install: fields => post("/api/cli-packages/install", fields),
    remove: fields => post("/api/cli-packages/remove", fields),
    update: fields => post("/api/cli-packages/update", fields),
    toggle: (fields, on) => post("/api/cli-packages/toggle", { ...fields, on }),
    marketplace: (action, fields = {}) => ({
      method: "POST",
      path: "/api/cli-packages/marketplace",
      body: { ...body(fields), action, ...ref(fields) },
    }),
    inspect: fields => ({
      method: "POST",
      path: "/api/cli-packages/inspect",
      body: { cli: id, target: (fields && (fields.name || fields.target)) || "" },
    }),
    // PiCode's own package API, for a CLI whose mutations are direct calls that
    // answer the new list (`Caps.Async` false): the source, its layer, and the
    // target the answer comes back for.
    direct: {
      install: fields => ({ method: "POST", path: "/api/packages", body: directBody(fields) }),
      update: fields => ({ method: "POST", path: "/api/packages/update", body: directBody(fields) }),
      remove: fields => {
        const query = new URLSearchParams({ source: fields.source, scope: fields.scope });
        if (workspaceId) query.set("workspace", workspaceId);
        if (agentId) query.set("agent", agentId);
        return { method: "DELETE", path: "/api/packages?" + query, body: null };
      },
    },
  };
}
