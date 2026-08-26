// Current @token at the caret. @ must start a word (start or after whitespace).
export function atQuery(text, caret) {
  const s = text || "";
  const n = caret == null ? s.length : Math.max(0, Math.min(caret, s.length));
  const left = s.slice(0, n);
  const m = /(^|[\s])@([^\s]*)$/.exec(left);
  if (!m) return null;
  return { start: m.index + m[1].length, query: m[2] };
}

export function insertAtPath(text, caret, path) {
  const tok = atQuery(text, caret);
  const s = text || "";
  const n = caret == null ? s.length : caret;
  const rel = String(path || "").replace(/\\/g, "/");
  const token = /\s/.test(rel) ? '@"' + rel + '" ' : "@" + rel + " ";
  if (!tok) {
    const next = s.slice(0, n) + token + s.slice(n);
    return { text: next, caret: n + token.length };
  }
  const next = s.slice(0, tok.start) + token + s.slice(n);
  return { text: next, caret: tok.start + token.length };
}
