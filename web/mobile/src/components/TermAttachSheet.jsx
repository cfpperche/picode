import { useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import WorkspaceAttach from "./WorkspaceAttach.jsx";
import { IconClip, IconFile, IconImage, IconSend, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { planAttachFiles, readAttachFile, MAX_ATTACH } from "@picode/shared/domain/termPrompt.js";
import { toast, toastError } from "../lib/toast.js";

const json = (body) => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

export default function TermAttachSheet({ term, open, onClose }) {
  const imgPick = useRef(null);
  const filePick = useRef(null);
  const [items, setItems] = useState([]);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [pick, setPick] = useState(false);

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

  async function send() {
    if (busy || (!text.trim() && !items.length)) return;
    setBusy(true);
    try {
      const paths = [];
      for (const it of items) {
        if (it.path) { paths.push(it.path); continue; }
        const d = await api("/api/terminals/" + encodeURIComponent(term.id) + "/drop", json({ name: it.name, mime: it.mime, data: it.data }));
        paths.push(d.path);
      }
      await api("/api/terminals/" + encodeURIComponent(term.id) + "/prompt", json({ message: text, paths }));
      setItems([]);
      setText("");
      toast.ok("Sent to the terminal.");
      onClose();
    } catch (e) {
      toastError(e);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-sheet" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Send to the terminal</Dialog.Title>
          <WorkspaceAttach open={pick} termId={term.id} onPick={addWorkspace} onClose={() => setPick(false)} />
          {items.length ? (
            <div className="term-attach-chips">
              {items.map((it) => (
                <span key={it.id} className="pin-att composer-pic">
                  <span className="pin-att-face" title={it.name}>
                    {it.image && it.url ? <img src={it.url} alt="" /> : <span className="pin-att-ext">{extOf(it.name)}</span>}
                  </span>
                  <button type="button" className="pin-att-x" title="Remove" onClick={() => setItems((cur) => cur.filter((x) => x.id !== it.id))}><IconX size={12} /></button>
                </span>
              ))}
            </div>
          ) : (
            <p className="term-attach-empty">Add a photo or a file.</p>
          )}
          <form className="term-attach-row" data-align-row noValidate onSubmit={(e) => { e.preventDefault(); send(); }}>
            <input ref={imgPick} type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/*" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
            <input ref={filePick} type="file" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
            <button type="button" className="icon-btn composer-attach" title="Attach image" aria-label="Attach image" onClick={() => imgPick.current && imgPick.current.click()}><IconImage /></button>
            <button type="button" className="icon-btn composer-attach" title="Attach file" aria-label="Attach file" onClick={() => filePick.current && filePick.current.click()}><IconFile /></button>
            <button type="button" className="icon-btn composer-attach" title="Attach from folder" aria-label="Attach from folder" onClick={() => setPick(true)}><IconClip /></button>
            <input
              className="term-attach-input"
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Message the terminal"
              aria-label="Message the terminal"
              autoComplete="off"
            />
            <button type="submit" className="icon-btn icon-btn-send" title="Send" disabled={busy || (!text.trim() && !items.length)}><IconSend size={16} /></button>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function extOf(name) {
  const m = /\.([a-z0-9]+)$/i.exec(name || "");
  return (m ? m[1] : "file").slice(0, 4).toUpperCase();
}
