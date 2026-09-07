import { useEffect, useRef, useState } from "react";
import { EditorView } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { fileEditorExtensions } from "../lib/fileEditor.js";
import { languageFor } from "../lib/fileLang.js";
import { previewKind } from "@picode/shared/domain/filePreview.js";
import FilePreview from "./FilePreview.jsx";

export default function FileDocument({ doc, view, path }) {
  const host = useRef(null), editor = useRef(null);
  const kind = previewKind(path);
  const [display, setDisplay] = useState(kind ? "preview" : "edit");
  useEffect(() => { setDisplay(kind ? "preview" : "edit"); }, [doc, kind]);
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
    {kind && view.kind === "text" ? <div className="m-file-display" role="group" aria-label="File display" data-align-row>
      <button type="button" className="btn btn-sm" aria-pressed={display === "preview"} onClick={() => setDisplay("preview")}>Preview</button>
      <button type="button" className="btn btn-sm" aria-pressed={display === "edit"} onClick={() => setDisplay("edit")}>Edit</button>
    </div> : null}
    {view.kind === "load" ? <div className="m-files-loading" role="status" aria-label="Loading file"><span /><span /><span /></div> : null}
    {view.kind === "text" ? <div className="m-file-editor" ref={host} hidden={display !== "edit"} /> : null}
    {kind && display === "preview" && ["text", "bin"].includes(view.kind) ? <div className="m-file-preview"><FilePreview key={path + ":" + view.src} kind={kind} text={view.text} src={view.src} /></div> : null}
  </div>;
}
