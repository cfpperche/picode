import { useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Excalidraw, exportToBlob } from "@excalidraw/excalidraw";
import "@excalidraw/excalidraw/index.css";

function theme() {
  return (typeof document !== "undefined" && document.documentElement.dataset.theme === "light") ? "light" : "dark";
}

// The attach composer's sketch pad (ADR-0089 prompt door). It borrows the
// Excalidraw dependency, not the pin studio (owner decision, 2026-09-12):
// no background picture, no pin tables, no scene kept anywhere but the
// caller's memory — the artifact that leaves is the PNG. `initial` is the
// scene of the chip being edited, if any.
export default function SketchEditor({ open, title, initial, confirmLabel, onSave, onClose }) {
  const apiRef = useRef(null);
  const [busy, setBusy] = useState(false);

  if (!open) return null;

  // Excalidraw reads initialData on mount; the parent mounts this only for
  // the open editor, so every open is a fresh scene.
  const seed = initial && initial.elements
    ? initial
    : { appState: { viewBackgroundColor: theme() === "dark" ? "#121212" : "#ffffff" } };

  async function save() {
    const api = apiRef.current;
    if (!api) return;
    setBusy(true);
    try {
      const elements = api.getSceneElements();
      const files = api.getFiles();
      const app = api.getAppState();
      const blob = await exportToBlob({ elements, appState: app, files, mimeType: "image/png", exportPadding: 16 });
      await onSave({
        scene: {
          type: "excalidraw",
          version: 2,
          elements,
          appState: { viewBackgroundColor: app.viewBackgroundColor },
          files,
        },
        preview: blob,
      });
    } finally {
      setBusy(false);
    }
  }

  return createPortal(
    <div className="sketch-editor">
      <header className="sketch-editor-head">
        <h2>{title || "Sketch"}</h2>
        <div className="sketch-editor-actions">
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={save}>{confirmLabel || "Insert"}</button>
        </div>
      </header>
      <div className="sketch-editor-board">
        <Excalidraw
          excalidrawAPI={(api) => { apiRef.current = api; }}
          initialData={seed}
          theme={theme()}
          UIOptions={{ canvasActions: { loadScene: false, saveToActiveFile: false } }}
        />
      </div>
    </div>,
    document.body,
  );
}
