import { lazy, Suspense, useEffect, useRef, useState } from "react";
import WorkspaceAttach from "./WorkspaceAttach.jsx";
import { IconClip, IconFile, IconImage, IconSend, IconSketch, IconX } from "./Icons.jsx";
import { planAttachFiles, readAttachFile, MAX_ATTACH } from "@picode/shared/domain/termPrompt.js";
import { fitAttachField, isAttachSendKey } from "@picode/shared/domain/attachText.js";
import { sceneHasInk } from "@picode/shared/domain/composerImage.js";
import { toast } from "../lib/toast.js";

// Excalidraw loads with the sketch, never with the composer.
const SketchEditor = lazy(() => import("./SketchEditor.jsx"));

const newId = () => (crypto.randomUUID && crypto.randomUUID()) || String(Date.now()) + Math.random().toString(16).slice(2);

// The attach composer: a message plus up to four photos, files, folder
// picks or sketches. The parent owns `text` and `items` and decides what
// Send means — the terminal bar pastes them into a running CLI
// (TermAttachBar), Fork agent… starts a new agent on them
// (ForkAgentDialog). Staged items carry `data` (read here, uploaded by the
// parent) or `path` (a file already in the folder).
//
// `agentId` / `termId` scope the folder picker. `seed` (the terminal
// menu's selection) is applied once per token so asking twice never wipes
// a typed question. `onClose` adds the × and Escape; `sendLabel` names the
// Enter action; `hideSend` leaves the button to the parent's own actions.
export default function AttachComposer({
  className = "term-attach",
  text,
  setText,
  items,
  setItems,
  onSubmit,
  busy = false,
  error = "",
  onClose,
  seed,
  agentId,
  termId,
  placeholder = "Message the terminal",
  sendLabel = "Send",
  hideSend = false,
  autoFocus = true,
}) {
  const imgPick = useRef(null);
  const filePick = useRef(null);
  const inputRef = useRef(null);
  const seeded = useRef("");
  // Live count for planning: two rapid pastes overlap in flight, and the
  // `items` closure each carries goes stale — the ref never does.
  const itemsRef = useRef([]);
  itemsRef.current = items;
  const [pick, setPick] = useState(false);
  const [sketch, setSketch] = useState(null);

  useEffect(() => { if (autoFocus && inputRef.current) inputRef.current.focus(); }, []);

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
    const plan = planAttachFiles(list, itemsRef.current.length);
    if (plan.tooMany) toast.error("Up to 4 files.");
    if (plan.tooLarge) toast.error("Each file must be under 4 MB.");
    if (!plan.files.length) return;
    // One functional append per file: concurrent addFiles calls each see
    // the latest list, so a double-paste stages both instead of last-win.
    // The cap is re-checked inside, so overlap can never stage a fifth.
    for (const f of plan.files) {
      try {
        const row = await readAttachFile(f);
        const id = newId();
        setItems((cur) => (cur.length >= MAX_ATTACH ? cur : cur.concat([{ id, ...row }])));
      } catch (err) {
        if (err && err.message === "too-large") toast.error("Each file must be under 4 MB.");
      }
    }
  }

  function addWorkspace(hit) {
    setPick(false);
    if (!hit || !hit.path) return;
    if (items.length >= MAX_ATTACH) { toast.error("Up to 4 files."); return; }
    setItems((cur) => cur.concat([{
      id: newId(),
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
        return cur.concat([{ ...row, id: newId(), scene }]);
      });
      setSketch(null);
    } catch (err) {
      if (err && err.message === "too-large") toast.error("Each file must be under 4 MB.");
    }
  }

  const canSend = !busy && (!!text.trim() || items.length > 0);
  function submit() {
    if (canSend && onSubmit) onSubmit();
  }

  // Escape closes the composer. The folder picker owns the key while it is
  // open, so it closes first.
  function onKeyDown(e) {
    if (e.key !== "Escape" || pick || !onClose) return;
    e.stopPropagation();
    onClose();
  }

  return (
    <div className={className} onKeyDown={onKeyDown}>
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
        {onClose ? <button type="button" className="ws-icon-btn" title="Close (Esc)" aria-label="Close the message bar" onClick={onClose}><IconX size={13} /></button> : null}
      </div>
      {/* Not a [data-align-row]: the field grows by design (overlayAudit's
          equal-height rule is for fixed controls). The row is bottom-aligned. */}
      {error ? <p className="term-attach-error" role="alert">{error}</p> : null}
      <div className="term-attach-row">
        <input ref={imgPick} type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/*" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
        <input ref={filePick} type="file" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
        <button type="button" className="icon-btn composer-attach" title="Attach image" aria-label="Attach image" disabled={busy} onClick={() => imgPick.current && imgPick.current.click()}><IconImage /></button>
        <button type="button" className="icon-btn composer-attach" title="Attach file" aria-label="Attach file" disabled={busy} onClick={() => filePick.current && filePick.current.click()}><IconFile /></button>
        <button type="button" className="icon-btn composer-attach" title="Attach from folder" aria-label="Attach from folder" disabled={busy} onClick={() => setPick(true)}><IconClip /></button>
        <button type="button" className="icon-btn composer-attach" title="Sketch" aria-label="Sketch" disabled={busy} onClick={() => openSketch()}><IconSketch /></button>
        <textarea
          ref={inputRef}
          className="term-attach-input"
          rows={1}
          value={text}
          readOnly={busy}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => { if (isAttachSendKey(e)) { e.preventDefault(); submit(); } }}
          placeholder={placeholder}
          aria-label={placeholder}
          autoComplete="off"
        />
        {hideSend ? null : (
          <button type="button" className="icon-btn icon-btn-send" title={sendLabel + " (Enter)"} aria-label={sendLabel} disabled={!canSend} onClick={submit}><IconSend size={16} /></button>
        )}
      </div>
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
