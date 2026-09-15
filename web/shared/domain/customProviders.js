// Custom provider endpoints (ADR-0129): shared form plumbing for both apps.
// The schema lives in contracts/schemas.js; these helpers move values
// between the form, the PUT payload and the catalog row.

import {
  parseForm, customProviderSchema, customModelIds, THINKING_LEVELS, DEFAULT_THINKING_LEVELS,
  THINKING_FORMAT_NEEDS, chatTemplateObject, CUSTOM_INPUT_MODALITIES,
} from "../contracts/schemas.js";

export { customModelIds, THINKING_LEVELS, DEFAULT_THINKING_LEVELS, THINKING_FORMAT_NEEDS, CUSTOM_INPUT_MODALITIES };  // re-exported: the pages import both from here

// validateCustomProvider runs the schema. Returns { ok, value, error } like
// parseForm; value carries the form shapes (payload building is separate).
export function validateCustomProvider(form, { takenIds = [], requireKey = true } = {}) {
  return parseForm(customProviderSchema({ takenIds, requireKey }), form);
}

// customProviderPayload builds the PUT /api/providers/custom/{id} body from
// a validated form value. The key is included only when non-empty, so an
// Edit with a blank key keeps the stored credential.
export function customProviderPayload(v) {
  const key = String(v.key || "").trim();
  // A reasoning model gets reasoning:true plus the levels it exposes;
  // everything else gets an explicit false so the form stays the owner of
  // the fields it shows. Levels are stored in pi's scale order.
  const levels = v.reasoningModel
    ? THINKING_LEVELS.filter((l) => v.thinkingLevels.includes(l))
    : [];
  // compat.thinkingFormat is pi's own value ("openai" for the reasoning_effort
  // branch); the empty choice leaves the key out so pi picks for the API type.
  // The chat-template object rides only with the format that reads it.
  const kwargs = THINKING_FORMAT_NEEDS[v.thinkingFormat] === "kwargs" ? chatTemplateObject(v.chatTemplateKwargs) : null;
  const args = THINKING_FORMAT_NEEDS[v.thinkingFormat] === "args" ? chatTemplateObject(v.chatTemplateArgs) : null;
  // Level values ride only for selected levels: a filled value overrides the
  // provider string, a blank one keeps the level's own name server-side.
  const levelValues = {};
  for (const l of levels) {
    const value = String((v.thinkingLevelValues || {})[l] || "").trim();
    if (value) levelValues[l] = value;
  }
  return {
    baseUrl: v.baseUrl,
    api: v.api,
    compat: { supportsDeveloperRole: !!v.compatDeveloper, supportsReasoningEffort: !!v.compatReasoning, supportsUsageInStreaming: !!v.compatStreaming },
    ...(v.thinkingFormat ? { thinkingFormat: v.thinkingFormat } : {}),
    ...(kwargs ? { chatTemplateKwargs: kwargs } : {}),
    ...(args ? { chatTemplateArgs: args } : {}),
    models: customModelIds(v.modelsText).map((id) => {
      const row = modelLimitRow(v.modelLimits, id);
      const name = String(row.name || "").trim();
      const input = CUSTOM_INPUT_MODALITIES.filter((m) => (row.input || []).includes(m));
      const rates = ["input", "output", "cacheRead", "cacheWrite"].map((k) => String((row.cost || {})[k] || "").trim());
      return {
        id,
        ...(name ? { name } : {}),
        ...(input.length ? { input } : {}),
        ...(rates.every(Boolean) ? { cost: { input: Number(rates[0]), output: Number(rates[1]), cacheRead: Number(rates[2]), cacheWrite: Number(rates[3]) } } : {}),
        ...(row.contextWindow ? { contextWindow: Number(row.contextWindow) } : {}),
        ...(row.maxTokens ? { maxTokens: Number(row.maxTokens) } : {}),
        reasoning: !!v.reasoningModel,
        ...(levels.length ? { thinkingLevels: levels } : {}),
        ...(Object.keys(levelValues).length ? { thinkingLevelValues: levelValues } : {}),
      };
    }),
    ...(key ? { key } : {}),
  };
}

// customProviderForm prefills the form from a catalog row (Edit) or starts
// a blank one (Add). Context/max apply to every listed model, so the first
// definition that carries them is the honest prefill.
export function customProviderForm(provider) {
  const p = provider || {};
  const defs = Array.isArray(p.definitions) ? p.definitions : [];
  return {
    id: p.id || "",
    baseUrl: p.baseUrl || "",
    api: p.api || "openai-completions",
    modelsText: defs.map((m) => m.id).join("\n"),
    modelLimits: limitsFromDefinitions(defs),
    compatDeveloper: !!(p.compat && p.compat.supportsDeveloperRole),
    compatReasoning: !!(p.compat && p.compat.supportsReasoningEffort),
    compatStreaming: !!(p.compat && p.compat.supportsUsageInStreaming),
    thinkingFormat: customThinkingFormat(p.thinkingFormat),
    chatTemplateKwargs: jsonText(p.chatTemplateKwargs),
    chatTemplateArgs: jsonText(p.chatTemplateArgs),
    reasoningModel: defs.some((m) => m && m.reasoning),
    thinkingLevels: customThinkingLevels(defs),
    thinkingLevelValues: customThinkingLevelValues(defs),
    key: "",
  };
}

// customThinkingLevels prefills the level row from what the file says. A
// model with an explicit map reports exactly the levels that map claims; a
// reasoning model without one gets pi's default set (through high) so an
// untouched Edit writes back the same shape it read.
export function customThinkingLevels(defs) {
  for (const m of defs) {
    if (m && Array.isArray(m.thinkingLevels) && m.thinkingLevels.length) {
      return THINKING_LEVELS.filter((l) => m.thinkingLevels.includes(l));
    }
    if (m && m.thinkingLevelMap && typeof m.thinkingLevelMap === "object") {
      return THINKING_LEVELS.filter((l) => typeof m.thinkingLevelMap[l] === "string");
    }
  }
  return [...DEFAULT_THINKING_LEVELS];
}

// customThinkingLevelValues prefills the per-level provider strings: only
// values that differ from the level name show (xhigh -> "high"); identity
// values read as blank, and unmanaged keys ("off") are not shown — the
// server keeps them untouched.
export function customThinkingLevelValues(defs) {
  for (const m of defs || []) {
    if (m && m.thinkingLevelMap && typeof m.thinkingLevelMap === "object") {
      const out = {};
      for (const l of THINKING_LEVELS) {
        const value = m.thinkingLevelMap[l];
        if (typeof value === "string" && value !== "" && value !== l) out[l] = value;
      }
      return out;
    }
  }
  return {};
}

// mergeModelIds folds what an endpoint listed into the ids a person already
// typed: ids already there stay where they are, new ones are appended in the
// endpoint's order, and nothing is ever removed or reordered. Returns the new
// text plus how many ids it added (the dialog reports the count); a list that
// adds nothing is not an error, it just has nothing to say.
export function mergeModelIds(text, found) {
  const lines = String(text || "").split("\n").map((l) => l.trim()).filter(Boolean);
  const have = new Set(lines);
  let added = 0;
  for (const m of found || []) {
    const id = String((m && m.id) || "").trim();
    if (!id || have.has(id)) continue;
    have.add(id);
    lines.push(id);
    added++;
  }
  return { text: lines.join("\n"), added };
}

// limitsFromDefinitions reads what the file says per model: each id carries
// its own context window and max output, which is what pi supports (the form
// used to write one number for the whole list, which had to refuse a gateway
// whose models disagree). Name, input modalities and cost ride the same row;
// numbers read back as the strings the inputs hold, zeros included.
export function limitsFromDefinitions(defs) {
  const out = {};
  for (const m of defs || []) {
    if (!m || !m.id) continue;
    out[m.id] = {
      contextWindow: m.contextWindow ? String(m.contextWindow) : "",
      maxTokens: m.maxTokens ? String(m.maxTokens) : "",
      name: typeof m.name === "string" ? m.name : "",
      input: Array.isArray(m.input) ? CUSTOM_INPUT_MODALITIES.filter((mod) => m.input.includes(mod)) : [],
      cost: {
        input: costText(m.cost && m.cost.input),
        output: costText(m.cost && m.cost.output),
        cacheRead: costText(m.cost && m.cost.cacheRead),
        cacheWrite: costText(m.cost && m.cost.cacheWrite),
      },
    };
  }
  return out;
}

function costText(value) {
  return typeof value === "number" && Number.isFinite(value) ? String(value) : "";
}

// syncModelLimits keeps the rows in step with the id list: one row per id, in
// the list's order, none for an id that is gone. Numbers already typed survive
// a trip through the textarea.
export function syncModelLimits(limits, ids) {
  const out = {};
  for (const id of ids || []) out[id] = modelLimitRow(limits, id);
  return out;
}

// modelLimitRow is one row, defaulted, so callers never check for existence.
export function modelLimitRow(limits, id) {
  const cur = (limits || {})[id] || {};
  const cost = (cur && cur.cost) || {};
  return {
    contextWindow: String(cur.contextWindow || ""),
    maxTokens: String(cur.maxTokens || ""),
    name: String(cur.name || ""),
    input: Array.isArray(cur.input) ? cur.input.filter((m) => CUSTOM_INPUT_MODALITIES.includes(m)) : [],
    cost: {
      input: String(cost.input || ""),
      output: String(cost.output || ""),
      cacheRead: String(cost.cacheRead || ""),
      cacheWrite: String(cost.cacheWrite || ""),
    },
  };
}

// jsonText renders a stored compat object back into the editor, indented so a
// person can read and edit it.
function jsonText(value) {
  if (!value || typeof value !== "object") return "";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return "";
  }
}

// customThinkingFormat prefills the select from the file. A hand-edited value
// the form does not offer (or the undocumented "reasoning_effort", which pi
// sends down the same branch as "openai") reads as the format pi will actually
// take, and the next save writes it explicitly.
export function customThinkingFormat(value) {
  const v = String(value || "");
  if (!v) return "";
  if (v === "reasoning_effort") return "openai";
  return THINKING_FORMAT_NEEDS[v] === undefined ? "" : v;
}

// customTakenIds lists the ids a new definition may not claim: every
// built-in provider in the catalog plus every other custom definition.
export function customTakenIds(catalogProviders, exceptId) {
  const all = (catalogProviders || []).map((p) => p.id);
  return all.filter((id) => id !== exceptId);
}
