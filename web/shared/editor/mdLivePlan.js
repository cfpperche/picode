import { syntaxTree } from "@codemirror/language";
import { ALERT_KINDS, anchorTargets, createSlugger, DOC_ID_PREFIX, splitFrontmatter } from "../domain/mdDocument.js";

// planLive decides what the Live view (Obsidian-style live preview) does to
// each piece of markdown in `ranges`: style it, hide its syntax, or stand a
// small widget in for it. Syntax is only ever hidden on lines the cursor is
// not on (`active`, a Set of line numbers), so the text under the cursor is
// always the file's own. The plan is data — mdLive.js turns it into
// CodeMirror decorations — so the rules are tested without a DOM.
//
//   { kind: "line", at, cls }          class on the line starting at `at`
//   { kind: "mark", from, to, cls }    class on a span
//   { kind: "hide", from, to }         syntax removed from view
//   { kind: "label", from, to, text, cls }  span replaced by a text label
//   { kind: "bullet", from, to }       list marker drawn as a bullet
//   { kind: "task", from, to, checked }     `[ ]` drawn as a checkbox
//   { kind: "image", from, to, url, alt, replace }  image shown (replacing
//                                      its syntax when `replace`)
const ALERT = /^\[!(note|tip|important|warning|caution)\][ \t]*$/i;
const FRONT_SCAN = 20000;

export function planLive(state, ranges, active) {
  const doc = state.doc;
  const out = [];
  const lineNo = (pos) => doc.lineAt(pos).number;
  const on = (pos) => active.has(lineNo(pos));
  const anyOn = (from, to) => {
    for (let n = lineNo(from), end = lineNo(to); n <= end; n++) if (active.has(n)) return true;
    return false;
  };
  const lines = (from, to, cls) => {
    for (let n = lineNo(from), end = lineNo(to); n <= end; n++) out.push({ kind: "line", at: doc.line(n).from, cls });
  };
  const text = (from, to) => doc.sliceString(from, to);
  // A mark and the one space after it go together (`# `, `> `, `- `).
  const withSpace = (to) => (text(to, to + 1) === " " ? to + 1 : to);
  const hide = (from, to) => { if (to > from) out.push({ kind: "hide", from, to }); };

  // Frontmatter is YAML, not markdown: shown as dim source, never styled.
  const { front, bodyLine } = splitFrontmatter(text(0, Math.min(doc.length, FRONT_SCAN)));
  let bodyFrom = 0;
  if (front != null) {
    const last = Math.min(doc.lines, Math.max(1, bodyLine - 1));
    lines(0, doc.line(last).from, "cm-md-frontmatter");
    bodyFrom = bodyLine <= doc.lines ? doc.line(bodyLine).from : doc.length;
  }

  const tree = syntaxTree(state);
  for (const { from, to } of ranges) {
    tree.iterate({
      from: Math.max(from, bodyFrom),
      to,
      enter(ref) {
        const { name } = ref;
        if (name !== "Document" && ref.from < bodyFrom) return false;
        const node = ref.node;
        const heading = /^ATXHeading(\d)$/.exec(name);
        if (heading) { lines(ref.from, ref.from, "cm-md-h" + heading[1]); return undefined; }
        const setext = /^SetextHeading(\d)$/.exec(name);
        if (setext) { lines(ref.from, ref.from, "cm-md-h" + setext[1]); return undefined; }
        switch (name) {
          case "HeaderMark": {
            if (node.parent && node.parent.name.startsWith("Setext")) {
              if (!on(ref.from)) lines(ref.from, ref.from, "cm-md-rule-mark");
            } else if (!on(ref.from)) {
              // An opening `## ` takes its space; a closing ` ##` takes the one before.
              const closing = ref.from > doc.lineAt(ref.from).from;
              hide(closing && text(ref.from - 1, ref.from) === " " ? ref.from - 1 : ref.from, closing ? ref.to : withSpace(ref.to));
            }
            return false;
          }
          case "StrongEmphasis": out.push({ kind: "mark", from: ref.from, to: ref.to, cls: "cm-md-strong" }); return undefined;
          case "Emphasis": out.push({ kind: "mark", from: ref.from, to: ref.to, cls: "cm-md-em" }); return undefined;
          case "Strikethrough": out.push({ kind: "mark", from: ref.from, to: ref.to, cls: "cm-md-strike" }); return undefined;
          case "EmphasisMark":
          case "StrikethroughMark":
            if (!on(ref.from)) hide(ref.from, ref.to);
            return false;
          case "InlineCode":
            out.push({ kind: "mark", from: ref.from, to: ref.to, cls: "cm-md-code" });
            if (!on(ref.from)) for (const m of node.getChildren("CodeMark")) hide(m.from, m.to);
            return false;
          case "FencedCode": {
            lines(ref.from, ref.to, "cm-md-codeblock");
            const marks = node.getChildren("CodeMark");
            const open = marks[0];
            const close = marks.length > 1 ? marks[marks.length - 1] : null;
            lines(ref.from, ref.from, "cm-md-fence");
            if (close) lines(close.from, close.from, "cm-md-fence");
            if (!anyOn(ref.from, ref.to)) {
              const info = node.getChild("CodeInfo");
              const first = doc.lineAt(ref.from);
              out.push({ kind: "label", from: first.from, to: first.to, text: info ? text(info.from, info.to) : "", cls: "cm-md-fence-label" });
              if (close) { const last = doc.lineAt(close.from); hide(last.from, last.to); }
            } else if (open) {
              out.push({ kind: "mark", from: open.from, to: open.to, cls: "cm-md-syntax" });
            }
            return false;
          }
          case "CodeBlock":
            lines(ref.from, ref.to, "cm-md-codeblock");
            return false;
          case "Blockquote": {
            const first = doc.lineAt(ref.from);
            const body = text(first.from, first.to).replace(/^\s*>\s?/, "");
            const alert = ALERT.exec(body);
            lines(ref.from, ref.to, alert ? "cm-md-quote cm-md-alert cm-md-alert-" + alert[1].toLowerCase() : "cm-md-quote");
            if (alert && !on(ref.from)) {
              const at = first.from + text(first.from, first.to).indexOf("[");
              const kind = alert[1].toLowerCase();
              out.push({ kind: "label", from: at, to: at + alert[1].length + 3, text: ALERT_KINDS[kind], cls: "cm-md-alert-title" });
            }
            return undefined;
          }
          case "QuoteMark":
            if (!on(ref.from)) hide(ref.from, withSpace(ref.to));
            return false;
          case "ListMark": {
            const item = node.parent;
            const line = doc.lineAt(ref.from);
            if (ref.from > line.from && /^[ \t]+$/.test(text(line.from, ref.from))) {
              out.push({ kind: "mark", from: line.from, to: ref.from, cls: "cm-md-indent" });
            }
            const task = item && item.getChild("Task");
            if (task) { if (!on(ref.from)) hide(ref.from, withSpace(ref.to)); return false; }
            if (item && item.parent && item.parent.name === "BulletList") {
              if (!on(ref.from)) out.push({ kind: "bullet", from: ref.from, to: ref.to });
            } else {
              out.push({ kind: "mark", from: ref.from, to: ref.to, cls: "cm-md-listmark" });
            }
            return false;
          }
          case "TaskMarker":
            if (!on(ref.from)) out.push({ kind: "task", from: ref.from, to: ref.to, checked: /x/i.test(text(ref.from, ref.to)) });
            return false;
          case "Link": {
            const url = node.getChild("URL");
            // Only inline links are links here: `[text]` with no target is
            // plain text (and the parser also reads `[!NOTE]` this way).
            if (!url) return undefined;
            const marks = node.getChildren("LinkMark");
            if (marks.length < 2) return undefined;
            const textFrom = marks[0].to;
            const textTo = marks[1].from;
            out.push({ kind: "mark", from: textFrom, to: textTo, cls: "cm-md-link", href: text(url.from, url.to) });
            if (!on(ref.from)) { hide(ref.from, textFrom); hide(textTo, ref.to); }
            else out.push({ kind: "mark", from: textTo, to: ref.to, cls: "cm-md-syntax" });
            return undefined;
          }
          case "Image": {
            const url = node.getChild("URL");
            const marks = node.getChildren("LinkMark");
            if (!url || marks.length < 2) return false;
            const alt = text(marks[0].to, marks[1].from);
            const src = text(url.from, url.to);
            const inactive = !on(ref.from) && lineNo(ref.from) === lineNo(ref.to);
            out.push({ kind: "image", from: inactive ? ref.from : ref.to, to: ref.to, url: src, alt, replace: inactive });
            return false;
          }
          case "Autolink": {
            const url = node.getChild("URL");
            if (url) out.push({ kind: "mark", from: url.from, to: url.to, cls: "cm-md-link", href: text(url.from, url.to) });
            if (!on(ref.from)) for (const m of node.getChildren("LinkMark")) hide(m.from, m.to);
            return false;
          }
          case "URL":
            // Inside a link or image the parent already decided.
            if (node.parent && /^(Link|Image|Autolink)$/.test(node.parent.name)) return false;
            out.push({ kind: "mark", from: ref.from, to: ref.to, cls: "cm-md-link", href: text(ref.from, ref.to) });
            return false;
          case "HorizontalRule":
            if (!on(ref.from)) { lines(ref.from, ref.from, "cm-md-hr"); hide(ref.from, ref.to); }
            return false;
          case "Table":
            lines(ref.from, ref.to, "cm-md-table");
            return false;
          case "HTMLBlock":
          case "CommentBlock":
            lines(ref.from, ref.to, "cm-md-html");
            return false;
          default:
            return undefined;
        }
      },
    });
  }
  return out;
}

// activeLines is every line a selection range touches — only while the
// editor has focus, so a pane the reader is not typing in reads clean.
export function activeLines(state, focused) {
  const set = new Set();
  if (!focused) return set;
  for (const r of state.selection.ranges) {
    for (let n = state.doc.lineAt(r.from).number, end = state.doc.lineAt(r.to).number; n <= end; n++) set.add(n);
  }
  return set;
}

// headingPos finds the heading a `#fragment` link names, with the same slugs
// the preview gives its headings (github-slugger over the heading's text).
export function headingPos(state, frag) {
  const want = new Set(anchorTargets(frag).map((id) => id.startsWith(DOC_ID_PREFIX) ? id.slice(DOC_ID_PREFIX.length) : id));
  if (!want.size) return -1;
  const slug = createSlugger();
  let found = -1;
  syntaxTree(state).iterate({
    enter(ref) {
      if (found >= 0) return false;
      if (!/^(ATX|Setext)Heading\d$/.test(ref.name)) return undefined;
      const raw = state.doc.sliceString(ref.from, ref.to).split("\n")[0];
      const title = raw.replace(/^\s*#{1,6}\s*/, "").replace(/\s+#+\s*$/, "")
        .replace(/!?\[([^\]]*)\]\([^)]*\)/g, "$1").replace(/[*_~`]/g, "").trim();
      if (want.has(slug(title))) found = ref.from;
      return false;
    },
  });
  return found;
}
