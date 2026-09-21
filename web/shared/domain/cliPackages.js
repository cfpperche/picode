// Pi's own pane, and nothing else: the eight guest CLIs are covered by
// GUEST_PACKAGES below, and an id that is in neither list gets the
// "in development" placeholder. Pi is not a guest — it keeps its own API,
// scopes (machine/workspace/agent) and config descriptors (ADR-0102/0099).
export const CLI_PACKAGES = [{ id: "pi", name: "Pi" }];
export const supportsCliPackages = id => CLI_PACKAGES.some(cli => cli.id === id);

// The eight guest CLIs whose vendor owns the plugin store (ADR-0167), in the
// catalog order `internal/clipkgs` declares. Pi is deliberately absent: its
// pane, API and package semantics are its own (ADR-0102) and it keeps
// `CLI_PACKAGES` above. A Go conformance test keeps the two lists in step.
export const GUEST_PACKAGES = ["claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy", "omp"];
export const usesGuestPackages = id => GUEST_PACKAGES.includes(id);

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

// --- guest CLI plugins (ADR-0167) --------------------------------------------
//
// The guest pane talks to one frozen HTTP contract (plan §10) and takes every
// control from the answer's `caps`, every absence from its `notes`. It never
// restates a CLI's capabilities: these two functions are the whole of the
// contract a browser owns — the request shapes, and the copy shown where a
// verb does not exist.

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

// guestPackagesNotes is the copy a capability-driven pane must not forget: one
// sentence per verb this CLI does not expose, then every note whose key is a
// concept rather than a verb (`capabilities`, `approve`, `local`, `app`) — so
// each blank cell is explained once, in the vendor's terms, and never by a
// control that could only fail.
export function guestPackagesNotes(caps, notes) {
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

// rowUpdateState decides what an installed row offers as an update. The vendor
// has to expose the verb (caps.update) AND its own catalog has to say something
// newer: a blind Update button is a fabricated control (ADR-0167), so before
// the first check the pane offers "Check for updates" instead of one button per
// row, and a row that is current offers nothing.
export function rowUpdateState(caps, row, checked) {
  const c = caps || {};
  const r = row || {};
  if (!c.update) return "none";
  if (r.updateAvailable) return "update";
  return checked ? "current" : "unknown";
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

// guestPackagesApi(cli, {workspaceId, scope}) is the request builder for every
// route in the frozen contract (plan §10). A pane never assembles a URL, a
// query name or a body field inline: it asks for the route it needs, so one
// rename happens in one place and the shapes are testable without a browser.
//
// `ref` on a marketplace action is the scope the vendor's own add/update runs
// in (Claude's `--scope`, Muse's and Omp's working directory), so a machine
// action sends none — Codex's `--ref <git ref>` is never handed a scope word.
// `confirmTerminals` mirrors the clijob lane it already has: PiCode asks before
// a job that lands in a CLI whose terminals are live (plan §6).
export function guestPackagesApi(cli, { workspaceId = "", scope = "user" } = {}) {
  const id = cli || "";
  const path = (route, { refresh = false } = {}) => {
    const query = new URLSearchParams({ cli: id });
    if (workspaceId) query.set("workspace", workspaceId);
    if (scope && scope !== "user") query.set("scope", scope);
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
  const get = (route, opts) => ({ method: "GET", path: path(route, opts), body: null });
  const ref = fields => (fields.ref ? { ref: fields.ref } : scope && scope !== "user" ? { ref: scope } : {});
  return {
    roster: opts => get("/api/cli-packages", opts),
    available: () => get("/api/cli-packages/available"),
    // The CLI's own configured sources. A CLI with no source-management verb
    // answers 400, which is the contract's "it has none", not a failure.
    marketplaces: () => get("/api/cli-packages/marketplaces"),
    // The availability check: the CLI's catalog compared with its roster.
    updates: opts => get("/api/cli-packages/updates", opts),
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
  };
}
