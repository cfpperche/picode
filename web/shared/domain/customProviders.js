// Custom provider endpoints (ADR-0129): shared form plumbing for both apps.
// The schema lives in contracts/schemas.js; these helpers move values
// between the form, the PUT payload and the catalog row.

import { parseForm, customProviderSchema, customModelIds } from "../contracts/schemas.js";

export { customModelIds };

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
  return {
    baseUrl: v.baseUrl,
    api: v.api,
    compat: { supportsDeveloperRole: !!v.compatDeveloper, supportsReasoningEffort: !!v.compatReasoning },
    models: customModelIds(v.modelsText).map((id) => ({
      id,
      ...(contextWindow ? { contextWindow: Number(contextWindow) } : {}),
      ...(maxTokens ? { maxTokens: Number(maxTokens) } : {}),
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
    key: "",
  };
}

// customTakenIds lists the ids a new definition may not claim: every
// built-in provider in the catalog plus every other custom definition.
export function customTakenIds(catalogProviders, exceptId) {
  const all = (catalogProviders || []).map((p) => p.id);
  return all.filter((id) => id !== exceptId);
}
