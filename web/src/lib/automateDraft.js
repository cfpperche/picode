// /automate (ADR-0046 v2): ask the current agent for an automation config,
// then read it back out of its reply. Pure — App.jsx does the sending and
// the navigation.
import { normalizeDraft } from "./automationDraft.js";

// isAutomateCommand("/automate every day …") -> "every day …"; "" when bare;
// null when the text is not the command.
export function isAutomateCommand(text) {
  const m = /^\s*\/automate(?:\s+([\s\S]*))?$/i.exec(text || "");
  if (!m) return null;
  return (m[1] || "").trim();
}

// automatePrompt(description, {workspaceName}) -> the message sent to the agent.
export function automatePrompt(description, ctx) {
  const where = ctx && ctx.workspaceName ? " for the " + ctx.workspaceName + " repository" : " for this repository";
  return [
    "Draft a PiCode automation" + where + " from this description:",
    "",
    "> " + description.trim().replace(/\n/g, "\n> "),
    "",
    "Look at the repository first (structure, test and build commands, docs) so the prompt is specific to it, then reply with exactly one ```json fence and at most two sentences after it:",
    "",
    "```json",
    "{",
    '  "name": "short name, at most 60 characters",',
    '  "prompt": "instructions for a future agent run that starts with NO chat context: what to inspect, what to report, what never to change",',
    '  "cron": "5-field cron in the user\'s local time (minute hour day month weekday), or \\"\\" when the automation should only run from a webhook",',
    '  "webhook": false,',
    '  "maxCostUsd": 1.0,',
    '  "maxRuns": 2,',
    '  "maxRunsWindowMin": 1440',
    "}",
    "```",
    "",
    "Rules: action is always a new run (do not include an action field). Use a cron the description implies; weekday work is `* * 1-5`. Keep maxRunsWindowMin one of 60, 1440 or 10080. Do not create anything yourself — PiCode opens the editor with your draft for the user to confirm.",
  ].join("\n");
}

function lastFence(text) {
  const re = /```(?:json|JSON)?\s*\n([\s\S]*?)```/g;
  let m;
  let last = null;
  while ((m = re.exec(text)) !== null) last = m[1];
  return last;
}

function firstBalancedObject(text) {
  const start = text.indexOf("{");
  if (start < 0) return null;
  let depth = 0;
  let inStr = false;
  let esc = false;
  for (let i = start; i < text.length; i++) {
    const c = text[i];
    if (inStr) {
      if (esc) esc = false;
      else if (c === "\\") esc = true;
      else if (c === '"') inStr = false;
      continue;
    }
    if (c === '"') inStr = true;
    else if (c === "{") depth++;
    else if (c === "}") {
      depth--;
      if (depth === 0) return text.slice(start, i + 1);
    }
  }
  return null;
}

// parseAutomateReply(text, isValidCron) -> normalized draft with name and
// prompt, or null when the reply carries no usable config.
export function parseAutomateReply(text, isValidCron) {
  const src = String(text || "");
  const candidates = [lastFence(src), firstBalancedObject(src)].filter(Boolean);
  for (const c of candidates) {
    let obj = null;
    try { obj = JSON.parse(c); } catch { obj = null; }
    if (!obj && c.includes("{")) {
      const inner = firstBalancedObject(c);
      if (inner) { try { obj = JSON.parse(inner); } catch { obj = null; } }
    }
    const d = normalizeDraft(obj, isValidCron);
    if (d && d.name && d.prompt) return d;
  }
  return null;
}
