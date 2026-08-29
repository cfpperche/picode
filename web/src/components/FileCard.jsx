import { useEffect, useState } from "react";
import { previewKind, isBlobKind, fileBlobUrl } from "../lib/filePreview.js";
import FilePreview from "./FilePreview.jsx";
import { IconExternal } from "./Icons.jsx";
import { humanizeError } from "../lib/api.js";
import { api } from "../lib/api.js";

function fileTextUrl(agentId, path) {
  return "/api/agents/" + encodeURIComponent(agentId) + "/text?path=" + encodeURIComponent(path);
}

export default function FileCard({ agentId, path, onClose, onOpenTab }) {
  const [view, setView] = useState({ kind: "load" });
  const kind = previewKind(path);
  const name = (path || "").split("/").pop() || path || "File";

  useEffect(() => {
    if (!agentId || !path) return;
    let stop = false;
    const blobRef = { current: "" };
    setView({ kind: "load" });
    if (isBlobKind(kind)) {
      fetch(fileBlobUrl(agentId, "", path))
        .then(async (res) => {
          if (stop) return;
          if (!res.ok) {
            let msg = res.statusText;
            try { msg = (await res.json()).error || msg; } catch { /* keep */ }
            throw new Error(msg);
          }
          const buf = await res.blob();
          blobRef.current = URL.createObjectURL(buf);
          if (stop) { URL.revokeObjectURL(blobRef.current); return; }
          setView({ kind: "bin", src: blobRef.current, name });
        })
        .catch((err) => {
          if (stop) return;
          setView({ kind: "msg", name, text: cardMsg(err && err.message) });
        });
    } else {
      api(fileTextUrl(agentId, path))
        .then((page) => {
          if (stop) return;
          setView({ kind: "text", text: page.text || "", name: page.name || name });
        })
        .catch((err) => {
          if (stop) return;
          setView({ kind: "msg", name, text: cardMsg(err && err.message) });
        });
    }
    return () => {
      stop = true;
      if (blobRef.current) URL.revokeObjectURL(blobRef.current);
    };
  }, [agentId, path, kind, name]);

  const gone = view.kind === "msg";
  const title = view.name || name;
  return (
    <section className="file-card" aria-label={title}>
      <header className="file-card-bar" data-align-row>
        <span className="file-pane-name" title={path}>{title}</span>
        {!gone ? (
          <button type="button" className="btn btn-ghost btn-sm file-card-out" onClick={() => onOpenTab && onOpenTab(path)}>
            <IconExternal />
            Open in tab
          </button>
        ) : null}
        <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Close</button>
      </header>
      <div className="file-card-body">
        {view.kind === "load" ? (
          <div className="file-skel" aria-hidden="true">
            <div className="skel-line w-80" />
            <div className="skel-line w-50" />
          </div>
        ) : null}
        {view.kind === "msg" ? <p className="file-pane-msg">{view.text}</p> : null}
        {view.kind === "bin" && kind ? <FilePreview kind={kind} src={view.src} /> : null}
        {view.kind === "text" && kind ? <FilePreview kind={kind} text={view.text} /> : null}
        {view.kind === "text" && !kind ? (
          <pre className="file-card-pre">{excerpt(view.text)}</pre>
        ) : null}
      </div>
    </section>
  );
}

function excerpt(text) {
  const lines = String(text || "").split("\n");
  const cut = lines.slice(0, 12).join("\n");
  return lines.length > 12 ? cut + "\n…" : cut;
}

function cardMsg(raw) {
  const s = String(raw || "").toLowerCase();
  if (s.includes("gone") || s.includes("not found") || s.includes("no such file")) return "That file is gone.";
  if (s.includes("too large")) return "This file is too large.";
  if (s.includes("escapes")) return "That path is outside this project.";
  if (s.includes("can't show") || s.includes("unsupported")) return "Can't show this file.";
  return humanizeError(raw);
}
