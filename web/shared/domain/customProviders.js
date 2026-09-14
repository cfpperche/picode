// Custom provider endpoints (ADR-0129): shared form plumbing for both apps.
// The schema lives in contracts/schemas.js; these helpers move values
// between the form, the PUT payload and the catalog row.

import {
  parseForm, customProviderSchema, customModelIds, THINKING_LEVELS, DEFAULT_THINKING_LEVELS,
} from "../contracts/schemas.js";

export { customModelIds, THINKING_LEVELS, DEFAULT_THINKING_LEVELS };

// validateCustomProvider runs the schema. Returns { ok, value, error } like
// parseForm; value carries the form shapes (payload building is separate).
export function validateCustomProvider(form, { takenIds = [], requireKey = true } = {}) {
  return parseForm(customProviderSchema({ takenIds, requireKey }), form);
}

// customProviderPayload builds the PUT /api/providers/custom/{id} body from
// a validated form value. The key is included only when non-empty, so an
// Edit with a blank key keeps the stored credential.
export function customProviderPayload(v) {
  const contextWindow = String(v.contextWindow || "").trim();
  const maxTokens = String(v.maxTokens || "").trim();
  const key = String(v.key || "").trim();
  // A reasoning model gets reasoning:true plus the levels it exposes;
  // everything else gets an explicit false so the form stays the owner of
  // the fields it shows. Levels are stored in pi's scale order.
  const levels = v.reasoningModel
    ? THINKING_LEVELS.filter((l) => v.thinkingLevels.includes(l))
    : [];
  return {
    baseUrl: v.baseUrl,
    api: v.api,
    compat: { supportsDeveloperRole: !!v.compatDeveloper, supportsReasoningEffort: !!v.compatReasoning },
    // Empty means pi's default for the API type: the key is left out of the
    // entry rather than written as an empty string.
    ...(v.thinkingFormat ? { thinkingFormat: v.thinkingFormat } : {}),
    models: customModelIds(v.modelsText).map((id) => ({
      id,
      ...(contextWindow ? { contextWindow: Number(contextWindow) } : {}),
      ...(maxTokens ? { maxTokens: Number(maxTokens) } : {}),
      reasoning: !!v.reasoningModel,
      ...(levels.length ? { thinkingLevels: levels } : {}),
    })),
    ...(key ? { key } : {}),
  };
}

// customProviderForm prefills the form from a catalog row (Edit) or starts
// a blank one (Add). Context/max apply to every listed model, so the first
// definition that carries them is the honest prefill.
export function customProviderForm(provider) {
  const p = provider || {};
  const defs = Array.isArray(p.definitions) ? p.definitions : [];
  const sized = defs.find((m) => m.contextWindow || m.maxTokens) || {};
  return {
    id: p.id || "",
    baseUrl: p.baseUrl || "",
    api: p.api || "openai-completions",
    modelsText: defs.map((m) => m.id).join("\n"),
    contextWindow: sized.contextWindow ? String(sized.contextWindow) : "",
    maxTokens: sized.maxTokens ? String(sized.maxTokens) : "",
    compatDeveloper: !!(p.compat && p.compat.supportsDeveloperRole),
    compatReasoning: !!(p.compat && p.compat.supportsReasoningEffort),
    thinkingFormat: p.thinkingFormat || "",
    reasoningModel: defs.some((m) => m && m.reasoning),
    thinkingLevels: customThinkingLevels(defs),
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

// agreedModelLimits answers the one context window and max output the form can
// honestly write for a whole list: a number every listed model reports and
// every one agrees on. One model is a special case of that rule, and a list
// where they differ returns nothing — writing the largest window onto a model
// that has a smaller one would be a lie pi then acts on.
export function agreedModelLimits(found) {
  const models = (found || []).filter((m) => m && m.id);
  if (!models.length) return {};
  const agreed = (key) => {
    const first = Number(models[0][key]) || 0;
    if (!first) return null;
    for (const m of models) {
      if ((Number(m[key]) || 0) !== first) return null;
    }
    return first;
  };
  const out = {};
  const contextWindow = agreed("contextWindow");
  if (contextWindow) out.contextWindow = contextWindow;
  const maxTokens = agreed("maxTokens");
  if (maxTokens) out.maxTokens = maxTokens;
  return out;
}

// customTakenIds lists the ids a new definition may not claim: every
// built-in provider in the catalog plus every other custom definition.
export function customTakenIds(catalogProviders, exceptId) {
  const all = (catalogProviders || []).map((p) => p.id);
  return all.filter((id) => id !== exceptId);
}
