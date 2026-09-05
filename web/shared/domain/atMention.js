// Current @token at the caret. @ must start a word (start or after whitespace).
export function atQuery(text, caret) {
  const s = text || "";
  const n = caret == null ? s.length : Math.max(0, Math.min(caret, s.length));
  const left = s.slice(0, n);
  const m = /(^|[\s])@([^\s]*)$/.exec(left);
  if (!m) return null;
  return { start: m.index + m[1].length, query: m[2] };
}

export function skillsFromSlash(extras) {
  const out = [];
  for (const x of extras || []) {
    const id = String(x.id || "");
    if (!id.startsWith("skill:")) continue;
    out.push({ name: id.slice(6), hint: x.hint || "Skill" });
  }
  return out;
}

function matchQ(q, ...parts) {
  if (!q) return true;
  return parts.some((p) => String(p || "").toLowerCase().includes(q));
}

export function mergeAtHits(query, { files, skills, agents } = {}) {
  const q = String(query || "").toLowerCase();
  const ag = [];
  for (const a of agents || []) {
    const name = a.name || a.id || "";
    if (!name) continue;
    if (matchQ(q, name, "agent:" + name, a.id)) {
      ag.push({ kind: "agent", path: "agent:" + name, name });
    }
  }
  const sk = [];
  for (const s of skills || []) {
    const name = s.name || "";
    if (!name) continue;
    if (matchQ(q, name, "skill:" + name)) {
      sk.push({ kind: "skill", path: "skill:" + name, name, hint: s.hint || "Skill" });
    }
  }
  const fl = [];
  for (const f of files || []) {
    if (matchQ(q, f.path, f.name)) {
      fl.push({ kind: "file", path: f.path, name: f.name || f.path });
    }
  }
  if (!q) {
    return fl.slice(0, 12).concat(ag.slice(0, 4), sk.slice(0, 4)).slice(0, 20);
  }
  return ag.slice(0, 5).concat(sk.slice(0, 5), fl.slice(0, 20)).slice(0, 20);
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
