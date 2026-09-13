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

function excerpt(body, at) {
  const s = String(body || "");
  const a = Math.max(0, (at | 0) - 8);
  return s.slice(a, a + 24).replace(/\n/g, " ");
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
      if (r.err) return { ok: false, error: r.err, at: r.i || i, excerpt: excerpt(s, r.i || i), placeholders: [] };
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
      enums: d.enums && typeof d.enums === "object" && !Array.isArray(d.enums) ? d.enums : {},
      slugLocked: !!d.slugLocked,
      base: d.base || "",
      origin: typeof d.origin === "string" ? d.origin : "",
    };
  } catch { return null; }
}

// `origin` records why a draft exists at all: "capture"/"import" mean the
// studio handed this text over on purpose, so an editor that offers to
// "restore unsaved changes" would be lying about where it came from. Any
// other value (or none) is a draft the reader was in the middle of.
export function writeDraft(store, id, draft, base, origin) {
  try {
    store.setItem(draftKey(id), JSON.stringify({
      title: draft.title || "",
      slug: draft.slug || "",
      description: draft.description || "",
      body: draft.body || "",
      tags: draft.tags || "",
      kind: draft.kind === "shell" ? "shell" : "prompt",
      enums: draft.enums && typeof draft.enums === "object" && !Array.isArray(draft.enums) ? draft.enums : {},
      slugLocked: !!draft.slugLocked,
      base: base || "",
      origin: origin || draft.origin || "",
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
  if (!server) return sameDraft(draft, { title: "", slug: "", description: "", body: "", tags: "", enums: {} }) ? null : draft;
  // Edit: only restore a draft taken from this server version. An empty
  // base is the editor's first paint, not a user edit.
  if (!draft.base || draft.base !== server.updatedAt) return null;
  return sameDraft(draft, server) ? null : draft;
}

// Capture (snippets v2, F3): a selection becomes a snippet title without
// asking. First words, markdown lead-ins dropped, one line, ends cut at a
// word boundary — the reader can always edit it in the editor.
export function titleFromText(text) {
  const s = String(text == null ? "" : text).replace(/\s+/g, " ").trim();
  if (!s) return "";
  const cut = s.replace(/^[#>*\-+`\d.\s]+/, "").trim() || s;
  const words = cut.split(" ").filter(Boolean);
  let t = "";
  for (const w of words) {
    if ((t + " " + w).trim().length > 60) break;
    t = (t + " " + w).trim();
  }
  t = (t || words[0] || "").slice(0, 60).trim().replace(/[.,;:!?]+$/, "");
  return t;
}

// Placeholder detection for import (F7). Other tools spell their slots
// [LIKE_THIS] or LIKE_THIS; the suggestion turns the ones we find into
// {{lower_snake}} — never inside an existing {{placeholder}}, and never
// rewriting anything the reader did not accept.
const CONV_BRACKET = /\[([A-Za-z][A-Za-z0-9 ._-]{0,40})\]/g;
const CONV_UPPER = /\b([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)\b/g;
const CONV_KINDS = Object.freeze({ bracket: "[BRACKETS]", upper: "UPPER_CASE" });

// placeholderRanges marks every {{…}} span so conversions skip them.
function placeholderRanges(s) {
  const out = [];
  let i = 0;
  while (i < s.length) {
    if (hasAt(s, i, "{{{{") || hasAt(s, i, "}}}}")) { i += 4; continue; }
    if (hasAt(s, i, "{{")) {
      const r = readPlaceholder(s, i);
      if (r.err) return out;
      out.push([i, r.i]);
      i = r.i;
      continue;
    }
    i++;
  }
  return out;
}

function inRanges(ranges, start, end) {
  return ranges.some(([a, b]) => start < b && end > a);
}

// convName lowercases and snake-cases a captured token into a valid name
// (a name must start with a letter or _ — ADR-0130 grammar).
export function convName(raw) {
  const s = String(raw == null ? "" : raw).trim().toLowerCase();
  let out = "";
  for (const ch of s) {
    if ((ch >= "a" && ch <= "z") || (ch >= "0" && ch <= "9") || ch === "_") out += ch;
    else if (out && !out.endsWith("_")) out += "_";
  }
  out = out.replace(/_+/g, "_").replace(/^_+|_+$/g, "");
  if (!out) return "";
  if (!(out[0] >= "a" && out[0] <= "z") && out[0] !== "_") out = "v_" + out;
  return out.slice(0, SNIP_LIMITS.name);
}

function scanConversions(body, kind) {
  const s = String(body == null ? "" : body);
  // A bracket token and the UPPER_CASE inside it are one slot, not two:
  // scanning UPPER_CASE skips what the bracket pass already claimed, so the
  // suggestions never double-count the same placeholder.
  const ranges = placeholderRanges(s);
  if (kind === "upper") {
    const re0 = CONV_BRACKET;
    re0.lastIndex = 0;
    let m0;
    while ((m0 = re0.exec(s)) !== null) {
      if (!inRanges(ranges, m0.index, m0.index + m0[0].length)) ranges.push([m0.index, m0.index + m0[0].length]);
    }
  }
  const re = kind === "bracket" ? CONV_BRACKET : CONV_UPPER;
  re.lastIndex = 0;
  const hits = [];
  let m;
  while ((m = re.exec(s)) !== null) {
    const start = m.index;
    const end = start + m[0].length;
    if (inRanges(ranges, start, end)) continue;
    const name = convName(m[1]);
    if (!name) continue;
    hits.push({ start, end, raw: m[0], name });
  }
  return hits;
}

// detectConversions reports what an import would change, per convention.
// A whole-body scan (not per line): a token is a token wherever it sits.
export function detectConversions(body) {
  const kinds = [];
  for (const kind of ["bracket", "upper"]) {
    const hits = scanConversions(body, kind);
    if (!hits.length) continue;
    const names = [];
    for (const h of hits) if (!names.includes(h.name)) names.push(h.name);
    kinds.push({ kind, label: CONV_KINDS[kind], count: hits.length, names, sample: hits.slice(0, 3).map((h) => h.raw) });
  }
  return { kinds };
}

// applyConversions rewrites the body for the accepted conventions. A
// token already inside an existing {{placeholder}} stays untouched; the
// rewrite is applied right-to-left so earlier spans keep their offsets.
export function applyConversions(body, accepted) {
  const s = String(body == null ? "" : body);
  const kinds = Array.isArray(accepted) ? accepted : [];
  const hits = kinds.flatMap((k) => scanConversions(s, k));
  if (!hits.length) return s;
  hits.sort((a, b) => b.start - a.start);
  let out = s;
  let last = s.length + 1;
  for (const h of hits) {
    if (h.end > last) continue; // overlapping with an already-applied span
    out = out.slice(0, h.start) + "{{" + h.name + "}}" + out.slice(h.end);
    last = h.start;
  }
  return out;
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

// encodeDefault escapes brace pairs so a default value survives the
// grammar: `{{` is written `{{{{` (literal {{), `}}` as `}}}}`. Single
// braces are literal in defaults and stay untouched.
export function encodeDefault(value) {
  return String(value == null ? "" : value).replace(/\{\{/g, "{{{{").replace(/\}\}/g, "}}}}");
}

// setDefaultInBody rewrites the FIRST occurrence of {{name}} in body to
// carry (or, with optional=false, drop) a default. The body is the
// source of truth; the placeholder table is a projection that calls
// this. Unknown name or invalid body → body unchanged.
export function setDefaultInBody(body, name, value, optional) {
  const s = String(body == null ? "" : body);
  let i = 0;
  while (i < s.length) {
    if (hasAt(s, i, "{{{{") || hasAt(s, i, "}}}}")) { i += 4; continue; }
    if (hasAt(s, i, "{{")) {
      const r = readPlaceholder(s, i);
      if (r.err) return s;
      if (r.ph.name === name) {
        const inner = optional ? "=" + encodeDefault(value) : "";
        return s.slice(0, i) + "{{" + name + inner + "}}" + s.slice(r.i);
      }
      i = r.i;
      continue;
    }
    i++;
  }
  return s;
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
