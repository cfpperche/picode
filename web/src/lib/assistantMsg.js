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

export function mergeAssistant(cur, m) {
  const blocks = blocksFromMessage(m);
  if (!blocks.length) return cur || [];
  const next = (cur || []).slice();
  for (const b of blocks) {
    let i = -1;
    for (let k = next.length - 1; k >= 0; k--) {
      const x = next[k];
      if (x && x.kind === "block" && x.cls === b.cls && x.actor === b.actor) {
        i = k;
        break;
      }
      if (x && x.kind === "block" && x.cls === "user") break;
    }
    if (i >= 0) {
      const have = String(next[i].text || "");
      if (have === b.text) continue;
      if (b.text.startsWith(have)) {
        next[i] = { ...next[i], text: b.text };
        continue;
      }
    }
    next.push({ ...b, ts: Date.now() });
  }
  return next;
}
