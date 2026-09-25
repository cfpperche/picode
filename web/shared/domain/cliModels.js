// The Models pane (ADR-0181, slice 2 of docs/benchmarks/2026-09-22-omp-helpers-placement.md):
// what a CLI says it can reach, and the two lists in its own config that
// decide which of those it may use. Only a CLI with a measured read-only
// catalog command has the pane; `internal/climodels` is the server's half and
// TestJSListMatchesTheReaders keeps the two lists equal.

export const MODELS_CLIS = ["omp"];

export const supportsCliModels = (id) => MODELS_CLIS.includes(id);

// CLIs PiCode can ask for their model list (internal/climodels.Supported();
// a Go test holds the two in step). Having a reader is not having the Models
// pane above: omp alone has that.
export const MODEL_READERS = ["omp", "pi", "codex", "opencode", "muse", "grok", "claude-code", "agy"];

// The native-settings text field each reader's list fills — the list is a
// shortcut beside the field, never a replacement: a model the CLI does not
// list (a new one, an alias) can still be typed.
const MODEL_PICK_FIELDS = { codex: "model", opencode: "model", muse: "model", grok: "models.default", "claude-code": "model", agy: "model" };
export const modelPickField = (cli, key) => MODEL_PICK_FIELDS[cli] === key;

// The one step beside a list that could not be read (the empty/blocked rule:
// one line + one action). The server names the step when PiCode can unblock
// the read (`action` in /api/cli-models' error: "signin", "open"); anything
// else — a slow CLI, a missing binary — is worth asking again.
export function modelsStep(cli, action, cliLabel = "") {
  const id = encodeURIComponent(cli);
  if (action === "signin") return { label: "Sign in", href: "#/clis/" + id + "/providers/new" };
  // Opening the CLI once is starting an agent of it: the new-agent screen.
  if (action === "open") return { label: "Open " + (cliLabel || "the CLI"), href: "#/clis/new/" + id };
  return { label: "Try again", retry: true };
}

// The two keys the pane edits, in the CLI's own names.
export const ALLOWED_KEY = "enabledModels";
export const HIDDEN_KEY = "disabledProviders";

// A config list arrives as whatever the file holds: a list of strings, one
// string, nothing, or — for these two keys — objects scoped to a folder
// (`{path, models}`), which the server reports as unreadable and never writes.
export function stringList(value) {
  if (Array.isArray(value)) return value.filter((v) => typeof v === "string");
  if (typeof value === "string" && value) return [value];
  return [];
}

// The list in force for one layer, and where it comes from. Arrays *replace*
// across layers in omp — a project list is the whole list for that project,
// not an addition to the global one — so the effective value is the nearest
// layer at or below this one that sets the key, never a merge.
export function effectiveList(layers = [], scope = "user", key = ALLOWED_KEY) {
  const index = layers.findIndex((l) => l.scope === scope);
  const upto = index < 0 ? layers.length - 1 : index;
  for (let i = upto; i >= 0; i -= 1) {
    const layer = layers[i];
    if (layer && Object.prototype.hasOwnProperty.call(layer.values || {}, key)) {
      return { list: stringList(layer.values[key]), setHere: i === index, from: i === index ? "" : layer.label };
    }
  }
  return { list: [], setHere: false, from: "" };
}

// The complete list to write when one entry is toggled on this layer. A layer
// that does not set the key yet starts from what it inherits — writing only
// the new entry would silently drop every entry the parent had, because the
// new list replaces the parent's.
export function toggledList(layers, scope, key, entry, on) {
  const { list } = effectiveList(layers, scope, key);
  const without = list.filter((v) => v !== entry);
  return on ? [...without, entry] : without;
}

// The layer's own value is unreadable when the server says so: a folder-scoped
// entry is a shape PiCode leaves as written.
export function isUnreadable(layer, key) {
  return !!layer && (layer.unreadable || []).includes(key);
}

// Rows grouped by provider, providers sorted, models sorted by name inside.
export function groupByProvider(models = []) {
  const map = new Map();
  for (const m of models) {
    if (!map.has(m.provider)) map.set(m.provider, []);
    map.get(m.provider).push(m);
  }
  return [...map.keys()].sort().map((provider) => ({
    provider,
    models: map.get(provider).slice().sort((a, b) => (a.name || a.id).localeCompare(b.name || b.id)),
  }));
}

// Kind chips and a free-text search, both over what the CLI reported.
export function filterModels(models = [], { kind = "", query = "" } = {}) {
  const q = query.trim().toLowerCase();
  return models.filter((m) => {
    if (kind && m.kind !== kind) return false;
    if (!q) return true;
    return [m.name, m.id, m.provider, m.selector].some((v) => (v || "").toLowerCase().includes(q));
  });
}

export function kindCounts(models = []) {
  const counts = {};
  for (const m of models) counts[m.kind || "chat"] = (counts[m.kind || "chat"] || 0) + 1;
  return counts;
}

// 1000000 → "1M", 128000 → "128K", 0 or absent → "".
export function formatContext(n) {
  if (!n || n <= 0) return "";
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(n % 1_000_000 === 0 ? 0 : 1).replace(/\.0$/, "") + "M";
  if (n >= 1000) return Math.round(n / 1000) + "K";
  return String(n);
}

// The vendor's price pair in USD per million tokens. A model with no price is
// shown blank, never as "$0" — free and unknown are different answers.
export function formatPrice(cost) {
  if (!cost || typeof cost.input !== "number" || typeof cost.output !== "number") return "";
  // 0 / 0 is what the catalog holds both for a local model and for a hosted
  // one it has no price for (OpenAI's image model reads 0 / 0 on 18.2.8), so
  // it says nothing rather than "free".
  if (cost.input === 0 && cost.output === 0) return "";
  // Whole dollars print whole ($3, $15); anything else keeps two decimals
  // ($0.30, $1.20, $2.50) so a column of prices lines up by the cent.
  const f = (v) => "$" + (Number.isInteger(v) ? String(v) : v.toFixed(2));
  return f(cost.input) + " / " + f(cost.output);
}

// Allowed entries that are not one of the reported selectors: globs, fuzzy
// names, or a model the CLI no longer reaches. They are listed as written,
// because deciding what a glob or a fuzzy name matches is the CLI's job.
export function patternsNotListed(allowed = [], models = []) {
  const selectors = new Set(models.map((m) => m.selector));
  return allowed.filter((a) => !selectors.has(a));
}

// An allowed entry that looks like a pattern (a glob) is the CLI's to match;
// one that looks like an exact selector and is not reported is unreachable.
const GLOB = /[*?[\]{}]/;
export function splitAllowedExtras(extras = []) {
  const patterns = extras.filter((e) => GLOB.test(e) || !e.includes("/"));
  const unreachable = extras.filter((e) => !patterns.includes(e));
  return { patterns, unreachable };
}

// omp uses only the models the allowed list matches, and none matching means
// no usable model (`resolveAllowedModels`, issue #1022 in its own comments).
// PiCode can say so only when every entry is an exact selector it can check:
// a glob or a fuzzy name might still match, and guessing would be a lie.
export function allowedLeavesNothing(allowed = [], models = []) {
  if (!allowed.length) return false;
  const { patterns, unreachable } = splitAllowedExtras(patternsNotListed(allowed, models));
  return patterns.length === 0 && unreachable.length === allowed.length;
}

// Kind names in words a reader knows: the same words omp gives the roles that
// pick a model by kind (Speech, Dictation, Web search, Judge).
const KIND_LABELS = {
  chat: "Chat", tiny: "Small", image: "Image", tts: "Speech", stt: "Dictation",
  search: "Web search", embedding: "Embedding", rerank: "Rerank", judge: "Judge", video: "Video",
};
export function kindLabel(kind) {
  return KIND_LABELS[kind] || kind || "";
}
