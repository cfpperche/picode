export function newHist() {
  return { entries: [], i: 0, draft: "" };
}

export function histPush(h, text) {
  const t = String(text || "").trim();
  if (!t) return;
  if (h.entries[h.entries.length - 1] !== t) h.entries.push(t);
  if (h.entries.length > 50) h.entries.shift();
  h.i = h.entries.length;
  h.draft = "";
}

export function histUp(h, current) {
  if (!h.entries.length) return current;
  if (h.i === h.entries.length) h.draft = current;
  if (h.i > 0) h.i--;
  return h.entries[h.i];
}

export function histDown(h, current) {
  if (h.i < h.entries.length - 1) {
    h.i++;
    return h.entries[h.i];
  }
  if (h.i === h.entries.length - 1) {
    h.i = h.entries.length;
    return h.draft;
  }
  return current;
}

export function histTyped(h) {
  h.i = h.entries.length;
}

export function caretFirstLine(el) {
  if (!el) return true;
  const s = el.selectionStart;
  return s === 0 || el.value.lastIndexOf("\n", s - 1) === -1;
}

export function caretLastLine(el) {
  if (!el) return true;
  const s = el.selectionStart;
  return s === el.value.length || el.value.indexOf("\n", s) === -1;
}
