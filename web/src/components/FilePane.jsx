import { useEffect, useState } from "react";
import { api, humanizeError } from "../lib/api.js";

export default function FilePane({ agentId, path, onClose }) {
  const [view, setView] = useState({ kind: "load" });
  useEffect(() => {
    if (!agentId || !path) return;
    let stop = false;
    setView({ kind: "load" });
    api("/api/agents/" + agentId + "/text?path=" + encodeURIComponent(path))
      .then((page) => {
        if (!stop) setView({ kind: "text", path: page.path || path, name: page.name || "", text: page.text || "" });
      })
      .catch((err) => {
        if (stop) return;
        const raw = err && err.message ? err.message : String(err);
        setView({ kind: "msg", path, name: path.split("/").pop() || path, text: fileMsg(raw) });
      });
    return () => { stop = true; };
  }, [agentId, path]);
  const title = view.name || view.path || path || "File";
  return (
    <section className="file-pane" aria-label={title}>
      <header className="file-pane-bar">
        <span className="file-pane-name" title={view.path || path}>{title}</span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Close</button>
      </header>
      <div className="file-pane-body">
        {view.kind === "load" ? (
          <div className="file-skel" aria-hidden="true">
            <div className="skel-line w-80" />
            <div className="skel-line w-90" />
            <div className="skel-line w-50" />
            <div className="skel-line w-70" />
          </div>
        ) : null}
        {view.kind === "text" ? <pre className="file-pre">{view.text}</pre> : null}
        {view.kind === "msg" ? <p className="file-pane-msg">{view.text}</p> : null}
      </div>
    </section>
  );
}

function fileMsg(raw) {
  const s = String(raw || "").toLowerCase();
  if (s.includes("gone") || s.includes("not found") || s.includes("no such file")) return "That file is gone.";
  if (s.includes("too large")) return "This file is too large.";
  if (s.includes("can't show") || s.includes("unsupported")) return "Can't show this file.";
  if (s.includes("folder")) return "That's a folder.";
  if (s.includes("escapes")) return "That path is outside this project.";
  return humanizeError(raw);
}
