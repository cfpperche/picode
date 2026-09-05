export function blocksFromMessage(m) {
  const content = m && m.content;
  const out = [];
  if (!Array.isArray(content)) return out;
  for (const b of content) {
    if (!b || typeof b !== "object") continue;
    if (b.type === "thinking" && String(b.thinking || "").trim()) {
      out.push({ kind: "block", cls: "thinking", actor: "thinking", text: String(b.thinking) });
    }
    if (b.type === "text" && String(b.text || "").trim()) {
      out.push({ kind: "block", cls: "", actor: "agent", text: String(b.text) });
    }
  }
  return out;
}

function lastUserIndex(items) {
  for (let k = (items || []).length - 1; k >= 0; k--) {
    if (items[k] && items[k].kind === "block" && items[k].cls === "user") return k;
  }
  return -1;
}

function findInTurn(items, userAt, cls, actor) {
  for (let k = items.length - 1; k > userAt; k--) {
    const x = items[k];
    if (x && x.kind === "block" && x.cls === cls && x.actor === actor) return k;
  }
  return -1;
}

function preferText(have, next) {
  if (!have) return next;
  if (!next) return have;
  if (have === next) return have;
  if (next.startsWith(have) || have.startsWith(next)) return next.length >= have.length ? next : have;
  return next;
}

export function mergeAssistant(cur, m) {
  const blocks = blocksFromMessage(m);
  if (!blocks.length) return cur || [];
  const next = (cur || []).slice();
  const userAt = lastUserIndex(next);
  for (const b of blocks) {
    const i = findInTurn(next, userAt, b.cls, b.actor);
    if (i >= 0) {
      const have = String(next[i].text || "");
      const text = preferText(have, b.text);
      if (text !== have) next[i] = { ...next[i], text };
      continue;
    }
    next.push({ ...b, ts: Date.now() });
  }
  return next;
}
