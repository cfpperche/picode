// Markdown as a document (the file preview), not as a chat message: the
// pieces GitHub renders around the CommonMark core — frontmatter, heading
// anchors, alerts, links and images that point into the repository. Pure
// functions and hast plugins, so both apps render the same document and the
// rules are tested without a DOM.

// Every id the document carries starts with this prefix, as on GitHub: a
// heading called "App" must not become an element whose id the app itself
// looks up. rehype-sanitize applies the same prefix to ids the author wrote.
export const DOC_ID_PREFIX = "user-content-";

const FRONT = /^﻿?---[ \t]*\r?\n(?:([\s\S]*?)\r?\n)?(?:---|\.\.\.)[ \t]*(?:\r?\n|$)/;
const FRONT_ROW = /^([A-Za-z_][\w.-]*)[ \t]*:[ \t]*(.*)$/;

// splitFrontmatter separates a leading YAML block from the body. `rows` is
// set only when every line is a plain `key: value` — the shape GitHub shows as
// a table; anything nested stays raw text.
export function splitFrontmatter(text) {
  const src = String(text || "");
  const m = FRONT.exec(src);
  if (!m) return { front: null, rows: null, body: src, bodyLine: 1 };
  const front = m[1] || "";
  const lines = front.split(/\r?\n/).filter((l) => l.trim() !== "");
  let rows = [];
  for (const line of lines) {
    const row = FRONT_ROW.exec(line);
    if (!row) { rows = null; break; }
    rows.push([row[1], unquote(row[2].trim())]);
  }
  // bodyLine: the file line the body starts on, for source-line mapping.
  return { front, rows, body: src.slice(m[0].length), bodyLine: m[0].split("\n").length };
}

function unquote(v) {
  if (v.length >= 2 && (v[0] === '"' || v[0] === "'") && v[v.length - 1] === v[0]) return v.slice(1, -1);
  return v;
}

// createSlugger follows github-slugger: lowercase, drop punctuation and
// symbols, spaces become dashes, repeats get -1, -2…
export function createSlugger() {
  const seen = new Map();
  return function slug(text) {
    const base = String(text || "").toLowerCase().replace(/[^\p{L}\p{M}\p{N}\p{Pc}\- ]/gu, "").replace(/ /g, "-");
    let out = base;
    let n = seen.get(base) || 0;
    while (seen.has(out)) out = base + "-" + ++n;
    seen.set(base, n);
    seen.set(out, 0);
    return out;
  };
}

export function hastText(node) {
  if (!node) return "";
  if (node.type === "text") return node.value || "";
  return (node.children || []).map(hastText).join("");
}

const HEADING = /^h[1-6]$/;

// rehypeDocHeadings gives every heading a stable id so the outline, the
// hover anchor and `#section` links all agree. Runs after sanitizing.
export function rehypeDocHeadings() {
  return (tree) => {
    const slug = createSlugger();
    walk(tree, (node) => {
      if (node.type === "element" && HEADING.test(node.tagName)) {
        node.properties = { ...node.properties, id: DOC_ID_PREFIX + slug(hastText(node).trim()) };
      }
    });
  };
}

function walk(node, fn) {
  fn(node);
  for (const child of node.children || []) walk(child, fn);
}

export const ALERT_KINDS = { note: "Note", tip: "Tip", important: "Important", warning: "Warning", caution: "Caution" };
const ALERT_MARK = /^\[!(note|tip|important|warning|caution)\][ \t]*(?:\r?\n|$)/i;

// rehypeGithubAlerts turns `> [!NOTE]` blockquotes into alerts. GitHub only
// honours them at the top level of the document, so nested quotes stay quotes.
export function rehypeGithubAlerts() {
  return (tree) => {
    for (const node of tree.children || []) {
      if (node.type !== "element" || node.tagName !== "blockquote") continue;
      const p = (node.children || []).find((c) => c.type === "element");
      if (!p || p.tagName !== "p") continue;
      const first = p.children && p.children[0];
      if (!first || first.type !== "text") continue;
      const m = ALERT_MARK.exec(first.value);
      if (!m) continue;
      const kind = m[1].toLowerCase();
      first.value = first.value.slice(m[0].length);
      if (!first.value) {
        p.children.shift();
        if (p.children[0] && p.children[0].type === "element" && p.children[0].tagName === "br") p.children.shift();
      }
      if (!p.children.some((c) => c.type !== "text" || c.value.trim())) {
        node.children = node.children.filter((c) => c !== p);
      }
      node.tagName = "div";
      node.properties = { className: ["md-alert", "md-alert-" + kind] };
      node.children.unshift({
        type: "element",
        tagName: "p",
        properties: { className: ["md-alert-title"], dataAlert: kind },
        children: [{ type: "text", value: ALERT_KINDS[kind] }],
      });
    }
  };
}

// anchorTargets lists the ids a `#frag` link may mean: the prefixed id this
// document writes, then the bare one (a footnote or an author's raw HTML).
export function anchorTargets(frag) {
  let id = String(frag || "").replace(/^#/, "");
  try { id = decodeURIComponent(id); } catch { /* keep as written */ }
  if (!id) return [];
  return id.startsWith(DOC_ID_PREFIX) ? [id] : [DOC_ID_PREFIX + id, id];
}

const SCHEME = /^([a-z][a-z0-9+.-]*):/i;
const WEB = new Set(["http", "https", "mailto"]);

// resolveDocLink says where a link in the document at `fromPath` (relative to
// the tree root) goes: a heading of this document, the web, another file of
// the same tree, or nowhere — a scheme we do not follow, or a path that climbs
// out of the root.
export function resolveDocLink(fromPath, href) {
  const raw = String(href || "").trim();
  if (!raw) return { kind: "none" };
  if (raw.startsWith("#")) return { kind: "anchor", id: raw.slice(1) };
  if (raw.startsWith("//")) return { kind: "external", href: "https:" + raw };
  const scheme = SCHEME.exec(raw);
  if (scheme) return WEB.has(scheme[1].toLowerCase()) ? { kind: "external", href: raw } : { kind: "none" };
  const hashAt = raw.indexOf("#");
  const id = hashAt >= 0 ? raw.slice(hashAt + 1) : "";
  let pathPart = hashAt >= 0 ? raw.slice(0, hashAt) : raw;
  const q = pathPart.indexOf("?");
  if (q >= 0) pathPart = pathPart.slice(0, q);
  if (!pathPart) return id ? { kind: "anchor", id } : { kind: "none" };
  let decoded = pathPart;
  try { decoded = decodeURIComponent(pathPart); } catch { /* keep as written */ }
  const from = String(fromPath || "").split("/").filter(Boolean);
  from.pop();
  const parts = decoded.startsWith("/") ? [] : from;
  for (const seg of decoded.split("/")) {
    if (!seg || seg === ".") continue;
    if (seg === "..") {
      if (!parts.length) return { kind: "none" };
      parts.pop();
      continue;
    }
    parts.push(seg);
  }
  if (!parts.length || decoded.endsWith("/")) return { kind: "none" };
  return { kind: "file", path: parts.join("/"), id };
}

// resolveDocImage is resolveDocLink for `src`: the web, inline data images,
// or a file of the same tree the preview fetches through the file API.
export function resolveDocImage(fromPath, src) {
  const raw = String(src || "").trim();
  if (/^data:image\//i.test(raw)) return { kind: "external", href: raw };
  const link = resolveDocLink(fromPath, raw);
  if (link.kind === "external" && link.href.toLowerCase().startsWith("mailto:")) return { kind: "none" };
  return link.kind === "anchor" ? { kind: "none" } : link;
}

const SOURCE_BLOCKS = new Set(["p", "h1", "h2", "h3", "h4", "h5", "h6", "li", "pre", "blockquote", "table", "tr", "hr", "div", "details", "dt", "dd", "section"]);

// rehypeSourceLines stamps each block with the file line it starts on
// (`data-line`), so a preview can be scrolled to match an editor and back.
// Runs last: it only reads positions the parser left on the nodes.
export function rehypeSourceLines({ offset = 0 } = {}) {
  return (tree) => {
    walk(tree, (node) => {
      if (node.type !== "element" || !SOURCE_BLOCKS.has(node.tagName)) return;
      const line = node.position && node.position.start && node.position.start.line;
      if (line) node.properties = { ...node.properties, dataLine: line + offset };
    });
  };
}
