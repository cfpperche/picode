// Native settings and native memory for the guest agent CLIs (ADR-0163).
//
// Pi is not here. It keeps its own editor, its own API and its own trust rules
// (ADR-0101); this module covers the eight CLIs whose settings PiCode edits by
// writing their own config files, and the memory view every CLI answers for —
// including the ones whose honest answer is "none".

// The CLIs with a settings declaration in internal/clisettings. The server is
// the authority; this list keeps the pane from asking for an editor that does
// not exist, and the test below keeps the two in step.
import { MEMORY_WORDS, readScope, writeScope } from "./scopes.js";
export const NATIVE_SETTINGS_CLIS = ["claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy", "omp"];

export const supportsNativeSettings = (id) => NATIVE_SETTINGS_CLIS.includes(id);

// The route says `layer`, the driver says `scope`. One vocabulary each, mapped
// here rather than in a component, so a link and a request cannot drift.
export function layerToScope(layer) {
  return layer === "project" ? "project" : "user";
}

export function scopeToLayer(scope) {
  return scope === "project" ? "project" : "global";
}

// A workspace layer is named, not generic: a pane bound to a workspace says
// its name where the vendor-declared label says "This workspace" — the rule
// Pi's pane has always followed (label: workspace.name) and Mcps' rows too.
// A pane bound to nothing keeps the generic word, because there is no name to
// say. The replacement is by words, not by scope, so a driver's suffix
// survives: Claude Code's "This workspace (local)" names the checkout its
// uncommitted file is kept in.
export function namedScope(label, workspaceName = "") {
  const name = String(workspaceName || "").trim();
  if (!name) return label;
  // A replacer function, not a replacement string: String.replace interprets
  // `$` sequences ($&, $', $`) in a string replacement, and a workspace name
  // may legally hold them (trimmed, ≤120 chars, no charset rule) — the name
  // must land literally, first occurrence only.
  return String(label || "").replace("This workspace", () => name);
}

// Every layer label in one report, renamed in one pass: a pane feeds its
// layers (settings, models) or memory stores through this once, so the layer
// switcher, the checked radio and the per-row provenance ("From …") all read
// the name. Layers the pane cannot name come back untouched.
export function namedLayers(layers = [], workspaceName = "") {
  const name = String(workspaceName || "").trim();
  if (!name || !Array.isArray(layers)) return layers;
  return layers.map((l) => (
    l && typeof l.label === "string" && l.label.includes("This workspace")
      ? { ...l, label: namedScope(l.label, name) }
      : l
  ));
}

// The layer the pane opens on: the one the route names when the CLI has it,
// otherwise the first layer the CLI declares.
export function defaultLayer(layers = [], routeLayer = "") {
  const wanted = layerToScope(routeLayer);
  if (routeLayer && layers.some((l) => l.scope === wanted)) return routeLayer;
  const first = layers[0];
  return first ? scopeToLayer(first.scope) : "global";
}

// Fields in declaration order, grouped by their declared group. A CLI that
// declares no group gets one unnamed section rather than a header saying
// nothing.
export function groupFields(fields = []) {
  const out = [];
  for (const field of fields) {
    // A row another pane owns (omp's allowed models and hidden providers live
    // in Models) is in the report for the revision, not for this pane.
    if (field.pane) continue;
    const name = field.group || "";
    let group = out.find((g) => g.name === name);
    if (!group) {
      group = { name, fields: [] };
      out.push(group);
    }
    group.fields.push(field);
  }
  return out;
}

// What one row shows: the value in force for this layer, whether this layer is
// the one that sets it, and where it comes from otherwise. `fallback` is the
// CLI's own behaviour when nobody sets the key — the honest answer to "what
// happens if I leave this alone".
export function rowState(field, layers = [], scope = "user") {
  const here = layers.find((l) => l.scope === scope);
  const setHere = !!here && Object.prototype.hasOwnProperty.call(here.values || {}, field.key);
  if (setHere) return { value: here.values[field.key], setHere: true, from: "" };
  // Layers arrive in the vendor's precedence order, last wins; a row this
  // layer does not set inherits from the nearest layer before it that does.
  const index = layers.findIndex((l) => l.scope === scope);
  for (let i = (index < 0 ? layers.length : index) - 1; i >= 0; i -= 1) {
    const layer = layers[i];
    if (layer && Object.prototype.hasOwnProperty.call(layer.values || {}, field.key)) {
      return { value: layer.values[field.key], setHere: false, from: layer.label };
    }
  }
  return { value: undefined, setHere: false, from: "" };
}

// The one-line warning a dangerous choice carries. Danger is named, never
// hidden (docs/benchmarks.md).
export function dangerNote(field, value) {
  if (!field.danger) return "";
  const dangerous = field.danger === "true" ? value === true : String(value) === field.danger;
  if (!dangerous) return "";
  // Each dangerous choice says what it specifically costs. One shared sentence
  // printed the same words twice inside one group (visual review 2026-09-20).
  return field.dangerNote || "This turns off a safety check.";
}

// Values arrive from a config file, so a number field can hold a string and a
// switch can hold anything. Coerce to the kind the field declares, and refuse
// to send a value the field cannot hold.
export function coerce(field, raw) {
  if (field.kind === "bool") return !!raw;
  if (field.kind === "number") {
    const n = Number(String(raw).trim());
    return Number.isFinite(n) ? n : null;
  }
  return String(raw);
}

// Memory item titles come from the file name; the pane shows the CLI's own
// frontmatter kind when a file declares one.
export const MEMORY_KIND_LABELS = {
  user: "About you",
  feedback: "Feedback",
  project: "Project",
  reference: "Reference",
};

export function memoryKindLabel(kind) {
  return MEMORY_KIND_LABELS[kind] || kind || "";
}

// One line for each tier that has nothing to list, plus the action that
// changes it. Chrome carries state and the next action, never an essay
// (.pi/skills/uiux-review).
export function memoryEmptyLine(report, store) {
  if (!report) return { line: "", action: "" };
  if (report.tier === "none" || report.tier === "unknown") {
    return { line: report.note || "", action: "" };
  }
  if (store && !store.resolved) return { line: store.note || "", action: report.toggle ? "settings" : "" };
  if (store && !store.exists) return { line: "Nothing remembered yet.", action: report.toggle ? "settings" : "" };
  return { line: "Nothing remembered yet.", action: report.toggle ? "settings" : "" };
}

export function cliMemoryHash(cli, { workspaceId = "", scope = "" } = {}) {
  const query = new URLSearchParams();
  if (workspaceId) query.set("workspaceId", workspaceId);
  writeScope(query, scope, { keepGlobal: true });
  return "#/clis/" + encodeURIComponent(cli || "") + "/memory" + (query.size ? "?" + query : "");
}

export function cliMemoryLocation(hash = "") {
  const [path, query = ""] = String(hash).replace(/^#/, "").split("?");
  const match = /^\/clis\/([^/]+)\/memory$/.exec(path);
  if (!match) return null;
  let cli = "";
  try {
    cli = decodeURIComponent(match[1]);
  } catch {
    cli = "";
  }
  const params = new URLSearchParams(query);
  const read = readScope(params, MEMORY_WORDS);
  return { view: "clis", pane: "memory", id: cli, workspaceId: params.get("workspaceId") || "", scope: read.value, ...(read.kind ? { scopeKind: read.kind } : {}) };
}

// ── Model roles (ADR-0181) ───────────────────────────────────────────────────
// A CLI that keeps a role → model-selector map reports its rows with the rest
// of its settings, so provenance, "Set here" and "Use inherited" are the same
// code as every other row. What is new is the shape of two values: a selector,
// which carries an optional thinking suffix, and a list.

export const isRoleField = (field) => field?.kind === "role";
export const isListField = (field) => field?.kind === "list";

// A selector is `provider/model-id` with an optional `:level` suffix, and the
// model id may itself carry slashes, dots and an `@upstream` routing suffix
// (omp's `parseModelString`). The split here is **for display only**: the pane
// shows the base in the picker and the level in its own control, and writing
// them back joins the exact two pieces it split, so a value PiCode did not
// understand survives a round trip unchanged.
export function splitSelector(value, levels = []) {
  const text = value === undefined || value === null ? "" : String(value);
  const colon = text.lastIndexOf(":");
  if (colon <= 0) return { base: text, level: "" };
  const suffix = text.slice(colon + 1);
  if (!levels.includes(suffix)) return { base: text, level: "" };
  return { base: text.slice(0, colon), level: suffix };
}

export function joinSelector(base, level) {
  const trimmed = (base || "").trim();
  if (!trimmed) return "";
  return level ? trimmed + ":" + level : trimmed;
}

// A list value arrives from a config file, so it can be a single string (which
// every CLI in this family accepts where a list belongs) or absent.
export function listValue(value) {
  if (Array.isArray(value)) return value.map((v) => String(v));
  if (typeof value === "string" && value !== "") return [value];
  return [];
}

// The role's place in the quick-switch cycle, 1-based, or 0 when it is not in
// it. omp's own hub draws this as `⟳ N` (`model-hub.ts`, "second stop of the
// ctrl+p cycle"), and the pane says the same thing in the same words.
export function cyclePosition(cycle, roleId) {
  return listValue(cycle).indexOf(roleId) + 1;
}

// Every selector any layer already uses, so the picker can offer what this
// machine is actually configured with before the CLI has been asked anything.
export function selectorsInUse(fields = [], layers = []) {
  const seen = new Set();
  for (const field of fields) {
    if (!isRoleField(field)) continue;
    for (const layer of layers) {
      const value = layer?.values?.[field.key];
      if (typeof value === "string" && value && !value.startsWith("@")) seen.add(value);
    }
  }
  return [...seen].sort();
}

// The aliases a role row accepts beside a model: every role is addressable as
// `@name`, and `*` is the CLI's own shorthand for the default role.
export function roleAliases(fields = []) {
  const out = ["*"];
  for (const field of fields) {
    if (isRoleField(field)) out.push("@" + field.key.split(".").pop());
  }
  return out;
}
