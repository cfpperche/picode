import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Excalidraw, exportToBlob, convertToExcalidrawElements } from "@excalidraw/excalidraw";
import "@excalidraw/excalidraw/index.css";
import { BG_FILE_PREFIX, missingBackgroundId } from "@picode/shared/domain/pinDraft.js";

function theme() {
  return (typeof document !== "undefined" && document.documentElement.dataset.theme === "light") ? "light" : "dark";
}

// The picture behind an annotation, as the Excalidraw file entry it needs.
// Its id is "bg:" + the picture's URL, so the studio can strip it before
// upload and put it back on open: the saved scene keeps the picture by
// reference (the sketch row's baseFileId), never the bytes.
async function loadImageFile(url, fileId) {
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
  return {
    file: { mimeType: blob.type || "image/png", id: fileId, dataURL, created: Date.now() },
    width: img.naturalWidth || 800,
    height: img.naturalHeight || 600,
  };
}

async function imageToScene(url) {
  const fileId = BG_FILE_PREFIX + url;
  const { file, width, height } = await loadImageFile(url, fileId);
  const elements = convertToExcalidrawElements([{ type: "image", fileId, x: 0, y: 0, width, height }]);
  return {
    elements,
    files: { [fileId]: file },
    appState: { viewBackgroundColor: theme() === "dark" ? "#121212" : "#ffffff" },
  };
}

// A saved annotation comes back without its picture; fetch it again under
// the id the scene expects. If the fetch fails the drawing still opens —
// the picture is simply absent, which beats refusing to open at all.
async function restoreBackground(scene, url) {
  const missing = missingBackgroundId(scene);
  if (!missing || !url) return scene;
  try {
    const { file } = await loadImageFile(url, missing);
    return { ...scene, files: { ...(scene.files || {}), [missing]: file } };
  } catch {
    return scene;
  }
}

export default function PinSketch({ open, title, initial, backgroundURL, onSave, onClose, confirmLabel }) {
  const apiRef = useRef(null);
  const [seed, setSeed] = useState(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) { setSeed(null); return; }
    let stop = false;
    (async () => {
      if (initial) {
        const sc = await restoreBackground(initial, backgroundURL);
        if (!stop) setSeed(sc);
        return;
      }
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

  const ui = (
    <div className="pin-sketch">
      <header className="pin-sketch-head">
        <h2>{title || "Sketch"}</h2>
        <div className="pin-sketch-actions">
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary btn-sm" disabled={busy || !seed} onClick={save}>{confirmLabel || "Save"}</button>
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
  return createPortal(ui, document.getElementById("m-app") || document.body);
}
