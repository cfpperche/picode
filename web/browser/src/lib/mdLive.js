import { Decoration, EditorView, ViewPlugin, WidgetType, keymap } from "@codemirror/view";
import { Prec, StateEffect, StateField } from "@codemirror/state";
import { syntaxTree } from "@codemirror/language";
import { api } from "@picode/shared/client/api.js";
import { svgDataUrl } from "@picode/shared/domain/filePreview.js";
import { resolveDocImage } from "@picode/shared/domain/mdDocument.js";
import { safeImgSrc } from "@picode/shared/domain/mdSafe.js";
import { activeLines, planLive } from "./mdLivePlan.js";
import { inlineTokens, planTables, tableSkip } from "./mdLiveTables.js";

const MAC = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform || "");
const OPEN_HINT = (MAC ? "⌘" : "Ctrl") + "+click to open";

// markdownLive is the Live view: the markdown stays the file's own text, but
// headings, emphasis, links, lists, quotes, code and images render in place,
// and their syntax appears only on the line being edited (after Obsidian's
// Live Preview). `context()` is read at use time — { path, assetUrl,
// openLink(href) } — so the owner can change without reconfiguring.
export function markdownLive(context) {
  const plugin = ViewPlugin.fromClass(class {
    constructor(view) { this.decorations = build(view, context); }
    update(u) {
      if (u.docChanged || u.viewportChanged || u.selectionSet || u.focusChanged || syntaxTree(u.startState) !== syntaxTree(u.state)) {
        this.decorations = build(u.view, context);
      }
    }
  }, { decorations: (v) => v.decorations });

  return [
    plugin,
    tableField(context),
    Prec.high(keymap.of([
      { key: "ArrowDown", run: (view) => stepIntoTable(view, true) },
      { key: "ArrowUp", run: (view) => stepIntoTable(view, false) },
    ])),
    EditorView.focusChangeEffect.of((_state, focusing) => setFocus.of(focusing)),
    EditorView.editorAttributes.of({ class: "cm-md-live" }),
    EditorView.domEventHandlers({
      mousedown(e, view) {
        if (e.button !== 0 || !(e.ctrlKey || e.metaKey)) return false;
        const pos = view.posAtCoords({ x: e.clientX, y: e.clientY });
        const href = pos == null ? "" : linkAt(view.state, pos);
        if (!href) return false;
        e.preventDefault();
        context().openLink?.(href);
        return true;
      },
    }),
  ];
}

function build(view, context) {
  const specs = planLive(view.state, view.visibleRanges, activeLines(view.state, view.hasFocus));
  const doc = view.state.doc;
  const decos = [];
  for (const s of specs) {
    // A replacement may not cross a line break from a view plugin.
    if (s.from !== undefined && s.to > s.from && doc.lineAt(s.from).number !== doc.lineAt(s.to).number) continue;
    switch (s.kind) {
      case "line": decos.push(Decoration.line({ class: s.cls }).range(s.at)); break;
      case "mark": decos.push(Decoration.mark({ class: s.cls, attributes: s.href ? { title: OPEN_HINT } : undefined }).range(s.from, s.to)); break;
      case "hide": decos.push(Decoration.replace({}).range(s.from, s.to)); break;
      case "label": decos.push(Decoration.replace({ widget: new LabelWidget(s.text, s.cls) }).range(s.from, s.to)); break;
      case "bullet": decos.push(Decoration.replace({ widget: new BulletWidget() }).range(s.from, s.to)); break;
      case "task": decos.push(Decoration.replace({ widget: new TaskWidget(s.checked) }).range(s.from, s.to)); break;
      case "image": {
        const w = new ImageWidget(s.url, s.alt, context);
        decos.push(s.replace ? Decoration.replace({ widget: w }).range(s.from, s.to) : Decoration.widget({ widget: w, side: 1 }).range(s.to));
        break;
      }
      default: break;
    }
  }
  return Decoration.set(decos, true);
}

// linkAt finds the target of the link, autolink or bare URL under `pos`.
function linkAt(state, pos) {
  for (let node = syntaxTree(state).resolveInner(pos, 1); node; node = node.parent) {
    if (node.name === "Link" || node.name === "Autolink" || node.name === "Image") {
      const url = node.getChild("URL");
      return url ? state.doc.sliceString(url.from, url.to) : "";
    }
    if (node.name === "URL") return state.doc.sliceString(node.from, node.to);
  }
  return "";
}

class LabelWidget extends WidgetType {
  constructor(text, cls) { super(); this.text = text; this.cls = cls; }
  eq(o) { return o.text === this.text && o.cls === this.cls; }
  toDOM() {
    const el = document.createElement("span");
    el.className = this.cls;
    el.textContent = this.text;
    return el;
  }
}

class BulletWidget extends WidgetType {
  eq() { return true; }
  toDOM() {
    const el = document.createElement("span");
    el.className = "cm-md-bullet";
    el.textContent = "•";
    return el;
  }
}

// A checkbox that edits the file: `[ ]` ↔ `[x]`, one undoable change.
class TaskWidget extends WidgetType {
  constructor(checked) { super(); this.checked = checked; }
  eq(o) { return o.checked === this.checked; }
  toDOM(view) {
    const box = document.createElement("input");
    box.type = "checkbox";
    box.className = "cm-md-task";
    box.checked = this.checked;
    box.setAttribute("aria-label", this.checked ? "Mark as not done" : "Mark as done");
    box.addEventListener("mousedown", (e) => {
      e.preventDefault();
      const pos = view.posAtDOM(box);
      const mark = view.state.doc.sliceString(pos, pos + 3);
      if (!/^\[[ xX]\]$/.test(mark)) return;
      view.dispatch({ changes: { from: pos + 1, to: pos + 2, insert: this.checked ? " " : "x" }, userEvent: "input" });
    });
    return box;
  }
  ignoreEvent() { return true; }
}

// An image from the web, or from this tree through the file API; an SVG from
// the tree is read as text and shown as a data: image (it never runs).
class ImageWidget extends WidgetType {
  constructor(url, alt, context) { super(); this.url = url; this.alt = alt; this.context = context; }
  eq(o) { return o.url === this.url && o.alt === this.alt; }
  get estimatedHeight() { return 160; }
  toDOM() {
    const wrap = document.createElement("span");
    wrap.className = "cm-md-image";
    const { path = "", assetUrl } = this.context() || {};
    const at = resolveDocImage(path, this.url);
    const img = document.createElement("img");
    img.alt = this.alt;
    img.loading = "lazy";
    const missing = () => {
      wrap.textContent = this.alt || this.url;
      wrap.className = "cm-md-image cm-md-image-missing";
    };
    img.addEventListener("error", missing);
    if (at.kind === "external" && safeImgSrc(at.href)) img.src = safeImgSrc(at.href);
    else if (at.kind === "file" && assetUrl && /\.svg$/i.test(at.path)) {
      api(assetUrl(at.path, "text"))
        .then((page) => { const u = svgDataUrl(page.text || ""); if (u) img.src = u; else missing(); })
        .catch(missing);
    } else if (at.kind === "file" && assetUrl) img.src = assetUrl(at.path, "blob");
    else { missing(); return wrap; }
    wrap.appendChild(img);
    return wrap;
  }
  ignoreEvent() { return false; }
}

// Tables: one block widget per table no selection touches. Focus is part of
// the field's state (a view plugin knows it, a state field has to be told),
// so an unfocused editor shows every table rendered.
const setFocus = StateEffect.define();

function tableField(context) {
  const decorate = (state, focused) => Decoration.set(
    planTables(state, activeLines(state, focused)).map((t) =>
      Decoration.replace({ widget: new TableWidget(t, context), block: true }).range(t.from, t.to)),
  );
  return StateField.define({
    create: (state) => ({ focused: false, deco: decorate(state, false) }),
    update(value, tr) {
      let focused = value.focused;
      for (const e of tr.effects) if (e.is(setFocus)) focused = e.value;
      if (focused === value.focused && !tr.docChanged && !tr.selection && syntaxTree(tr.startState) === syntaxTree(tr.state)) return value;
      return { focused, deco: decorate(tr.state, focused) };
    },
    provide: (f) => EditorView.decorations.from(f, (v) => v.deco),
  });
}

// Up/Down would step over a rendered table (a block widget); stop on its
// near row instead, which reveals its source like a click does.
function stepIntoTable(view, forward) {
  const range = view.state.selection.main;
  if (!range.empty || view.state.selection.ranges.length > 1) return false;
  const target = view.moveVertically(range, forward).head;
  const stop = tableSkip(planTables(view.state, activeLines(view.state, true)), range.head, target, forward);
  if (stop == null) return false;
  view.dispatch({ selection: { anchor: stop }, scrollIntoView: true, userEvent: "select" });
  return true;
}

class TableWidget extends WidgetType {
  constructor(table, context) {
    super();
    this.table = table;
    this.context = context;
    this.key = JSON.stringify([table.quoted, table.align, table.rows.map((r) => [r.header, r.cells.map((c) => c.text)])]);
  }
  eq(o) { return o.key === this.key; }
  get estimatedHeight() { return this.table.rows.length * 33 + 16; }
  toDOM(view) {
    const wrap = document.createElement("div");
    wrap.className = "cm-md-table-widget" + (this.table.quoted ? " cm-md-table-quoted" : "");
    const el = document.createElement("table");
    const head = document.createElement("thead");
    const body = document.createElement("tbody");
    this.table.rows.forEach((row, r) => {
      const tr = document.createElement("tr");
      row.cells.forEach((cell, c) => {
        const td = document.createElement(row.header ? "th" : "td");
        if (this.table.align[c]) td.style.textAlign = this.table.align[c];
        td.dataset.row = String(r);
        td.dataset.col = String(c);
        for (const tok of inlineTokens(cell.text)) td.appendChild(inlineNode(tok));
        tr.appendChild(td);
      });
      (row.header ? head : body).appendChild(tr);
    });
    if (head.childNodes.length) el.appendChild(head);
    el.appendChild(body);
    wrap.appendChild(el);
    // A click puts the cursor in the cell it landed on (the table turns back
    // into its source); Ctrl/⌘+click on a link follows it. Positions are read
    // at click time: an unchanged widget may outlive edits above it.
    wrap.addEventListener("mousedown", (e) => {
      if (e.button !== 0) return;
      const target = e.target instanceof Element ? e.target : null;
      const link = target && target.closest("[data-href]");
      if (link && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        this.context().openLink?.(link.getAttribute("data-href"));
        return;
      }
      e.preventDefault();
      const from = view.posAtDOM(wrap);
      const now = planTables(view.state, new Set()).find((t) => t.from === from);
      const td = target && target.closest("[data-row]");
      const cell = now && td ? now.rows[Number(td.dataset.row)]?.cells[Number(td.dataset.col)] : null;
      view.dispatch({ selection: { anchor: cell ? cell.from : from } });
      view.focus();
    });
    return wrap;
  }
  ignoreEvent() { return true; }
}

function inlineNode(tok) {
  if (tok.type === "text") return document.createTextNode(tok.text);
  const el = document.createElement(tok.type === "code" ? "code" : tok.type === "strong" ? "strong" : tok.type === "em" ? "em" : tok.type === "strike" ? "s" : "span");
  el.textContent = tok.text;
  if (tok.type === "code") el.className = "cm-md-code";
  if (tok.type === "link") {
    el.className = "cm-md-link";
    el.setAttribute("data-href", tok.href);
    el.title = OPEN_HINT;
  }
  return el;
}
