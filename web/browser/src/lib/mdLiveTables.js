import { syntaxTree } from "@codemirror/language";

// Tables for the Live view. A table spans lines, and only a state field may
// replace across line breaks, so tables are planned here — apart from
// mdLivePlan.js's per-line rules — and drawn as one block widget whenever no
// selection touches the table (see mdLive.js). Pure data, tested without a DOM.

// Containers a table may sit in; everything else is skipped whole.
const DESCEND = new Set(["Document", "Blockquote", "BulletList", "OrderedList", "ListItem"]);

// planTables lists every GFM table in the document that no active line
// touches: its full-line range, its cells (text and source position, so a
// click can land in the right cell) and each column's alignment.
export function planTables(state, active) {
  const doc = state.doc;
  const out = [];
  syntaxTree(state).iterate({
    enter(ref) {
      if (ref.name === "Table") {
        const first = doc.lineAt(ref.from);
        const last = doc.lineAt(ref.to);
        for (let n = first.number; n <= last.number; n++) if (active.has(n)) return false;
        const table = readTable(state, ref.node);
        let quoted = false;
        for (let p = ref.node.parent; p; p = p.parent) if (p.name === "Blockquote") quoted = true;
        if (table.rows.length) out.push({ from: first.from, to: last.to, quoted, ...table });
        return false;
      }
      return DESCEND.has(ref.name) ? undefined : false;
    },
  });
  return out;
}

function readTable(state, node) {
  const text = (from, to) => state.doc.sliceString(from, to);
  const rows = [];
  let align = [];
  for (let child = node.firstChild; child; child = child.nextSibling) {
    if (child.name === "TableHeader" || child.name === "TableRow") {
      rows.push({
        header: child.name === "TableHeader",
        cells: child.node.getChildren("TableCell").map((c) => ({ text: text(c.from, c.to).trim().replace(/\\\|/g, "|"), from: c.from })),
        // An empty cell has no node; the row's start is where a click lands.
        from: child.from,
      });
    } else if (child.name === "TableDelimiter" && child.node.parent === node) {
      align = text(child.from, child.to).split("|").map((c) => c.trim()).filter(Boolean).map(alignOf);
    }
  }
  const width = Math.max(align.length, ...rows.map((r) => r.cells.length));
  for (const r of rows) while (r.cells.length < width) r.cells.push({ text: "", from: r.from });
  while (align.length < width) align.push("");
  return { rows, align };
}

function alignOf(spec) {
  const left = spec.startsWith(":");
  const right = spec.endsWith(":");
  return left && right ? "center" : right ? "right" : left ? "left" : "";
}

// inlineTokens splits a cell's markdown into the few inline forms a table
// cell usually carries — code, bold, italic, strike and links — so the widget
// can build DOM nodes without ever parsing HTML. Anything else stays text.
const INLINE = /(`+)([\s\S]*?[^`])\1(?!`)|\*\*([^*]+)\*\*|__([^_]+)__|~~([^~]+)~~|\*([^*\s][^*]*)\*|_([^_\s][^_]*)_|!?\[([^\]]*)\]\(([^)\s]*)(?:\s+"[^"]*")?\)|<(https?:\/\/[^>\s]+)>/g;

export function inlineTokens(src) {
  const out = [];
  let at = 0;
  const s = String(src || "");
  for (const m of s.matchAll(INLINE)) {
    if (m.index > at) out.push({ type: "text", text: s.slice(at, m.index) });
    if (m[1]) out.push({ type: "code", text: m[2].replace(/^ (.*) $/, "$1") });
    else if (m[3] || m[4]) out.push({ type: "strong", text: m[3] || m[4] });
    else if (m[5]) out.push({ type: "strike", text: m[5] });
    else if (m[6] || m[7]) out.push({ type: "em", text: m[6] || m[7] });
    else if (m[10]) out.push({ type: "link", text: m[10], href: m[10] });
    else out.push(m[0].startsWith("!") ? { type: "text", text: m[8] } : { type: "link", text: m[8], href: m[9] });
    at = m.index + m[0].length;
  }
  if (at < s.length) out.push({ type: "text", text: s.slice(at) });
  return out;
}

// tableSkip: CodeMirror's vertical motion steps over a block widget. When a
// move from `head` to `target` would cross a rendered table, it stops on the
// table's near edge instead — its first line going down, its last going up —
// which puts the cursor in the table and turns it back into source.
export function tableSkip(tables, head, target, forward) {
  for (const t of forward ? tables : [...tables].reverse()) {
    if (forward && t.from > head && t.from <= target) return t.from;
    if (!forward && t.to < head && t.to >= target) return lastLineStart(t);
  }
  return null;
}

function lastLineStart(t) {
  const rows = t.rows;
  return rows.length ? rows[rows.length - 1].from : t.from;
}
