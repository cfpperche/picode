import { useEffect, useRef, useState } from "react";
import { EditorView } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { fileEditorExtensions } from "../lib/fileEditor.js";
import { languageFor } from "../lib/fileLang.js";
import { previewKind, previewEmpty } from "@picode/shared/domain/filePreview.js";
import { usePreviewTicket } from "../lib/usePreviewTicket.js";
import FilePreview from "./FilePreview.jsx";

export default function FileDocument({ doc, view, path, owner, root }) {
  const host = useRef(null), editor = useRef(null);
  const kind = previewKind(path);
  const [display, setDisplay] = useState(kind ? "preview" : "edit");
  useEffect(() => { setDisplay(kind ? "preview" : "edit"); }, [doc, kind]);
  // A document too large for the text read still has a page to show: the
  // ticket serves it from disk, so the pane renders preview-only — no editor
  // and no "too large to display" notice.
  const previewOnly = kind === "html" && view.kind === "msg" && /too large/i.test(view.error || "");
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
      ] }), parent: host.current,
    });
    editor.current = cm;
    // Touching the editor is the only action that focuses its input.
    return () => { cm.destroy(); editor.current = null; };
  }, [doc, path, view.kind, view.revision]);
  useEffect(() => { if (display === "edit") editor.current?.requestMeasure(); }, [display]);
  return <div className="m-file-document">
    {kind && (view.kind === "text" || previewOnly) ? <div className="m-file-display" role="group" aria-label="File display" data-align-row>
      {view.kind === "text" ? (<>
        <button type="button" className="btn btn-sm" aria-pressed={display === "preview"} onClick={() => setDisplay("preview")}>Preview</button>
        <button type="button" className="btn btn-sm" aria-pressed={display === "edit"} onClick={() => setDisplay("edit")}>Edit</button>
      </>) : null}
      {kind === "html" && display === "preview" && !view.dirty && (previewOnly || !previewEmpty(view.text)) ? (
        <button type="button" className="btn btn-sm" disabled={html.status === "loading"} onClick={() => { void html.reload(); }}>Reload</button>
      ) : null}
      {kind === "html" && display === "preview" && !view.dirty && html.status === "ready" ? (
        <button type="button" className="btn btn-sm" onClick={() => window.open(html.url, "_blank", "noopener,noreferrer")}>Open</button>
      ) : null}
    </div> : null}
    {view.kind === "load" ? <div className="m-files-loading" role="status" aria-label="Loading file"><span /><span /><span /></div> : null}
    {view.kind === "text" ? <div className="m-file-editor" ref={host} hidden={display !== "edit"} /> : null}
    {kind && display === "preview" && (previewOnly || ["text", "bin"].includes(view.kind)) ? <div className="m-file-preview">
      {kind === "html" && !previewOnly && previewEmpty(view.text) ? (
        <p className="file-pane-msg">Nothing to preview.</p>
      ) : (
        <FilePreview key={path + ":" + view.src} kind={kind} text={view.text} src={view.src} html={kind === "html" ? html : undefined} />
      )}
    </div> : null}
  </div>;
}
