import { piFields, PI_TOOLS } from "./piRows.js";

export { PI_TOOLS };

// What one layer of Pi settings shows, derived from the row table.
//
// This used to be a hand-written list of keys beside a hand-written list of
// rows, so a setting needed adding in two places and a key added to only one
// of them rendered empty — which is exactly how the Theme row shipped
// (2026-09-20). Both now come from `piRows.js`: a row is one line, and this
// function cannot fall behind it.
//
// The rule per key: if this layer sets it, its value is what runs. If it does
// not, the parent's value runs, and where the parent has nothing the field's
// declared `unset` is what pi itself does.
export function resolveLayer(layer, parent) {
  const has = (layer && layer.has) || {};
  const l = layer || {};
  const p = parent || {};
  const out = {};
  for (const field of piFields()) {
    const { key, type, unset } = field;
    if (has[key]) {
      // Set in this layer: its value stands, even when that value is the
      // type's zero — an explicitly empty tool list means no tools, not the
      // default set.
      out[key] = own(l[key], type);
      continue;
    }
    out[key] = inherited(p[key], type, unset);
  }
  return out;
}

function own(value, type) {
  if (type === "bool") return !!value;
  if (type === "list") return Array.isArray(value) ? value : [];
  return value === undefined || value === null ? "" : value;
}

function inherited(value, type, unset) {
  if (type === "bool") return typeof value === "boolean" ? value : !!unset;
  if (type === "list") return Array.isArray(value) && value.length ? value : (Array.isArray(value) ? value : unset);
  return value === undefined || value === null || value === "" ? unset : value;
}

// catalogBase is the floor every layer inherits from: what pi does with a
// settings file that sets nothing, plus the first provider and model the
// catalog offers so the Defaults row has something to show before a choice.
export function catalogBase(catalog) {
  const provider = catalog && catalog.providers && catalog.providers[0];
  const model = provider && provider.models && provider.models[0];
  const out = {};
  for (const { key, type, unset } of piFields()) {
    out[key] = type === "bool" ? !!unset : type === "list" ? (unset || []) : unset;
  }
  out.defaultProvider = provider ? provider.id : "";
  out.defaultModel = model ? model.id : "";
  // pi's own thinking default, kept here rather than in the table: it is the
  // catalog's floor, not the value pi writes when the key is absent.
  out.defaultThinkingLevel = "medium";
  return out;
}
