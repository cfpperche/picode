// The tri-state a terminal settings field is in, and how to label it.
//
// The whole panel turns on one distinction: a field is *inherited* when it is
// absent from this scope's stored values — not when it happens to equal what
// it would inherit. The two look identical on screen and behave differently
// the moment the global changes, which is exactly the bug this file exists to
// keep out of the component.

export const INHERIT = null;

// selectedKey returns the segment to show as chosen: INHERIT when this scope
// stores nothing for the flag, otherwise the stored value.
export function selectedKey(values, flagKey) {
  if (!values || !Object.prototype.hasOwnProperty.call(values, flagKey)) return INHERIT;
  return values[flagKey];
}

function valueLabel(value) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

// choicesFor builds the segments for one flag. The first is always the
// inherit-shaped one, and it carries the value it would fall back to so the
// user can see what choosing it does before choosing it.
export function choicesFor(flag, inherited, isGlobal) {
  const from = inherited && inherited[flag.key];
  const head = isGlobal ? "Default" : "Inherit";
  return [
    { key: INHERIT, label: from ? `${head} (${valueLabel(from)})` : head, inherit: true },
    ...(flag.values || []).map((v) => ({ key: v, label: valueLabel(v), inherit: false })),
  ];
}

// EFFECT_TEXT says when a change takes hold. A panel that leaves this out
// looks broken the first time a flag only applies to new panes.
export const EFFECT_TEXT = {
  live: "Takes effect right away, including on terminals that are already open.",
  "new-panes": "Takes effect on terminals opened from now on.",
};

export function effectText(effect) {
  return EFFECT_TEXT[effect] || "";
}

// isOverridden is what the row's provenance chip reads from.
export function isOverridden(values, flagKey) {
  return selectedKey(values, flagKey) !== INHERIT;
}

// withChoice applies one segment click to a stored values map, mirroring what
// the server does with the same patch. Choosing inherit REMOVES the key — the
// same rule as the server's Apply, and for the same reason: storing the value
// it would have inherited pins it, and the field silently stops following the
// default it is still labelled as following.
export function withChoice(values, flagKey, value) {
  const next = { ...(values || {}) };
  if (value === INHERIT) delete next[flagKey];
  else next[flagKey] = value;
  return next;
}

// ---- full-catalog page helpers (ADR-0024, open catalog) --------------------

// inheritedValueFor: what a field falls back to when this scope stores
// nothing. For curated flags the settings payload's `inherited` map already
// carries it; for the rest of the catalog the running tmux's global value is
// the truth. The layering is: tmux default/global ← PiCode global ← override.
export function inheritedValueFor(name, inherited, catalogValue) {
  if (inherited && Object.prototype.hasOwnProperty.call(inherited, name)) return inherited[name];
  return catalogValue ?? "";
}

// matchesQuery: the search box over 150+ options. Name match is what people
// type; danger text is included so "clipboard" finds set-clipboard's warning.
export function matchesQuery(row, query) {
  const q = String(query || "").trim().toLowerCase();
  if (!q) return true;
  return row.name.toLowerCase().includes(q) || String(row.danger || "").toLowerCase().includes(q);
}

// groupCatalog splits catalog rows for one page: curated rows are rendered by
// the featured section, so they are excluded here; server rows are only
// offered on the global page, labelled machine-wide.
export function groupCatalog(rows, { isGlobal }) {
  const perTerminal = [];
  const server = [];
  for (const row of rows || []) {
    if (row.curated) continue;
    if (row.scope === "server") {
      if (isGlobal) server.push(row);
      continue;
    }
    perTerminal.push(row);
  }
  return { perTerminal, server };
}
