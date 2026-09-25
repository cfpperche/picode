import { lineFor, previewTopFor, syncBlocks } from "../domain/scrollSync.js";

// A programmatic scroll fires its own scroll event; the side that was just
// moved ignores events for this long so the two panes never chase each other.
const ECHO_MS = 120;

// The desktop pane scrolls `.file-preview`; the phone scrolls the box around
// it. By default the preview's scroll box is the first of the two that exists.
const defaultBox = (host) => host.querySelector(".file-preview") || host;

// attachSplitSync ties a CodeMirror editor and a markdown preview together,
// the way VS Code's side-by-side preview does: scrolling either one scrolls
// the other to the same source line, and a double-click in the preview puts
// the editor's cursor on the line that block came from. The preview's blocks
// carry `data-line` (MarkdownDoc's `sourceLines`); `host` contains the
// preview, which may mount late, and `findBox(host)` names its scroll box.
// Returns the function that detaches it; each app wraps it in its own hook.
export function attachSplitSync(cm, host, findBox = defaultBox) {
  const sd = cm.scrollDOM;
  let echo = { side: "", until: 0 };
  let lead = "editor";
  let frame = 0;

  const previewBox = () => findBox(host);
  const schedule = (fn) => {
    if (frame) cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => { frame = 0; fn(); });
  };
  const moved = (side) => { echo = { side, until: performance.now() + ECHO_MS }; };
  const isEcho = (side) => echo.side === side && performance.now() < echo.until;
  const atTop = (el) => el.scrollTop <= 0;
  const atEnd = (el) => el.scrollTop + el.clientHeight >= el.scrollHeight - 1;

  function blocks(box) {
    const base = box.getBoundingClientRect().top - box.scrollTop;
    return syncBlocks([...box.querySelectorAll("[data-line]")].map((el) => {
      const r = el.getBoundingClientRect();
      return { line: Number(el.dataset.line), top: r.top - base, height: r.height };
    }));
  }
  // Heights in CodeMirror are measured from the document's top, which sits
  // a little below the scroller's (content padding).
  const docOffset = () => cm.documentTop - sd.getBoundingClientRect().top + sd.scrollTop;

  function editorLine() {
    const y = sd.scrollTop - docOffset();
    const block = cm.lineBlockAtHeight(Math.max(0, y));
    const n = cm.state.doc.lineAt(block.from).number;
    return n + Math.min(1, Math.max(0, (y - block.top) / Math.max(1, block.height)));
  }

  function fromEditor() {
    const box = previewBox();
    if (!box) return;
    let top;
    if (atTop(sd)) top = 0;
    else if (atEnd(sd)) top = box.scrollHeight;
    else top = previewTopFor(blocks(box), editorLine());
    if (Math.abs(box.scrollTop - top) < 1) return;
    moved("preview");
    box.scrollTop = top;
  }

  function fromPreview(box) {
    let y;
    if (atTop(box)) y = 0;
    else if (atEnd(box)) y = sd.scrollHeight;
    else {
      const doc = cm.state.doc;
      const line = Math.min(doc.lines, Math.max(1, lineFor(blocks(box), box.scrollTop, doc.lines)));
      const n = Math.floor(line);
      const block = cm.lineBlockAt(doc.line(n).from);
      y = block.top + (line - n) * block.height + docOffset();
    }
    if (Math.abs(sd.scrollTop - y) < 1) return;
    moved("editor");
    sd.scrollTop = y;
  }

  const onEditor = () => {
    if (isEcho("editor")) return;
    lead = "editor";
    schedule(fromEditor);
  };
  const onPreview = (e) => {
    const box = e.target;
    if (!(box instanceof HTMLElement) || box !== previewBox()) return;
    if (isEcho("preview")) return;
    lead = "preview";
    schedule(() => fromPreview(box));
  };
  // The preview re-renders as the text changes (and mounts late, being a
  // lazy chunk): keep it where the editor is.
  const onRender = () => { if (lead === "editor") schedule(fromEditor); };
  const onDouble = (e) => {
    const el = e.target instanceof Element ? e.target.closest("[data-line]") : null;
    if (!el || !host.contains(el)) return;
    const doc = cm.state.doc;
    const n = Math.min(doc.lines, Math.max(1, Number(el.dataset.line) || 1));
    const pos = doc.line(n).from;
    // The block the reader clicked stays where it is: the editor brings its
    // line level with it, and the preview is told not to follow.
    const box = previewBox();
    const dy = box ? el.getBoundingClientRect().top - box.getBoundingClientRect().top : 0;
    const y = cm.lineBlockAt(pos).top + docOffset() - dy;
    lead = "preview";
    moved("editor");
    sd.scrollTop = Math.max(0, y);
    cm.dispatch({ selection: { anchor: pos } });
    cm.focus({ preventScroll: true });
  };

  sd.addEventListener("scroll", onEditor, { passive: true });
  host.addEventListener("scroll", onPreview, { capture: true, passive: true });
  host.addEventListener("dblclick", onDouble);
  const watch = new MutationObserver(onRender);
  watch.observe(host, { childList: true, subtree: true });
  schedule(fromEditor);
  return () => {
    sd.removeEventListener("scroll", onEditor);
    host.removeEventListener("scroll", onPreview, { capture: true });
    host.removeEventListener("dblclick", onDouble);
    watch.disconnect();
    if (frame) cancelAnimationFrame(frame);
  };
}
