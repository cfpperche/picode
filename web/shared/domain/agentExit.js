// Exit records (ADR-0194): the removal dialog's question and the numbers the
// catalog and the dashboard read. The server owns the codes and their words
// (the taxonomy it sends with the cleanup preview and every exit list);
// these helpers only shape them, so desktop and phone ask the same thing.

export function emptyExitDraft() {
  return { outcome: "", reasons: [], note: "" };
}

export function takesReasons(taxonomy, outcome) {
  return !!outcome && ((taxonomy && taxonomy.reasonsFor) || []).includes(outcome);
}

// Picking the chosen outcome again clears it: the question stays optional.
// Reasons belong to the outcomes that ask for them and go when it changes.
export function pickOutcome(taxonomy, draft, id) {
  const outcome = draft.outcome === id ? "" : id;
  return { ...draft, outcome, reasons: takesReasons(taxonomy, outcome) ? draft.reasons : [] };
}

export function toggleReason(draft, id) {
  const has = draft.reasons.includes(id);
  return { ...draft, reasons: has ? draft.reasons.filter((r) => r !== id) : [...draft.reasons, id] };
}

// The body of DELETE /api/agents/{id}: which face removed the agent,
// whether it showed the question, and the answer. A question that was not
// shown sends no answer, whatever the draft held.
export function exitRequestBody({ taxonomy, draft, shown, origin }) {
  const d = draft || emptyExitDraft();
  const outcome = shown ? String(d.outcome || "") : "";
  return {
    exit: {
      origin,
      asked: !!shown,
      outcome,
      reasons: shown && takesReasons(taxonomy, outcome) ? [...d.reasons] : [],
      note: shown && outcome ? String(d.note || "").trim() : "",
    },
  };
}

export function choiceLabel(list, id) {
  const c = (list || []).find((x) => x.id === id);
  return c ? c.label : id;
}

// A lifetime in the largest unit that still reads as a whole number, in the
// same short units the removal time uses (relTime: "4m", "2h", "3d"), so a
// row never mixes two ways of writing time.
export function fmtLifetime(seconds) {
  const s = Math.max(0, Math.floor(Number(seconds) || 0));
  if (s < 60) return s + "s";
  if (s < 3600) return Math.floor(s / 60) + "m";
  if (s < 86400) return Math.floor(s / 3600) + "h";
  return Math.floor(s / 86400) + "d";
}

// Turns are "not measured" (null) for agents born before the counter —
// silence is a value, never a zero.
export function fmtTurns(turns) {
  return turns == null ? "—" : String(turns);
}

export function fmtShare(part, whole) {
  if (!whole) return "—";
  return Math.round((part / whole) * 100) + "%";
}

// The headline numbers of a summary: resolved among the agents that were
// set a task (a "Just trying" answer is left aside — nothing was asked of
// that agent), and answered in the dialog among the times it asked.
export function exitHeadline(summary) {
  const s = summary || {};
  const outcomes = s.outcomes || {};
  const resolved = outcomes.resolved || 0;
  const attempts = resolved + (outcomes.partial || 0) + (outcomes.unresolved || 0);
  return {
    total: s.total || 0,
    answered: s.answered || 0,
    attempts,
    resolved,
    resolvedShare: fmtShare(resolved, attempts),
    answerRate: fmtShare(s.askedAnswered || 0, s.asked || 0),
  };
}

// Reasons as RankedBars rows, in the words the taxonomy gives them.
export function reasonRows(summary, taxonomy) {
  return ((summary && summary.reasons) || []).map((r) => ({
    key: r.id,
    label: choiceLabel(taxonomy && taxonomy.reasons, r.id),
    value: r.count,
    display: String(r.count),
  }));
}

// Outcomes as RankedBars rows, unanswered last, zero rows left out.
export function outcomeRows(summary, taxonomy) {
  const outcomes = (summary && summary.outcomes) || {};
  const rows = ((taxonomy && taxonomy.outcomes) || []).map((o) => ({
    key: o.id, label: o.label, value: outcomes[o.id] || 0, display: String(outcomes[o.id] || 0),
  }));
  rows.push({ key: "unanswered", label: "No answer", value: outcomes.unanswered || 0, display: String(outcomes.unanswered || 0), muted: true });
  return rows.filter((r) => r.value > 0);
}

// What an exit's sessions cost, as the dashboard writes spend: "—" when not
// measured (a CLI whose sessions are not files), "not priced" when every
// turn went unpriced, "~" when part of it is a list-price estimate
// (ADR-0185), and never "$0.00" for something that merely was not priced.
export function fmtExitCost(cost) {
  if (!cost) return "—";
  const n = Number(cost.cost) || 0;
  if (n === 0 && cost.unpriced > 0) return "not priced";
  const est = (Number(cost.estimated) || 0) > 0 ? "~" : "";
  if (n > 0 && n < 0.01) return est + "<$0.01";
  return est + "$" + n.toFixed(2);
}
