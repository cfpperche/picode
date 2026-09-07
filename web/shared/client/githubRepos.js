import { api } from "./api.js";

// Repository rows for the clone picker (New workspace → Clone repository).
// The server answers from the machine's own gh CLI: {available, reason?,
// repos?}. This module maps that onto picker states so both shells render
// the same one-line-with-action feedback.
//
// Phases:
// - "ready"      → repos[] (may be empty; the picker shows its own empty line)
// - "blocked"    → a visible reason the list can't exist here, with kind
//                  "install" | "login" telling which action applies
// - "error"      → transient failure (network, timeout); Retry applies
export async function fetchGithubRepos(refresh = false) {
  let data;
  try {
    data = await api("/api/github/repos" + (refresh ? "?refresh=1" : ""));
  } catch (err) {
    return { phase: "error", reason: humanReason(err) };
  }
  if (data && data.available) return { phase: "ready", repos: data.repos || [] };
  const reason = (data && data.reason) || "Could not load repositories.";
  if (/not installed/i.test(reason)) return { phase: "blocked", kind: "install", reason };
  if (/not logged in/i.test(reason)) return { phase: "blocked", kind: "login", reason };
  return { phase: "error", reason };
}

function humanReason(err) {
  const msg = err && err.message ? err.message : "";
  return msg || "Could not load repositories.";
}
