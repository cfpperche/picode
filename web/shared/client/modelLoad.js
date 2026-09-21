// Loading an endpoint's model list (open topic P1).
//
// Framework-free on purpose: web/shared holds no React (the two apps own their
// rendering), so this file is the request, the address line and the pure
// outcome — what changed in the form and the one line to show for it. Both
// dialogs call the same code, which is what keeps browser and phone identical.

import { api } from "./api.js";
import { customModelIds, mergeModelIds, syncModelLimits } from "../domain/customProviders.js";

// loadModelsFor asks the server for the ids a form's endpoint serves. The key
// travels once when the person typed one; otherwise the server reads the saved
// credential for `id` from the CLI's own file (cli picks which). The key never
// comes back.
export async function loadModelsFor({ baseUrl, api: apiType, key, id, cli }) {
  const trimmed = String(key || "").trim();
  return api("/api/providers/custom/models", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      baseUrl: String(baseUrl || "").trim(),
      api: apiType || "openai-completions",
      ...(trimmed ? { key: trimmed } : {}),
      ...(id ? { id } : {}),
      ...(cli ? { cli } : {}),
    }),
  });
}

// hostOf is what the loading line names: the address being asked.
export function hostOf(baseUrl) {
  try {
    return new URL(String(baseUrl || "").trim()).host;
  } catch {
    return "the endpoint";
  }
}

// loadingLine is the one line shown while the request is out. It names the
// address, so a slow gateway is visibly a slow gateway, not a stuck dialog.
export function loadingLine(baseUrl) {
  return "Asking " + hostOf(baseUrl) + " for its model list…";
}

// modelLoadChanges folds a successful answer into the form: the ids that were
// missing, plus the context window and max output every listed model agreed
// on — and only where the person has not already typed one (the form is
// theirs; the endpoint is an offer). Returns the changes to merge and the
// line that reports what really happened.
export function modelLoadChanges(res, form) {
  const found = (res && res.models) || [];
  const merged = mergeModelIds(form.modelsText, found);
  const limits = syncModelLimits(form.modelLimits, customModelIds(merged.text));
  // Each model keeps its own numbers, so a gateway that publishes different
  // limits per model is filled correctly instead of refused for disagreeing.
  // A number the person typed stays theirs.
  let filled = 0;
  for (const m of found) {
    const id = String((m && m.id) || "").trim();
    if (!id || !limits[id]) continue;
    const row = limits[id];
    if (Number(m.contextWindow) > 0 && !row.contextWindow) {
      row.contextWindow = String(m.contextWindow);
      filled++;
    }
    if (Number(m.maxTokens) > 0 && !row.maxTokens) {
      row.maxTokens = String(m.maxTokens);
      filled++;
    }
  }
  const changes = { modelsText: merged.text, modelLimits: limits };
  return { changes, line: doneLine(found.length, merged.added, filled) };
}

// doneLine reports what actually happened, including the honest nothing: an
// endpoint that lists no models is a real answer, not a failure.
export function doneLine(found, added, filled) {
  if (!found) return "The endpoint reports no models — check the API type, or type the ids by hand.";
  const parts = [
    "Found " + found + (found === 1 ? " model" : " models") + ".",
    added ? "Added " + added + " to the list." : "Everything it lists is already here.",
  ];
  if (filled) parts.push("Filled " + filled + (filled === 1 ? " limit" : " limits") + " from the endpoint.");
  return parts.join(" ");
}