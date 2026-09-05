// History and socket delivery overlap. Match tool identity, not tool name or
// screenshot URL. Callers must reject stale agent/session/request generations.
function same(a, b) {
  if (a.kind !== b.kind) return false;
  if (a.kind === "tool") return !!a.id && a.id === b.id;
  if (a.kind === "block") return a.cls === b.cls && a.actor === b.actor && a.text === b.text &&
    (!a.ts || !b.ts || a.ts === b.ts);
  return !!a.id && a.id === b.id;
}

export function startTool(items, item) {
  return items.some((it) => it.kind === "tool" && it.id === item.id) ? items : [...items, item];
}

export function reconcileTranscript(history, live) {
  const result = [...history];
  // Walk in source order, so repeated equal messages match occurrences rather
  // than collapsing every identical "OK" in a session into one message.
  let cursor = 0;
  let insertion = result.length;
  for (const item of live) {
    const index = result.findIndex((candidate, i) => (item.kind === "tool" || i >= cursor) && same(candidate, item));
    if (index >= 0) {
      const stored = result[index];
      if (item.kind === "tool") {
        // A completed persisted result beats a partial socket row. A completed
        // live result beats the unfinished transcript read while it was running.
        result[index] = stored.status !== "···" && item.status === "···"
          ? { ...stored, expanded: item.expanded }
          : { ...stored, ...item };
      }
      cursor = index + 1;
      insertion = cursor;
    } else {
      result.splice(insertion, 0, item);
      insertion++;
      cursor = insertion;
    }
  }
  return result;
}

// Only retain changes made during the read, or an already running suffix.
// An unchanged historical prefix must not resurrect old compaction history.
export function liveSince(items, baseline, includePending = true) {
  const index = items.findIndex((it, i) => it !== baseline[i] ||
    (includePending && ((it.kind === "tool" && it.status === "···") ||
      it.kind === "ask" || it.kind === "note" || (it.kind === "block" && !it.ts))));
  return index < 0 ? [] : items.slice(index);
}

// Generation gate shared by both apps. Invalidate on explicit navigation and
// teardown, even if a user returns to the same agent before an old read ends.
export function transcriptGate() {
  let generation = 0;
  return {
    begin: () => ++generation,
    token: () => generation,
    current: (ticket) => ticket === generation,
    invalidate: () => { generation++; },
  };
}
