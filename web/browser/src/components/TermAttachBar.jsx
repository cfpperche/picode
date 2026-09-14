import { lazy, Suspense, useEffect, useRef, useState } from "react";
import WorkspaceAttach from "./WorkspaceAttach.jsx";
import { IconClip, IconFile, IconImage, IconSend, IconSketch, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { planAttachFiles, readAttachFile, MAX_ATTACH } from "@picode/shared/domain/termPrompt.js";
import { fitAttachField, isAttachSendKey } from "@picode/shared/domain/attachText.js";
import { sceneHasInk } from "@picode/shared/domain/composerImage.js";
import { toast, toastError } from "../lib/toast.js";

// Excalidraw loads with the sketch, never with the terminal pane.
const SketchEditor = lazy(() => import("./SketchEditor.jsx"));

const json = (body) => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

// The prompt door of ADR-0089, now opened on demand from the terminal's
// context menu instead of standing between the pane and the window edge.
// `seed` carries what the menu started it with — a selected line as the
// message, a larger selection as a staged text file (lib/termMenu.js) — and
// is applied once per token so asking twice never wipes a typed question.
export default function TermAttachBar({ term, seed, ownerKind, onClose }) {
  const agentId = ownerKind === "agent" ? term.id : "";
  const termId = ownerKind === "agent" ? "" : term.id;
  const dropBase = agentId
    ? "/api/agents/" + encodeURIComponent(agentId)
    : "/api/terminals/" + encodeURIComponent(termId);
  const imgPick = useRef(null);
  const filePick = useRef(null);
  const inputRef = useRef(null);
  const seeded = useRef("");
  const [items, setItems] = useState([]);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [pick, setPick] = useState(false);
  const [sketch, setSketch] = useState(null);

  useEffect(() => { if (inputRef.current) inputRef.current.focus(); }, []);

  // One line at rest, four lines maximum (shared/domain/attachText.js).
  useEffect(() => { fitAttachField(inputRef.current); }, [text]);

  useEffect(() => {
    if (!seed || !seed.token || seeded.current === seed.token) return;
    seeded.current = seed.token;
    if (seed.text) setText((cur) => (cur.trim() ? cur.replace(/\s*$/, " ") + seed.text : seed.text));
    if (seed.files && seed.files.length) addFiles(seed.files);
    if (inputRef.current) inputRef.current.focus();
  }, [seed]);

  async function addFiles(list) {
    const plan = planAttachFiles(list, items.length);
    if (plan.tooMany) toast.error("Up to 4 files.");
    if (plan.tooLarge) toast.error("Each file must be under 4 MB.");
    if (!plan.files.length) return;
    const next = items.slice();
    for (const f of plan.files) {
      try {
        const row = await readAttachFile(f);
        next.push({ id: (crypto.randomUUID && crypto.randomUUID()) || String(Date.now()) + next.length, ...row });
      } catch (err) {
        if (err && err.message === "too-large") toast.error("Each file must be under 4 MB.");
      }
    }
    setItems(next);
  }

  function addWorkspace(hit) {
    setPick(false);
    if (!hit || !hit.path) return;
    if (items.length >= MAX_ATTACH) { toast.error("Up to 4 files."); return; }
    setItems((cur) => cur.concat([{
      id: (crypto.randomUUID && crypto.randomUUID()) || String(Date.now()),
      name: hit.name,
      path: hit.path,
      image: /\.(png|jpe?g|gif|webp)$/i.test(hit.name || ""),
    }]));
  }

  function openSketch(edit) {
    if (!edit && items.length >= MAX_ATTACH) { toast.error("Up to 4 files."); return; }
    setSketch(edit || {});
  }

  // The sketch leaves as a PNG attachment; the scene stays in this session
  // only, so the chip can be reopened and edited before Send.
  async function insertSketch({ scene, preview }) {
    if (!sceneHasInk(scene && scene.elements)) { toast.error("Draw something first."); return; }
    try {
      const row = await readAttachFile(new File([preview], "sketch.png", { type: "image/png" }));
      setItems((cur) => {
        if (sketch && sketch.id) return cur.map((x) => (x.id === sketch.id ? { ...row, id: x.id, scene } : x));
        if (cur.length >= MAX_ATTACH) return cur;
        return cur.concat([{ ...row, id: (crypto.randomUUID && crypto.randomUUID()) || String(Date.now()), scene }]);
      });
      setSketch(null);
    } catch (err) {
      if (err && err.message === "too-large") toast.error("Each file must be under 4 MB.");
    }
  }

  async function send() {
    if (busy || (!text.trim() && !items.length)) return;
    setBusy(true);
    try {
      const paths = [];
      for (const it of items) {
        if (it.path) { paths.push(it.path); continue; }
        const d = await api(dropBase + "/drop", json({ name: it.name, mime: it.mime, data: it.data }));
        paths.push(d.path);
      }
      // No success toast: the user is looking at the terminal and sees the
      // message land. Toasts stay reserved for failures (toastError below).
      await api(dropBase + "/prompt", json({ message: text, paths }));
      setItems([]);
      setText("");
    } catch (e) {
      toastError(e);
    } finally {
      setBusy(false);
    }
  }

  // Escape gives the pane its full height back. The folder picker owns the
  // key while it is open, so it closes first.
  function onKeyDown(e) {
    if (e.key !== "Escape" || pick) return;
    e.stopPropagation();
    if (onClose) onClose();
  }

  return (
    <div className="term-attach" onKeyDown={onKeyDown}>
      <WorkspaceAttach open={pick} agentId={agentId || undefined} termId={termId || undefined} onPick={addWorkspace} onClose={() => setPick(false)} />
      <div className="term-attach-head">
        {items.length ? (
          <div className="term-attach-chips">
            {items.map((it) => (
              <span key={it.id} className="pin-att composer-pic">
                {it.scene ? (
                  <button type="button" className="pin-att-face" title="Edit sketch" onClick={() => openSketch({ id: it.id, scene: it.scene })}>
                    <img src={it.url} alt="" />
                  </button>
                ) : (
                  <span className="pin-att-face" title={it.name}>
                    {it.image && it.url ? <img src={it.url} alt="" /> : <span className="pin-att-ext">{extOf(it.name)}</span>}
                  </span>
                )}
                <button type="button" className="pin-att-x" title="Remove" onClick={() => setItems((cur) => cur.filter((x) => x.id !== it.id))}><IconX size={12} /></button>
              </span>
            ))}
          </div>
        ) : (
          <p className="term-attach-empty">Add a photo, a file or a sketch.</p>
        )}
        <button type="button" className="ws-icon-btn" title="Close (Esc)" aria-label="Close the message bar" onClick={onClose}><IconX size={13} /></button>
      </div>
      {/* Not a [data-align-row]: the field grows by design (overlayAudit's
          equal-height rule is for fixed controls). The row is bottom-aligned. */}
      <form className="term-attach-row" noValidate onSubmit={(e) => { e.preventDefault(); send(); }}>
        <input ref={imgPick} type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/*" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
        <input ref={filePick} type="file" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
        <button type="button" className="icon-btn composer-attach" title="Attach image" aria-label="Attach image" onClick={() => imgPick.current && imgPick.current.click()}><IconImage /></button>
        <button type="button" className="icon-btn composer-attach" title="Attach file" aria-label="Attach file" onClick={() => filePick.current && filePick.current.click()}><IconFile /></button>
        <button type="button" className="icon-btn composer-attach" title="Attach from folder" aria-label="Attach from folder" onClick={() => setPick(true)}><IconClip /></button>
        <button type="button" className="icon-btn composer-attach" title="Sketch" aria-label="Sketch" onClick={() => openSketch()}><IconSketch /></button>
        <textarea
          ref={inputRef}
          className="term-attach-input"
          rows={1}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => { if (isAttachSendKey(e)) { e.preventDefault(); send(); } }}
          placeholder="Message the terminal"
          aria-label="Message the terminal"
          autoComplete="off"
        />
        <button type="submit" className="icon-btn icon-btn-send" title="Send (Enter)" disabled={busy || (!text.trim() && !items.length)}><IconSend size={16} /></button>
      </form>
      {sketch ? (
        <Suspense fallback={null}>
          <SketchEditor
            open
            title={sketch.id ? "Edit sketch" : "Sketch"}
            initial={sketch.scene || null}
            confirmLabel="Insert"
            onSave={insertSketch}
            onClose={() => setSketch(null)}
          />
        </Suspense>
      ) : null}
    </div>
  );
}

function extOf(name) {
  const m = /\.([a-z0-9]+)$/i.exec(name || "");
  return (m ? m[1] : "file").slice(0, 4).toUpperCase();
}
