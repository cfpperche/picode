import { useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Excalidraw, exportToBlob } from "@excalidraw/excalidraw";
import "@excalidraw/excalidraw/index.css";

function theme() {
  return (typeof document !== "undefined" && document.documentElement.dataset.theme === "light") ? "light" : "dark";
}

// The shell root owns the visual viewport and the safe areas (ADR-0044):
// a body portal is fixed to the layout viewport, which on iOS
// (viewport-fit=cover) can sit under the status bar and leave the
// home-indicator strip unpainted — the pad's header overlapped the clock
// and the drawing area broke past the usable screen. #m-app is `position:
// fixed` with overflow hidden, so an absolute child fills exactly the area
// the shell already reserves.
function portalRoot() {
  return document.getElementById("m-app") || document.body;
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
  // the open editor, so every open is a fresh scene. A phone opens the pad
  // to draw, so the pen is the default tool (no toolbar tap first).
  //
  // The canvas is white in both themes: Excalidraw's dark theme inverts the
  // bitmap (invert(.93) hue-rotate(180deg)), so a dark viewBackgroundColor
  // came back as a light canvas under a dark UI — and the exported PNG
  // carried that inverted paper. White keeps the export a plain sheet and
  // lets the theme filter darken the screen.
  const seed = initial && initial.elements
    ? { ...initial, appState: { ...(initial.appState || {}), viewBackgroundColor: "#ffffff", activeTool: { type: "freedraw" } } }
    : { appState: { viewBackgroundColor: "#ffffff", activeTool: { type: "freedraw" } } };

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
    <div className="sketch-editor" role="dialog" aria-modal="true" aria-label={title || "Sketch"}>
      <header className="sketch-editor-head">
        <h2>{title || "Sketch"}</h2>
        <div className="sketch-editor-actions">
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={save}>{confirmLabel || "Insert"}</button>
        </div>
      </header>
      <div className="sketch-editor-board">
        {/* The pad inserts into the composer, so the canvas menu keeps only
            what belongs to the drawing: background and clear. */}
        <Excalidraw
          excalidrawAPI={(api) => { apiRef.current = api; }}
          initialData={seed}
          theme={theme()}
          UIOptions={{ canvasActions: { loadScene: false, saveToActiveFile: false, export: false, saveAsImage: false } }}
        />
      </div>
    </div>,
    portalRoot(),
  );
}
