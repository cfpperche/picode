import { syntaxTree } from "@codemirror/language";

// Math and Mermaid for the Live view, planned as data like mdLiveTables.js.
// The markdown grammar has no math, so `$$` blocks and `$…$` spans are found
// in the text and kept out of code; Mermaid is a fenced block whose info
// string is `mermaid`. Blocks become block widgets in the state field that
// also draws tables; inline math is a per-line widget in the view plugin.

// Nodes whose text is never math: code, raw HTML, link targets.
const NOT_MATH = new Set(["InlineCode", "FencedCode", "CodeBlock", "CodeText", "HTMLBlock", "HTMLTag", "CommentBlock", "Comment", "URL", "Autolink", "LinkLabel"]);

function insideNonMath(tree, pos) {
  for (let n = tree.resolveInner(pos, 1); n; n = n.parent) if (NOT_MATH.has(n.name)) return true;
  return false;
}

const touches = (active, doc, from, to) => {
  for (let n = doc.lineAt(from).number, end = doc.lineAt(to).number; n <= end; n++) if (active.has(n)) return true;
  return false;
};

// planMathBlocks finds `$$` display blocks (a line that opens with `$$`, up
// to the line that closes with `$$`; or one line `$$ … $$`) that no active
// line touches. `lastFrom` is where the cursor lands coming up from below.
export function planMathBlocks(state, active) {
  const doc = state.doc;
  const tree = syntaxTree(state);
  const out = [];
  for (let n = 1; n <= doc.lines; n++) {
    const line = doc.line(n);
    const open = /^ {0,3}\$\$/.exec(line.text);
    if (!open || insideNonMath(tree, line.from)) continue;
    const rest = line.text.slice(open[0].length);
    let end = null;
    let tex;
    if (/\$\$\s*$/.test(rest)) {
      end = line;
      tex = rest.replace(/\$\$\s*$/, "");
    } else {
      const body = [rest];
      for (let m = n + 1; m <= doc.lines; m++) {
        const l = doc.line(m);
        const close = /^(.*?)\$\$\s*$/.exec(l.text);
        if (close) { body.push(close[1]); end = l; break; }
        body.push(l.text);
      }
      tex = body.join("\n");
    }
    if (!end) continue;
    if (!touches(active, doc, line.from, end.to)) out.push({ kind: "math", from: line.from, to: end.to, lastFrom: end.from, tex: tex.trim() });
    n = end.number;
  }
  return out;
}

// planMermaid finds ```mermaid fences that no active line touches.
export function planMermaid(state, active) {
  const doc = state.doc;
  const out = [];
  syntaxTree(state).iterate({
    enter(ref) {
      if (ref.name !== "FencedCode") return ref.name === "Document" || ref.name === "Blockquote" || /List|ListItem/.test(ref.name) ? undefined : false;
      const info = ref.node.getChild("CodeInfo");
      if (!info || doc.sliceString(info.from, info.to).trim().toLowerCase() !== "mermaid") return false;
      const marks = ref.node.getChildren("CodeMark");
      if (marks.length < 2) return false;
      const first = doc.lineAt(ref.from);
      const last = doc.lineAt(ref.to);
      if (touches(active, doc, first.from, last.to)) return false;
      const text = ref.node.getChild("CodeText");
      out.push({ kind: "mermaid", from: first.from, to: last.to, lastFrom: last.from, src: text ? doc.sliceString(text.from, text.to) : "" });
      return false;
    },
  });
  return out;
}

// Inline `$…$` follows pandoc's rule, which keeps prices apart from math: the
// opening `$` is followed by a non-space, the closing `$` follows a non-space
// and is not followed by a digit; `\$` is a dollar sign; `$$` is not inline.
const INLINE = /(?<![\\$])\$(?![\s$])((?:\\.|[^\\$\n])*?[^\s\\$])\$(?![\d$])/g;

// planInlineMath lists inline math in `ranges`, replaced by a rendered widget
// off the cursor line and marked (source kept) on it.
export function planInlineMath(state, ranges, active) {
  const doc = state.doc;
  const tree = syntaxTree(state);
  const out = [];
  for (const { from, to } of ranges) {
    for (let n = doc.lineAt(from).number, end = doc.lineAt(to).number; n <= end; n++) {
      const line = doc.line(n);
      if (!line.text.includes("$") || /^ {0,3}\$\$/.test(line.text)) continue;
      for (const m of line.text.matchAll(INLINE)) {
        const at = line.from + m.index;
        if (insideNonMath(tree, at)) continue;
        out.push({ from: at, to: at + m[0].length, tex: m[1], replace: !active.has(n) });
      }
    }
  }
  return out;
}
