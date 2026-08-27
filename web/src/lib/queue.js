export function pendingFollowUps(items) {
  return (items || []).filter((it) =>
    it.kind === "block" && it.cls === "user" && it.chip === "follow_up"
    && it.pending && !it.dropped && !it.editing);
}

export function dropQueued(items, qid) {
  return (items || []).map((it) => (it.qid === qid && it.pending
    ? { ...it, dropped: true, pending: false, editing: false }
    : it));
}

export function startEditQueued(items, qid) {
  return (items || []).map((it) => (it.qid === qid && it.pending && !it.dropped
    ? { ...it, editing: true }
    : it));
}

export function saveEditQueued(items, qid, text) {
  const t = String(text || "").trim();
  if (!t) return dropQueued(items, qid);
  return (items || []).map((it) => (it.qid === qid && it.pending
    ? { ...it, text: t, editing: false }
    : it));
}

export function cancelEditQueued(items, qid) {
  return (items || []).map((it) => (it.qid === qid ? { ...it, editing: false } : it));
}
