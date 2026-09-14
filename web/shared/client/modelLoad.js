// Loading an endpoint's model list (open topic P1).
//
// Framework-free on purpose: web/shared holds no React (the two apps own their
// rendering), so this file is the request, the address line and the pure
// outcome — what changed in the form and the one line to show for it. Both
// dialogs call the same code, which is what keeps browser and phone identical.

import { api } from "./api.js";
import { mergeModelIds, agreedModelLimits } from "../domain/customProviders.js";

// loadModelsFor asks the server for the ids a form's endpoint serves. The key
// travels once when the person typed one; otherwise the server reads the saved
// credential for `id`. The key never comes back.
export async function loadModelsFor({ baseUrl, api: apiType, key, id }) {
  const trimmed = String(key || "").trim();
  return api("/api/providers/custom/models", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      baseUrl: String(baseUrl || "").trim(),
      api: apiType || "openai-completions",
      ...(trimmed ? { key: trimmed } : {}),
      ...(id ? { id } : {}),
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
  const limits = agreedModelLimits(found);
  const changes = { modelsText: merged.text };
  const filled = [];
  if (limits.contextWindow && !String(form.contextWindow || "").trim()) {
    changes.contextWindow = String(limits.contextWindow);
    filled.push("context window");
  }
  if (limits.maxTokens && !String(form.maxTokens || "").trim()) {
    changes.maxTokens = String(limits.maxTokens);
    filled.push("max output");
  }
  // A list whose models disagree on limits says so instead of leaving a blank
  // field unexplained: one shared value cannot be true for all of them, and
  // the form writes one value for the whole list.
  const reported = found.some((m) => Number(m && m.contextWindow) > 0 || Number(m && m.maxTokens) > 0);
  const mixed = reported && !filled.length && !String(form.contextWindow || "").trim() && !String(form.maxTokens || "").trim();
  return { changes, line: doneLine(found.length, merged.added, filled, mixed) };
}

// doneLine reports what actually happened, including the honest nothing: an
// endpoint that lists no models is a real answer, not a failure.
export function doneLine(found, added, filled, mixed) {
  if (!found) return "The endpoint reports no models — check the API type, or type the ids by hand.";
  const parts = [
    "Found " + found + (found === 1 ? " model" : " models") + ".",
    added ? "Added " + added + " to the list." : "Everything it lists is already here.",
  ];
  if (filled.length) parts.push("Filled " + filled.join(" and ") + " from the endpoint.");
  if (mixed) parts.push("It reports different limits per model, so those were left blank.");
  return parts.join(" ");
}