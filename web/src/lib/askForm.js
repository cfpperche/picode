/** Sequential extension_ui_request cards in one turn become one growing form. */

/**
 * Sentinel option an extension puts on a select to mean "go back one field"
 * (packages/pi-roles does). Cancel always means abort. The UI hides this
 * option from dropdowns and answers it when a prior pill is clicked.
 */
export const BACK = "‹ back";

function isUser(it) {
  return it && it.kind === "block" && it.cls === "user";
}

function isAsk(it) {
  return it && it.kind === "ask";
}

function lastUserIndex(items) {
  let n = -1;
  for (let i = 0; i < items.length; i++) {
    if (isUser(items[i])) n = i;
  }
  return n;
}

function stepFromDialog(d, status) {
  return {
    id: d.id,
    method: d.method || "select",
    title: d.title || "",
    message: d.message || "",
    options: d.options || [],
    placeholder: d.placeholder || "",
    prefill: d.prefill || "",
    status: status || "open",
    answer: "",
  };
}

function askToStep(it) {
  return {
    id: it.id,
    method: it.method || "select",
    title: it.title || "",
    message: it.message || "",
    options: it.options || [],
    placeholder: it.placeholder || "",
    prefill: it.prefill || "",
    status: it.status || "open",
    answer: it.answer || "",
  };
}

function stepsOf(it) {
  if (it.steps && it.steps.length) return it.steps.slice();
  return [askToStep(it)];
}

/** Index of the ask card that should grow, or -1. */
export function stitchIndex(items) {
  const list = items || [];
  const from = lastUserIndex(list);
  for (let i = list.length - 1; i > from; i--) {
    const it = list[i];
    if (isAsk(it)) {
      if (it.status === "cancelled" || it.status === "timeout") return -1;
      return i;
    }
    if (it.kind === "block" || it.kind === "tool") return -1;
  }
  return -1;
}

/** The slash command whose flow this card belongs to ("/roles clear"). */
function cmdOf(items) {
  const list = items || [];
  for (let i = list.length - 1; i >= 0; i--) {
    if (isUser(list[i])) {
      const text = String(list[i].text || "").trim().split("\n")[0];
      return /^\//.test(text) ? text : "";
    }
  }
  return "";
}

function cardFromDialog(d, status, cmd) {
  const step = stepFromDialog(d, status);
  return {
    kind: "ask",
    id: d.id,
    method: step.method,
    title: step.title,
    message: step.message,
    options: step.options,
    placeholder: step.placeholder,
    prefill: step.prefill,
    timeout: d.timeout || 0,
    status: status || "open",
    answer: "",
    cmd: cmd || "",
    steps: [step],
    ts: Date.now(),
  };
}

function applyDialog(card, d, status) {
  return {
    ...card,
    id: d.id,
    method: d.method || card.method,
    title: d.title || "",
    message: d.message || "",
    options: d.options || [],
    placeholder: d.placeholder || "",
    prefill: d.prefill || "",
    timeout: d.timeout || 0,
    status: status || "open",
  };
}

export function putAsk(items, dialog, status) {
  if (!dialog || !dialog.id) return items || [];
  const nextStatus = status || "open";
  const cur = items || [];
  for (let i = 0; i < cur.length; i++) {
    const it = cur[i];
    if (!isAsk(it)) continue;
    const steps = stepsOf(it);
    const si = steps.findIndex((s) => s.id === dialog.id);
    if (it.id !== dialog.id && si < 0) continue;
    const copy = cur.slice();
    if (si >= 0) {
      steps[si] = { ...steps[si], ...stepFromDialog(dialog, nextStatus), answer: steps[si].answer };
      copy[i] = { ...applyDialog(it, dialog, nextStatus), steps };
    } else {
      steps.push(stepFromDialog(dialog, nextStatus));
      copy[i] = { ...applyDialog(it, dialog, nextStatus), steps };
    }
    return copy;
  }
  const at = stitchIndex(cur);
  if (at >= 0) {
    const it = cur[at];
    const steps = stepsOf(it);
    // One dialog is outstanding at a time, so an open step is always the
    // one this dialog replaces (a back-walk target or a restored ghost);
    // answered steps stay and the next field appends.
    const openI = steps.findIndex((s) => s.status === "open");
    if (openI >= 0) steps[openI] = stepFromDialog(dialog, nextStatus);
    else steps.push(stepFromDialog(dialog, nextStatus));
    const copy = cur.slice();
    copy[at] = { ...applyDialog(it, dialog, nextStatus), steps, backTo: "" };
    return copy;
  }
  return [...cur, cardFromDialog(dialog, nextStatus, cmdOf(cur))];
}

export function answerAsk(items, id, answer, cancelled) {
  const status = cancelled ? "cancelled" : "answered";
  const text = cancelled ? "Cancelled" : (answer || "Answered");
  return (items || []).map((it) => {
    if (!isAsk(it) || it.status !== "open") return it;
    const steps = stepsOf(it);
    let hit = it.id === id;
    const nextSteps = steps.map((s) => {
      if (s.id !== id || s.status !== "open") return s;
      hit = true;
      return { ...s, status, answer: text };
    });
    if (!hit) return it;
    return { ...it, status, answer: text, steps: nextSteps };
  });
}

export function timeoutAsk(items, id) {
  return (items || []).map((it) => {
    if (!isAsk(it) || it.status !== "open") return it;
    const steps = stepsOf(it);
    let hit = it.id === id;
    const nextSteps = steps.map((s) => {
      if (s.id !== id || s.status !== "open") return s;
      hit = true;
      return { ...s, status: "timeout", answer: "Timed out" };
    });
    if (!hit) return it;
    return { ...it, status: "timeout", answer: "Timed out", steps: nextSteps };
  });
}

export function cancelOpenAsks(items) {
  return (items || []).map((it) => {
    if (!isAsk(it) || it.status !== "open") return it;
    const steps = stepsOf(it).map((s) => (
      s.status === "open" ? { ...s, status: "cancelled", answer: "Cancelled" } : s
    ));
    return { ...it, status: "cancelled", answer: "Cancelled", steps };
  });
}

const THINKING = new Set(["off", "minimal", "low", "medium", "high", "xhigh", "max"]);

function cap(s) {
  if (!s) return "Choose";
  return s.charAt(0).toUpperCase() + s.slice(1);
}

/** Short field name for a select title. */
export function fieldLabel(title) {
  const t = String(title || "").toLowerCase();
  if (t.includes("save to")) return "Save";
  // Both the scope select ("Clear which config?") and the confirm that
  // follows it ("Delete this roles file?") — the confirm is the only step
  // when the scope came as a command argument, so it alone must still
  // mark the flow as a clear (else the note-parsing fallback below
  // mistakes a file path for a provider/id pair).
  if (t.includes("clear which") || t.includes("delete this roles file")) return "Clear";
  // Only the role *picker* titles ("Edit which role?", "Roles (current:
  // …)") — not every dialog that happens to mention "roles" in passing
  // (e.g. "Delete this roles file?").
  if (t.includes("which role") || /^roles\b/.test(t)) return "Role";
  if (t.includes("provider")) return "Provider";
  if (t.includes("thinking")) return "Thinking";
  if (t.includes("model")) return "Model";
  if (t.includes("preset") || t.includes("name")) return "Name";
  const parts = String(title || "").split("—");
  if (parts.length > 1) return cap(parts[parts.length - 1].trim());
  return "Choose";
}

/**
 * Structured outcome of a finished form. `note` is the extension's
 * completion notify (e.g. "xai/grok-4.6 · high · lock /default"); it fills
 * in what the answers alone cannot say — the model behind a role pick, the
 * provider when its select was skipped, or a clear/kept result.
 *
 * kind: "definition" (role/model chips) | "role" (role only) |
 *       "cleared" | "kept" | "empty" (nothing to act on) | "text".
 */
export function summaryParts(steps, note) {
  const by = {};
  const answers = [];
  for (const s of steps || []) {
    if (s.status !== "answered" || !s.answer) continue;
    answers.push(s.answer);
    const lab = fieldLabel(s.title);
    if (!(lab in by)) by[lab] = s.answer;
  }
  if (!answers.length) return null;
  const text = String(note || "");
  // A finished clear flow reads as its result, not as a definition.
  if (by.Clear) {
    let m = /^Cleared\s+(\S+)/.exec(text);
    if (m) return { kind: "cleared", file: m[1], text };
    m = /^Kept\s+(\S+)/.exec(text);
    if (m) return { kind: "kept", file: m[1], text };
    if (/^Nothing to clear/.test(text)) return { kind: "empty", text };
    return { kind: "text", text: text || answers.join(" · ") };
  }
  const role = by.Role || by.Name || "";
  const scope = by.Save === "workspace" ? "workspace" : "";
  let thinking = by.Thinking && by.Thinking !== "none" ? by.Thinking : "";
  let model = by.Provider && by.Model ? by.Provider + "/" + by.Model : "";
  if (!model || !thinking) {
    for (const t of text.split(/[\s·]+/)) {
      if (!model && /^[\w.-]+\/\S+$/.test(t)) model = t;
      else if (!thinking && THINKING.has(t)) thinking = t;
    }
  }
  if (!model && by.Model) model = by.Model;
  if (model) {
    const slash = model.indexOf("/");
    return {
      kind: "definition",
      role,
      model,
      provider: slash > 0 ? model.slice(0, slash) : "",
      modelId: slash > 0 ? model.slice(slash + 1) : model,
      thinking,
      scope,
      text,
    };
  }
  if (role) return { kind: "role", role, text };
  const main = answers.join(" · ");
  return {
    kind: "text",
    text: thinking && !answers.includes(thinking) ? main + " · " + thinking : main,
  };
}

/** summaryParts flattened to one line (persistence, tests, fallbacks). */
export function summaryLine(steps, note) {
  const p = summaryParts(steps, note);
  if (!p) return "";
  if (p.kind === "cleared" || p.kind === "kept" || p.kind === "empty") return p.text;
  if (p.kind === "definition") {
    const line = (p.role ? p.role + " — " : "") + p.model;
    return (p.thinking ? line + " · " + p.thinking : line) + (p.scope ? " (" + p.scope + ")" : "");
  }
  if (p.kind === "role") return "Role — " + p.role;
  return p.text;
}

/** Fold the extension's completion notify into the just-answered card. */
export function noteAsk(items, text) {
  const list = items || [];
  for (let i = list.length - 1; i >= 0; i--) {
    const it = list[i];
    if (isAsk(it)) {
      if (it.status !== "answered" || it.note) return list;
      const copy = list.slice();
      copy[i] = { ...it, note: text };
      return copy;
    }
    if (isUser(it) || it.kind === "block" || it.kind === "tool") return list;
  }
  return list;
}

/** Reopen an optimistically answered step when the server rejected the reply. */
export function unanswerAsk(items, id) {
  return (items || []).map((it) => {
    if (!isAsk(it)) return it;
    const steps = stepsOf(it);
    if (!steps.some((s) => s.id === id && s.status === "answered")) return it;
    const nextSteps = steps.map((s) => (
      s.id === id ? { ...s, status: "open", answer: "" } : s
    ));
    return { ...it, status: "open", answer: "", id, steps: nextSteps };
  });
}

/** Reopen the clicked pill as the current field; drop everything after it. */
export function backAsk(items, id, keepCount) {
  return (items || []).map((it) => {
    if (!isAsk(it) || it.status !== "open") return it;
    const steps = stepsOf(it);
    const hit = it.id === id || steps.some((s) => s.id === id);
    if (!hit) return it;
    const answered = steps.filter((s) => s.status === "answered");
    const clicked = answered[keepCount];
    if (!clicked) return it;
    const kept = [];
    for (const s of steps) {
      if (s.id === clicked.id) break;
      if (s.status === "answered") kept.push(s);
    }
    const reopen = { ...clicked, status: "open", answer: "" };
    return {
      ...it,
      status: "open",
      answer: "",
      backTo: fieldLabel(clicked.title),
      steps: [...kept, reopen],
      id: reopen.id,
    };
  });
}

/**
 * While walking back to a clicked pill: the reply to auto-send for an
 * incoming dialog. BACK when it is a wrong field that offers BACK; ""
 * to show the dialog (target reached, or the extension cannot go back).
 */
export function walkReply(items, dialog) {
  const at = stitchIndex(items);
  if (at < 0) return "";
  const it = items[at];
  if (!it || !it.backTo) return "";
  if (fieldLabel(dialog && dialog.title) === it.backTo) return "";
  const opts = (dialog && dialog.options) || [];
  return opts.includes(BACK) ? BACK : "";
}

/** True when the latest ask in this turn is answered (form finished or between steps). */
export function askJustAnswered(items) {
  const list = items || [];
  for (let i = list.length - 1; i >= 0; i--) {
    if (isAsk(list[i])) return list[i].status === "answered";
    if (isUser(list[i]) || list[i].kind === "block" || list[i].kind === "tool") return false;
  }
  return false;
}
