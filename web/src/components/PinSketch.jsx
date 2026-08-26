import { useEffect, useRef, useState } from "react";
import { Excalidraw, exportToBlob, convertToExcalidrawElements } from "@excalidraw/excalidraw";
import "@excalidraw/excalidraw/index.css";

function theme() {
  return (typeof document !== "undefined" && document.documentElement.dataset.theme === "light") ? "light" : "dark";
}

async function imageToScene(url) {
  const res = await fetch(url);
  const blob = await res.blob();
  const dataURL = await new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(r.result);
    r.onerror = reject;
    r.readAsDataURL(blob);
  });
  const img = await new Promise((resolve, reject) => {
    const el = new Image();
    el.onload = () => resolve(el);
    el.onerror = reject;
    el.src = dataURL;
  });
  const fileId = "bg" + Date.now();
  const elements = convertToExcalidrawElements([{
    type: "image",
    fileId,
    x: 0,
    y: 0,
    width: img.naturalWidth || 800,
    height: img.naturalHeight || 600,
  }]);
  return {
    elements,
    files: { [fileId]: { mimeType: blob.type || "image/png", id: fileId, dataURL, created: Date.now() } },
    appState: { viewBackgroundColor: theme() === "dark" ? "#121212" : "#ffffff" },
  };
}

export default function PinSketch({ open, title, initial, backgroundURL, onSave, onClose }) {
  const apiRef = useRef(null);
  const [seed, setSeed] = useState(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) { setSeed(null); return; }
    let stop = false;
    (async () => {
      if (initial) { setSeed(initial); return; }
      if (backgroundURL) {
        try {
          const sc = await imageToScene(backgroundURL);
          if (!stop) setSeed(sc);
        } catch {
          if (!stop) setSeed({});
        }
        return;
      }
      setSeed({});
    })();
    return () => { stop = true; };
  }, [open, initial, backgroundURL]);

  if (!open) return null;

  async function save() {
    const api = apiRef.current;
    if (!api) return;
    setBusy(true);
    try {
      const elements = api.getSceneElements();
      const files = api.getFiles();
      const app = api.getAppState();
      const blob = await exportToBlob({ elements, appState: app, files, mimeType: "image/png", exportPadding: 16 });
      const scene = {
        type: "excalidraw",
        version: 2,
        elements,
        appState: { viewBackgroundColor: app.viewBackgroundColor },
        files,
      };
      await onSave({ scene, preview: blob });
    } finally { setBusy(false); }
  }

  return (
    <div className="pin-sketch">
      <header className="pin-sketch-head">
        <h2>{title || "Sketch"}</h2>
        <div className="pin-sketch-actions">
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary btn-sm" disabled={busy || !seed} onClick={save}>Save</button>
        </div>
      </header>
      <div className="pin-sketch-board">
        {seed ? (
          <Excalidraw
            excalidrawAPI={(api) => { apiRef.current = api; }}
            initialData={seed}
            theme={theme()}
            UIOptions={{ canvasActions: { loadScene: false, saveToActiveFile: false } }}
          />
        ) : <p className="pin-sketch-wait">Loading…</p>}
      </div>
    </div>
  );
}
