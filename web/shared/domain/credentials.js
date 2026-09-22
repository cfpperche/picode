// The credential roster's vocabulary (ADR-0165). The eight guest agent CLIs
// share one vault, and this module is the one place the pane's words for it
// live: a provider id becomes the vendor's name, a row's origin becomes a
// chip, and a Verify answer becomes a label with a tone. Pure — no React, no
// network — so both apps say the same thing (ADR-0072).
//
// The id is the row's real identity (the server, pi's own /login set and the
// environment variables all use it); these are only the names a person reads.
import { normalizeTerminalCli, terminalCliLabel } from "./terminalCli.js";
import { formatAge } from "./providerRows.js";

// PROVIDER_NAMES is the vault vocabulary spoken out loud. An id with no entry
// is shown as itself: a guess would be worse than "openrouter".
const PROVIDER_NAMES = Object.freeze({
  anthropic: "Anthropic",
  openai: "OpenAI",
  "openai-codex": "ChatGPT (Codex)",
  xai: "xAI",
  google: "Google Gemini",
  "meta-ai": "Meta AI",
  meta: "Meta AI",
  muse: "Muse Code",
  openrouter: "OpenRouter",
  "github-copilot": "GitHub Copilot",
  opencode: "OpenCode",
  zai: "Z.ai",
  "kimi-coding": "Kimi (Moonshot)",
  moonshot: "Moonshot",
  minimax: "MiniMax",
  "llama.cpp": "llama.cpp",
  deepseek: "DeepSeek",
  mistral: "Mistral",
  groq: "Groq",
  cerebras: "Cerebras",
  nvidia: "NVIDIA",
  huggingface: "Hugging Face",
  "amazon-bedrock": "Amazon Bedrock",
  together: "Together AI",
  fireworks: "Fireworks AI",
});

// name is the CLI's own name for a provider its catalog lists (Omp's roster
// carries one per row): it answers where the vocabulary is silent, before the
// bare id.
export function providerName(id, name) {
  const key = String(id || "").trim();
  if (!key) return "";
  return PROVIDER_NAMES[key] || PROVIDER_NAMES[key.toLowerCase()] || String(name || "").trim() || key;
}

// The two shapes a vault row can hold, in the words the plan settled on: an
// OAuth login is a subscription, a pasted key is an API key. A shape we do not
// know keeps its own name rather than being filed under one of the two.
const KINDS = Object.freeze({ oauth: "Subscription", api_key: "API key" });

export function kindLabel(type) {
  const key = String(type || "").trim().toLowerCase();
  if (!key) return "";
  return KINDS[key] || key;
}

// sourceLabel is where a row came from: added here, read out of a CLI's own
// store, or absorbed from ADR-0013's accounts.json. An empty origin says
// nothing — older rows have none, and a chip would invent one.
export function sourceLabel(origin) {
  const value = String(origin || "").trim();
  if (!value) return "";
  if (value === "vault") return "Vault";
  if (value === "migrated") return "Migrated";
  if (value.startsWith("imported:")) {
    const cli = value.slice("imported:".length).trim();
    if (!cli) return "From a CLI";
    return "From " + (normalizeTerminalCli(cli) ? terminalCliLabel(cli) : cli);
  }
  return value;
}

// Verify spends one listing call against the provider, so the control that
// spends it says so before the click (ADR-0129's amendment).
export const VERIFY_LABEL = "Verify with the provider (1 request)";

// HEALTH is what a Verify answer can be. `unknown` is not a failure and not a
// pass: it is "we did not learn" — the row keeps whatever the check said in
// words, and the chip says the rest.
const HEALTH = Object.freeze({
  ok: { label: "Works", tone: "ok" },
  invalid: { label: "Key refused", tone: "danger" },
  expired: { label: "Expired", tone: "danger" },
  no_credit: { label: "No credit", tone: "danger" },
  rate_limited: { label: "Rate limited", tone: "warn" },
  unknown: { label: "Unverified", tone: "muted" },
});

// healthChip turns a row's health record into a chip, or nothing when the row
// has never been checked. `age` is how long ago the check ran ("now" / "4m").
export function healthChip(health, nowMs = Date.now()) {
  const state = String((health && health.state) || "").trim();
  if (!state) return null;
  const key = state.toLowerCase();
  const known = HEALTH[key] || { label: state, tone: "muted" };
  const at = Date.parse(String((health && health.at) || ""));
  return {
    label: known.label,
    tone: known.tone,
    state: key,
    message: String((health && health.message) || "").trim(),
    age: Number.isFinite(at) ? formatAge((Number(nowMs) - at) / 1000) : "",
  };
}

// orderAccounts orders one provider's rows: the accounts in play first, then
// by the name the person gave them, with the id as the tie-break so two rows
// never swap places between renders. `active` is deliberately not used: for a
// guest CLI's provider the vault has no active slot, so sorting by one would
// invent a meaning the API does not have.
export function orderAccounts(accounts) {
  return [...(accounts || [])].sort((a, b) => {
    const paused = (a.paused ? 1 : 0) - (b.paused ? 1 : 0);
    if (paused) return paused;
    const an = String((a && a.label) || "").toLocaleLowerCase();
    const bn = String((b && b.label) || "").toLocaleLowerCase();
    if (an !== bn) return an < bn ? -1 : 1;
    return String((a && a.id) || "").localeCompare(String((b && b.id) || ""));
  });
}

// vaultProblemText turns the store's own error into the sentence the pane
// shows. The vault's two unreadable states are both "put a file back", so the
// line names the file and the action; anything else is passed through rather
// than paraphrased into something we cannot promise.
export function vaultProblemText(problem = "") {
  const text = String(problem || "");
  if (/key is missing/i.test(text)) {
    return "This machine’s vault can’t be read: its key file (credentials.key) is missing. Put it back beside credentials.json to see the accounts.";
  }
  if (/not a vault key|cannot be read/i.test(text)) {
    return "This machine’s vault can’t be read: that key file does not open it. Restore the key file that matches this vault, or a backup taken from this machine.";
  }
  return "This machine’s vault can’t be read. " + text;
}
