import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { EditorView } from "@codemirror/view";
import { Compartment, EditorState } from "@codemirror/state";
import { fileEditorExtensions } from "../lib/fileEditor.js";
import { languageFor } from "../lib/fileLang.js";
import { previewKind, previewEmpty } from "@picode/shared/domain/filePreview.js";
import { usePreviewTicket } from "../lib/usePreviewTicket.js";
import FilePreview from "./FilePreview.jsx";
import { ownerFileURL } from "../lib/fileIO.js";
import { markdownLive } from "@picode/shared/editor/mdLive.js";
import { headingPos } from "@picode/shared/editor/mdLivePlan.js";
import { attachSplitSync } from "@picode/shared/editor/splitSync.js";
import { resolveDocLink } from "@picode/shared/domain/mdDocument.js";

// Markdown opens the way the reader last read markdown (the desktop pane
// keeps the same key; its "raw" is this screen's Edit).
const MD_VIEW_KEY = "picode-md-view";
function initialDisplay(kind) {
  if (!kind) return "edit";
  if (kind !== "markdown") return "preview";
  try {
    const saved = localStorage.getItem(MD_VIEW_KEY);
    if (saved === "live" || saved === "split") return saved;
    if (saved === "raw") return "edit";
  } catch { /* storage unavailable */ }
  return "preview";
}

export default function FileDocument({ doc, view, path, owner, root, onOpenPath }) {
  const host = useRef(null), editor = useRef(null), previewHost = useRef(null);
  const kind = previewKind(path);
  const markdown = kind === "markdown";
  const [display, setDisplayState] = useState(() => initialDisplay(kind));
  const setDisplay = (next) => {
    setDisplayState(next);
    if (markdown) { try { localStorage.setItem(MD_VIEW_KEY, next === "edit" ? "raw" : next); } catch { /* storage unavailable */ } }
  };
  useEffect(() => { setDisplayState(initialDisplay(kind)); }, [doc, kind]);
  const [cmTick, setCmTick] = useState(0);
  // Live is one compartment of the same editor: Edit ↔ Live keeps the text,
  // the cursor and the undo history.
  const liveComp = useMemo(() => new Compartment(), []);
  const liveOn = useRef(false);
  const displayRef = useRef(display);
  displayRef.current = display;
  const liveCtx = useRef(null);
  // A document too large for the text read still has a page to show: the
  // ticket serves it from disk, so the pane renders preview-only — no editor
  // and no "too large to display" notice.
  // Images a markdown file points at come from the same tree, through the
  // same file API.
  const assetUrl = useCallback((p, resource = "blob") => (owner ? ownerFileURL(owner, resource, p, root) : ""), [owner, root]);
  const previewOnly = kind === "html" && view.kind === "msg" && /too large/i.test(view.error || "");
  liveCtx.current = {
    path,
    assetUrl,
    openLink(href) {
      const to = resolveDocLink(path, href);
      const cm = editor.current;
      if (to.kind === "external") window.open(to.href, "_blank", "noopener,noreferrer");
      else if (to.kind === "file") onOpenPath?.(to.path);
      else if (to.kind === "anchor" && cm) {
        const pos = headingPos(cm.state, to.id);
        if (pos >= 0) cm.dispatch({ selection: { anchor: pos }, effects: EditorView.scrollIntoView(pos, { y: "start", yMargin: 16 }) });
      }
    },
  };
  // One capability ticket per open HTML preview (ADR-0136). Unsaved editor
  // text travels as the ticket's overlay (the hook PUTs it), so the preview
  // shows what the editor holds while assets still come from disk.
  const html = usePreviewTicket({
    ownerKind: owner && owner.kind,
    ownerId: owner && owner.id,
    path,
    root,
    text: view.text,
    dirty: view.dirty,
    enabled: kind === "html" && display === "preview" && (previewOnly || !previewEmpty(view.text)),
  });
  useEffect(() => {
    if (view.kind !== "text" || !host.current) return;
    const cm = new EditorView({
      state: EditorState.create({ doc: doc.getSnapshot().text, extensions: [
        ...fileEditorExtensions({ lang: languageFor(path), dark: document.documentElement.dataset.theme !== "light", onDoc: () => doc.edit(cm.state.doc.toString()), onSave: () => doc.save() }),
        EditorView.contentAttributes.of({ "aria-label": "File contents", spellcheck: "false", autocorrect: "off", autocapitalize: "off" }),
        liveComp.of(displayRef.current === "live" ? markdownLive(() => liveCtx.current) : []),
      ] }), parent: host.current,
    });
    editor.current = cm;
    liveOn.current = displayRef.current === "live";
    setCmTick((t) => t + 1);
    // Touching the editor is the only action that focuses its input.
    return () => { cm.destroy(); editor.current = null; };
  }, [doc, path, view.kind, view.revision]);
  useEffect(() => { if (display !== "preview") editor.current?.requestMeasure(); }, [display]);
  useEffect(() => {
    const cm = editor.current;
    const want = display === "live";
    if (!cm || liveOn.current === want) return;
    liveOn.current = want;
    cm.dispatch({ effects: liveComp.reconfigure(want ? markdownLive(() => liveCtx.current) : []) });
  }, [display, cmTick, liveComp]);
  // Split scrolls source and page together; on the phone the preview's box
  // (.m-file-preview) is the one that scrolls.
  useEffect(() => {
    const cm = editor.current;
    const box = previewHost.current;
    if (display !== "split" || !cm || !box) return undefined;
    return attachSplitSync(cm, box, (h) => h);
  }, [display, cmTick]);
  const editing = display !== "preview";
  const showsPreview = display === "preview" || display === "split";
  return <div className="m-file-document">
    {kind && (view.kind === "text" || previewOnly) ? <div className="m-file-display" role="group" aria-label="File display" data-align-row>
      {view.kind === "text" ? (<>
        <button type="button" className="btn btn-sm" aria-pressed={display === "preview"} onClick={() => setDisplay("preview")}>Preview</button>
        {markdown ? <button type="button" className="btn btn-sm" aria-pressed={display === "live"} onClick={() => setDisplay("live")}>Live</button> : null}
        {markdown ? <button type="button" className="btn btn-sm" aria-pressed={display === "split"} onClick={() => setDisplay("split")}>Split</button> : null}
        <button type="button" className="btn btn-sm" aria-pressed={display === "edit"} onClick={() => setDisplay("edit")}>Edit</button>
      </>) : null}
      {kind === "html" && display === "preview" && !view.dirty && (previewOnly || !previewEmpty(view.text)) ? (
        <button type="button" className="btn btn-sm" disabled={html.status === "loading"} onClick={() => { void html.refresh(); }}>Reload</button>
      ) : null}
      {kind === "html" && display === "preview" && !view.dirty && html.status === "ready" ? (
        <button type="button" className="btn btn-sm" onClick={() => window.open(html.url, "_blank", "noopener,noreferrer")}>Open</button>
      ) : null}
    </div> : null}
    {view.kind === "load" ? <div className="m-files-loading" role="status" aria-label="Loading file"><span /><span /><span /></div> : null}
    <div className={"m-file-body" + (display === "split" ? " is-split" : "")}>
    {view.kind === "text" ? <div className="m-file-editor" ref={host} hidden={!editing} /> : null}
    {kind && showsPreview && (previewOnly || ["text", "bin"].includes(view.kind)) ? <div className="m-file-preview" ref={previewHost}>
      {kind === "html" && !previewOnly && previewEmpty(view.text) ? (
        <p className="file-pane-msg">Nothing to preview.</p>
      ) : (
        <FilePreview key={path + ":" + view.src} kind={kind} text={view.text} src={view.src} html={kind === "html" ? html : undefined} path={path} assetUrl={assetUrl} onOpenPath={onOpenPath} sourceLines={display === "split"} />
      )}
    </div> : null}
    </div>
  </div>;
}
