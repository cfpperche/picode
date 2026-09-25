// One scope vocabulary for every setup pane's address (2026-09-24): Global,
// the workspace, one agent — `?scope=global|workspace|agent`. Each pane still
// speaks its own words inside (the APIs and the CLIs' files name them), and
// the table below is the only place those words meet the address. An address
// carries only the shared words (the older words and `?layer=` were retired
// on 2026-09-25); the table still turns a pane's own word into one.
export const SCOPES = ["global", "workspace", "agent"];

const KINDS = {
  global: "global", machine: "global", user: "global",
  workspace: "workspace", project: "workspace", local: "workspace",
  agent: "agent",
};

// The shared word for any pane's scope word; "" when it is none of the three.
export function scopeKind(scope) {
  return KINDS[String(scope || "").toLowerCase()] || "";
}

// Each pane's own words, by shared word.
export const PACKAGE_WORDS = { global: "user", workspace: "project", agent: "agent" };
export const SKILL_WORDS = { global: "machine", workspace: "workspace", agent: "agent" };
export const LAYER_WORDS = { global: "global", workspace: "project", agent: "agent" };
export const MODEL_WORDS = { global: "global", workspace: "project" };
export const MEMORY_WORDS = { global: "global", workspace: "workspace" };

// The scope an address names, in the pane's words. `kind` is the shared word,
// set only when the address named one, so a pane's default never travels to
// the next pane as if chosen. A word that is not a shared one is invalid.
export function readScope(params, words) {
  const raw = params.get("scope");
  if (raw === null || raw === "") return { value: "", kind: "", invalid: false };
  const kind = SCOPES.includes(raw) ? raw : "";
  const value = (kind && words[kind]) || "";
  return { value, kind: value ? kind : "", invalid: !value };
}

// Writes a scope (any pane's word) to an address in the shared word. Global
// is a pane's default and is left out unless the pane keeps it explicit.
export function writeScope(query, scope, { keepGlobal = false } = {}) {
  const kind = scopeKind(scope);
  if (kind && (kind !== "global" || keepGlobal)) query.set("scope", kind);
}
