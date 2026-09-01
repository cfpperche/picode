// One-shot handoff of an automation draft into the editor (ADR-0046 v2).
// sessionStorage, read once then removed — the PinStudio "picode-sketch"
// pattern. A template card, /automate, or anything else that wants the
// editor pre-filled writes here and navigates to #/automations/new.

const KEY = "picode-automation-draft";
const FIELDS = ["name", "prompt", "action", "targetAgentId", "workspaceId", "cron", "webhook",
  "maxCostUsd", "maxRuns", "maxRunsWindowMin", "provider", "model", "thinking", "source", "sourceLabel", "templateId"];
const WINDOWS = new Set([60, 1440, 10080]);

function str(v, max) {
  if (v == null) return "";
  const s = String(v).trim();
  return max ? s.slice(0, max) : s;
}

function num(v) {
  const n = typeof v === "number" ? v : parseFloat(String(v ?? "").replace(",", "."));
  return Number.isFinite(n) && n > 0 ? n : 0;
}

// normalizeDraft(obj) -> a safe subset with coerced types; null for junk.
export function normalizeDraft(obj, isValidCron) {
  if (!obj || typeof obj !== "object" || Array.isArray(obj)) return null;
  const out = {};
  for (const k of FIELDS) if (obj[k] !== undefined) out[k] = obj[k];
  out.name = str(out.name, 60);
  out.prompt = str(out.prompt);
  out.action = out.action === "message" ? "message" : "start";
  out.targetAgentId = str(out.targetAgentId);
  out.workspaceId = str(out.workspaceId);
  out.cron = str(out.cron).split(/\s+/).filter(Boolean).join(" ");
  if (out.cron && typeof isValidCron === "function" && !isValidCron(out.cron)) out.cron = "";
  out.webhook = out.webhook === true || out.webhook === "true";
  out.maxCostUsd = num(out.maxCostUsd);
  out.maxRuns = Math.floor(num(out.maxRuns));
  out.maxRunsWindowMin = Math.floor(num(out.maxRunsWindowMin));
  if (!WINDOWS.has(out.maxRunsWindowMin)) out.maxRunsWindowMin = out.maxRuns ? 1440 : 0;
  if (!out.maxRuns) out.maxRunsWindowMin = 0;
  out.provider = str(out.provider);
  out.model = str(out.model);
  out.thinking = str(out.thinking);
  out.source = str(out.source, 20);
  out.sourceLabel = str(out.sourceLabel, 80);
  out.templateId = str(out.templateId, 40);
  return out;
}

export function writeAutomationDraft(obj) {
  try { sessionStorage.setItem(KEY, JSON.stringify(obj || {})); } catch { /* private mode */ }
}

// readAutomationDraft() -> normalized draft or null; the slot is cleared.
export function readAutomationDraft(isValidCron) {
  let raw = null;
  try {
    raw = sessionStorage.getItem(KEY);
    sessionStorage.removeItem(KEY);
  } catch { return null; }
  if (!raw) return null;
  try { return normalizeDraft(JSON.parse(raw), isValidCron); } catch { return null; }
}

// draftFromTemplate(t) -> the draft a template card hands to the editor.
export function draftFromTemplate(t) {
  return {
    templateId: t.id, name: t.name, prompt: t.prompt, cron: t.cron, action: "start",
    maxCostUsd: t.maxCostUsd, maxRuns: t.maxRuns, maxRunsWindowMin: t.maxRunsWindowMin,
    source: "template", sourceLabel: t.name,
  };
}
