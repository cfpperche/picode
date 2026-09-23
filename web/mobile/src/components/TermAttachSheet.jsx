import { lazy, Suspense, useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import WorkspaceAttach from "./WorkspaceAttach.jsx";
import { IconClip, IconFile, IconImage, IconSend, IconSketch, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { planAttachFiles, readAttachFile, MAX_ATTACH } from "@picode/shared/domain/termPrompt.js";
import { fitAttachField } from "@picode/shared/domain/attachText.js";
import { sceneHasInk } from "@picode/shared/domain/composerImage.js";
import { terminalOwnerBase } from "@picode/shared/domain/agentTerminal.js";
import { isSubmitKey } from "../lib/agentDrafts.js";
import { toast, toastError } from "../lib/toast.js";
import { createUseDeliveryModes } from "@picode/shared/client/useDeliveryModes.js";
import { deliveryNotice, deliveryPlaceholder } from "@picode/shared/domain/deliveryModes.js";

const useDeliveryModes = createUseDeliveryModes({ useEffect, useState });

// Excalidraw loads with the sketch, never with the terminal screen.
const SketchEditor = lazy(() => import("./SketchEditor.jsx"));

const json = (body) => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

export default function TermAttachSheet({ term, owner = { kind: "term", id: term.id }, open, onClose }) {
  const base = terminalOwnerBase(owner);
  const imgPick = useRef(null);
  const filePick = useRef(null);
  const fieldRef = useRef(null);
  const [items, setItems] = useState([]);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [sendError, setSendError] = useState("");
  const [pick, setPick] = useState(false);
  const [sketch, setSketch] = useState(null);
  // Steer / Follow-up while the CLI works (ADR-0206); read when the sheet
  // opens, hidden when the CLI is idle.
  const { options, delivery, setDelivery } = useDeliveryModes(open ? base : "");
  const placeholder = deliveryPlaceholder(delivery, "Message the terminal");

  // One line at rest, four lines maximum (shared/domain/attachText.js).
  useEffect(() => { fitAttachField(fieldRef.current); }, [text]);

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
        const d = await api(base + "/drop", json({ name: it.name, mime: it.mime, data: it.data }));
        paths.push(d.path);
      }
      // No success toast: the user is looking at the terminal and sees the
      // message land. Toasts stay reserved for failures and for the one
      // receipt the pane cannot show: a prompt PiCode could not confirm.
      const res = await api(base + "/prompt", json({ message: text, paths, delivery }));
      const notice = deliveryNotice(res, delivery);
      if (notice) toast.warn(notice);
      setItems([]);
      setText("");
      setSendError("");
      onClose();
    } catch (e) {
      // Door refusals name their fix — show them inside the sheet, above
      // the composer, where the Send happened.
      setSendError(e && e.message ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
    {/* The sketch pad replaces the sheet while it is open: vaul/Radix treats
        a pointerdown outside the dialog as a dismiss, and the pad is a body
        sibling, so leaving the sheet mounted closed it under the drawing.
        State (items, text) lives in this component and survives. */}
    <Dialog.Root open={!sketch && !!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-sheet" onCloseAutoFocus={(e) => e.preventDefault()}>
          <div className="term-attach-heading">
            <Dialog.Title className="dlg-title">Send to the terminal</Dialog.Title>
            <button type="button" className="m-tool-icon" aria-label="Close attachments" title="Close attachments" onClick={onClose}><IconX size={16} /></button>
          </div>
          {sendError ? <p className="dlg-body" role="alert" style={{ color: "var(--danger)", margin: "0 0 8px" }}>{sendError}</p> : null}
          <WorkspaceAttach open={pick} termId={owner.kind === "term" ? owner.id : undefined} agentId={owner.kind === "agent" ? owner.id : undefined} onPick={addWorkspace} onClose={() => setPick(false)} />
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
          <form className="term-attach-form" noValidate onSubmit={(e) => { e.preventDefault(); send(); }}>
            <input ref={imgPick} type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/*" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
            <input ref={filePick} type="file" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
            <div className="term-attach-row" data-align-row>
              <button type="button" className="icon-btn composer-attach term-attach-choice" title="Attach image" aria-label="Attach image" onClick={() => imgPick.current && imgPick.current.click()}><IconImage /><span>Photo</span></button>
              <button type="button" className="icon-btn composer-attach term-attach-choice" title="Attach file" aria-label="Attach file" onClick={() => filePick.current && filePick.current.click()}><IconFile /><span>File</span></button>
              <button type="button" className="icon-btn composer-attach term-attach-choice" title="Attach from folder" aria-label="Attach from folder" onClick={() => setPick(true)}><IconClip /><span>Folder</span></button>
              <button type="button" className="icon-btn composer-attach term-attach-choice" title="Sketch" aria-label="Sketch" onClick={() => openSketch()}><IconSketch /><span>Sketch</span></button>
            </div>
            {options.length ? (
              <div className="m-composer-delivery" data-align-row>
                <label htmlFor="attach-kind">Delivery</label>
                <select id="attach-kind" value={delivery} onChange={(e) => setDelivery(e.target.value)}>
                  {options.map((o) => <option key={o.id} value={o.id}>{o.label}</option>)}
                </select>
              </div>
            ) : null}
            {/* Not a [data-align-row]: the field grows by design
                (overlayAudit's equal-height rule is for fixed controls). */}
            <div className="term-attach-row">
              <textarea
                ref={fieldRef}
                className="term-attach-input"
                rows={1}
                data-vaul-no-drag
                value={text}
                onChange={(e) => setText(e.target.value)}
                onKeyDown={(e) => { if (isSubmitKey(e.nativeEvent)) { e.preventDefault(); send(); } }}
                placeholder={placeholder}
                aria-label={placeholder}
                autoComplete="off"
              />
              <button type="submit" className="icon-btn icon-btn-send" title="Send · Ctrl/⌘ Enter" disabled={busy || (!text.trim() && !items.length)}><IconSend size={16} /></button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
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
    </>
  );
}

function extOf(name) {
  const m = /\.([a-z0-9]+)$/i.exec(name || "");
  return (m ? m[1] : "file").slice(0, 4).toUpperCase();
}
