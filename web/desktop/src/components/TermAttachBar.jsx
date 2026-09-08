import { useEffect, useRef, useState } from "react";
import WorkspaceAttach from "./WorkspaceAttach.jsx";
import { IconClip, IconFile, IconImage, IconSend, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { planAttachFiles, readAttachFile, MAX_ATTACH } from "@picode/shared/domain/termPrompt.js";
import { toast, toastError } from "../lib/toast.js";

const json = (body) => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

// The prompt door of ADR-0089, now opened on demand from the terminal's
// context menu instead of standing between the pane and the window edge.
// `seed` carries what the menu started it with — a selected line as the
// message, a larger selection as a staged text file (lib/termMenu.js) — and
// is applied once per token so asking twice never wipes a typed question.
export default function TermAttachBar({ term, seed, onClose }) {
  const imgPick = useRef(null);
  const filePick = useRef(null);
  const inputRef = useRef(null);
  const seeded = useRef("");
  const [items, setItems] = useState([]);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [pick, setPick] = useState(false);

  useEffect(() => { if (inputRef.current) inputRef.current.focus(); }, []);

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
      <WorkspaceAttach open={pick} termId={term.id} onPick={addWorkspace} onClose={() => setPick(false)} />
      <div className="term-attach-head">
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
        <button type="button" className="ws-icon-btn" title="Close (Esc)" aria-label="Close the message bar" onClick={onClose}><IconX size={13} /></button>
      </div>
      <form className="term-attach-row" data-align-row noValidate onSubmit={(e) => { e.preventDefault(); send(); }}>
        <input ref={imgPick} type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/*" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
        <input ref={filePick} type="file" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
        <button type="button" className="icon-btn composer-attach" title="Attach image" aria-label="Attach image" onClick={() => imgPick.current && imgPick.current.click()}><IconImage /></button>
        <button type="button" className="icon-btn composer-attach" title="Attach file" aria-label="Attach file" onClick={() => filePick.current && filePick.current.click()}><IconFile /></button>
        <button type="button" className="icon-btn composer-attach" title="Attach from folder" aria-label="Attach from folder" onClick={() => setPick(true)}><IconClip /></button>
        <input
          ref={inputRef}
          className="term-attach-input"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Message the terminal"
          aria-label="Message the terminal"
          autoComplete="off"
        />
        <button type="submit" className="icon-btn icon-btn-send" title="Send" disabled={busy || (!text.trim() && !items.length)}><IconSend size={16} /></button>
      </form>
    </div>
  );
}

function extOf(name) {
  const m = /\.([a-z0-9]+)$/i.exec(name || "");
  return (m ? m[1] : "file").slice(0, 4).toUpperCase();
}
