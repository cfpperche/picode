// Client rules for Snippets (ADR-0130). Mirrors internal/snips: Parse does
// not unescape, Expand is one-pass, Slug hyphenates. No DOM beyond
// sessionStorage through an injected store.

export const SNIP_LIMITS = Object.freeze({
  title: 200,
  description: 500,
  tag: 40,
  tags: 16,
  bodyBytes: 100_000,
  name: 40,
  placeholders: 32,
  defaultRunes: 500,
  slug: 64,
});

export const SNIP_RESERVED = Object.freeze(["cwd", "workspace", "branch", "date", "agent", "cli"]);

const RESERVED = new Set(SNIP_RESERVED);

export function utf8Bytes(s) {
  const str = String(s == null ? "" : s);
  if (typeof TextEncoder !== "undefined") return new TextEncoder().encode(str).length;
  let n = 0;
  for (let i = 0; i < str.length; i++) {
    const c = str.charCodeAt(i);
    if (c < 0x80) n += 1;
    else if (c < 0x800) n += 2;
    else if (c >= 0xd800 && c <= 0xdbff) { n += 4; i++; } else n += 3;
  }
  return n;
}

export function bodyLimit(body) {
  const bytes = utf8Bytes(body);
  const max = SNIP_LIMITS.bodyBytes;
  return { bytes, max, over: bytes > max, near: bytes > max * 0.9 };
}

export function snipSlug(s) {
  s = String(s || "").trim().toLowerCase();
  let out = "";
  let dash = false;
  for (const ch of s) {
    if ((ch >= "a" && ch <= "z") || (ch >= "0" && ch <= "9")) {
      out += ch;
      dash = false;
    } else if (out && !dash) {
      out += "-";
      dash = true;
    }
  }
  out = out.replace(/^-+|-+$/g, "");
  if (out.length > SNIP_LIMITS.slug) out = out.slice(0, SNIP_LIMITS.slug).replace(/-+$/g, "");
  return out;
}

function hasAt(s, i, pre) {
  return s.slice(i, i + pre.length) === pre;
}

function skipWS(s, i) {
  while (i < s.length && (s[i] === " " || s[i] === "\t")) i++;
  return i;
}

function isNameStart(ch) {
  return ch === "_" || (ch >= "A" && ch <= "Z") || (ch >= "a" && ch <= "z");
}

function isNameCont(ch) {
  return isNameStart(ch) || (ch >= "0" && ch <= "9");
}

function readName(body, i) {
  if (i >= body.length || !isNameStart(body[i])) {
    return { err: "invalid placeholder name", i };
  }
  let j = i + 1;
  while (j < body.length && isNameCont(body[j])) j++;
  const name = body.slice(i, j);
  if (name.length > SNIP_LIMITS.name) return { err: "placeholder name too long", i };
  return { name, i: j };
}

function readDefault(body, i) {
  let def = "";
  while (i < body.length) {
    if (hasAt(body, i, "}}}}")) { def += "}}"; i += 4; continue; }
    if (hasAt(body, i, "{{{{")) { def += "{{"; i += 4; continue; }
    if (hasAt(body, i, "}}")) return { def: def.trim(), i };
    def += body[i];
    i++;
  }
  return { err: "unclosed placeholder" };
}

function readPlaceholder(body, i) {
  if (!hasAt(body, i, "{{")) return { err: "not a placeholder" };
  const start = i;
  i += 2;
  i = skipWS(body, i);
  if (i >= body.length) return { err: "unclosed placeholder", i: start };
  const n = readName(body, i);
  if (n.err) return { err: n.err, i: start };
  i = skipWS(body, n.i);
  const ph = { name: n.name, optional: false, default: "" };
  if (i < body.length && body[i] === "=") {
    ph.optional = true;
    i++;
    const d = readDefault(body, i);
    if (d.err) return { err: d.err, i: start };
    if ([...d.def].length > SNIP_LIMITS.defaultRunes) return { err: "default too long", i: start };
    ph.default = d.def;
    i = d.i;
  }
  i = skipWS(body, i);
  if (!hasAt(body, i, "}}")) return { err: "unclosed placeholder", i: start };
  return { ph, i: i + 2 };
}

export function parseSnip(body) {
  const s = String(body == null ? "" : body);
  const out = [];
  const seen = new Set();
  let i = 0;
  while (i < s.length) {
    if (hasAt(s, i, "{{{{") || hasAt(s, i, "}}}}")) { i += 4; continue; }
    if (hasAt(s, i, "{{")) {
      const r = readPlaceholder(s, i);
      if (r.err) return { ok: false, error: r.err, placeholders: [] };
      if (!seen.has(r.ph.name)) {
        if (out.length >= SNIP_LIMITS.placeholders) return { ok: false, error: "too many placeholders", placeholders: [] };
        seen.add(r.ph.name);
        out.push(r.ph);
      }
      i = r.i;
      continue;
    }
    i++;
  }
  return { ok: true, error: "", placeholders: out };
}

function lookup(ph, values, ctx) {
  if (RESERVED.has(ph.name)) {
    if (ph.name === "date") {
      if (ctx && Object.prototype.hasOwnProperty.call(ctx, "date")) return { ok: true, val: String(ctx.date || "") };
      const d = new Date();
      const iso = d.toISOString().slice(0, 10);
      return { ok: true, val: iso };
    }
    return { ok: true, val: ctx && ctx[ph.name] != null ? String(ctx[ph.name]) : "" };
  }
  if (values && values[ph.name]) return { ok: true, val: String(values[ph.name]) };
  if (ph.optional) return { ok: true, val: ph.default || "" };
  return { ok: false, val: "" };
}

export function expandSnip(body, values, ctx) {
  const s = String(body == null ? "" : body);
  let out = "";
  const missing = [];
  const seenMiss = new Set();
  let i = 0;
  while (i < s.length) {
    if (hasAt(s, i, "{{{{")) { out += "{{"; i += 4; continue; }
    if (hasAt(s, i, "}}}}")) { out += "}}"; i += 4; continue; }
    if (hasAt(s, i, "{{")) {
      const r = readPlaceholder(s, i);
      if (r.err) return { ok: false, error: r.err, text: "", missing: [] };
      const got = lookup(r.ph, values || {}, ctx || {});
      if (!got.ok) {
        if (!seenMiss.has(r.ph.name)) { missing.push(r.ph.name); seenMiss.add(r.ph.name); }
      } else out += got.val;
      i = r.i;
      continue;
    }
    out += s[i];
    i++;
  }
  return { ok: true, error: "", text: out, missing };
}

export function insertLiteralBraces(body, start, end) {
  const s = String(body == null ? "" : body);
  const a = Math.max(0, start | 0);
  const b = Math.max(a, end | 0);
  return s.slice(0, a) + "{{{{" + s.slice(b);
}

export function draftKey(id) {
  return "picode-snip-draft:" + (id || "new");
}

export function readDraft(store, id) {
  try {
    const raw = store && store.getItem(draftKey(id));
    if (!raw) return null;
    const d = JSON.parse(raw);
    if (!d || typeof d !== "object") return null;
    return {
      title: String(d.title || ""),
      slug: String(d.slug || ""),
      description: String(d.description || ""),
      body: String(d.body || ""),
      tags: String(d.tags || ""),
      kind: d.kind === "shell" ? "shell" : "prompt",
      slugLocked: !!d.slugLocked,
      base: d.base || "",
    };
  } catch { return null; }
}

export function writeDraft(store, id, draft, base) {
  try {
    store.setItem(draftKey(id), JSON.stringify({
      title: draft.title || "",
      slug: draft.slug || "",
      description: draft.description || "",
      body: draft.body || "",
      tags: draft.tags || "",
      kind: draft.kind === "shell" ? "shell" : "prompt",
      slugLocked: !!draft.slugLocked,
      base: base || "",
    }));
  } catch { /* quota */ }
}

export function clearDraft(store, id) {
  try { store.removeItem(draftKey(id)); } catch { /* ignore */ }
}

export function sameDraft(a, b) {
  if (!a || !b) return false;
  return (a.title || "") === (b.title || "")
    && (a.slug || "") === (b.slug || "")
    && (a.description || "") === (b.description || "")
    && (a.body || "") === (b.body || "")
    && (a.tags || "") === (b.tags || "")
    && (a.kind || "prompt") === (b.kind || "prompt");
}

export function draftToRestore(draft, server) {
  if (!draft) return null;
  if (!server) return sameDraft(draft, { title: "", slug: "", description: "", body: "", tags: "" }) ? null : draft;
  // Edit: only restore a draft taken from this server version. An empty
  // base is the editor's first paint, not a user edit.
  if (!draft.base || draft.base !== server.updatedAt) return null;
  return sameDraft(draft, server) ? null : draft;
}

export function formFromSnip(p) {
  if (!p) return { title: "", slug: "", description: "", body: "", tags: "", kind: "prompt", slugLocked: false };
  return {
    title: p.title || "",
    slug: p.slug || "",
    description: p.description || "",
    body: p.body || "",
    tags: (p.tags || []).join(", "),
    kind: p.kind === "shell" ? "shell" : "prompt",
    slugLocked: true,
  };
}

export function tagsFromInput(raw) {
  return String(raw || "").split(",").map((t) => t.trim()).filter(Boolean);
}

export function userPlaceholderNames(placeholders) {
  return (placeholders || []).map((p) => p.name || p).filter((n) => n && !RESERVED.has(n));
}

// D4: "see /snip:review please" + "look at PR 1" → "see look at PR 1 please"
export function replaceSnipToken(draft, slug, expanded) {
  const s = String(draft || "");
  const exp = String(expanded || "");
  const exact = "/snip:" + String(slug || "");
  const i = s.indexOf(exact);
  if (i >= 0) return s.slice(0, i) + exp + s.slice(i + exact.length);
  const m = s.match(/\/snip:[A-Za-z0-9_-]*/);
  if (m && m.index >= 0) return s.slice(0, m.index) + exp + s.slice(m.index + m[0].length);
  if (!s.trim()) return exp;
  return s.replace(/\s*$/, (end) => (end ? " " : "") + exp);
}
