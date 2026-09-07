import { useEffect, useRef, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import KeyBar from "../components/KeyBar.jsx";
import TermSurface from "../components/TermSurface.jsx";
import { terms } from "../lib/terms.js";
import { scheduleTermFit } from "@picode/shared/domain/termFit.js";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { termLine } from "@picode/shared/domain/repoLine.js";
import { IconKeyboard, IconGit, IconClip, IconMore, IconTrash } from "../components/Icons.jsx";
import { useTermAccessory } from "../hooks/useTermAccessory.js";
import TermAttachSheet from "../components/TermAttachSheet.jsx";
import * as Dialog from "../components/MobileSheet.jsx";
import "../styles/mobile-tools.css";

// The pushed terminal screen (#/term/<id>): the same xterm the desktop
// attaches to the tmux session. Extra keys are an IME accessory — they
// open and close with the phone keyboard (ADR-0044). Attach does not
// focus xterm; a tap on the pane or the header icon does.

export default function TerminalScreen({ term, onBack, onRemove, busy, onOpenChanges }) {
  const [page, setPage] = useState(null);
  const [error, setError] = useState("");
  const [attach, setAttach] = useState(false);
  const [actions, setActions] = useState(false);
  const [retry, setRetry] = useState(0);
  const hostRef = useRef(null);
  const id = term && term.id;
  const entryOf = () => terms.get("sh:" + id);
  const attached = !!(page && !error && id);
  const keys = useTermAccessory(hostRef, entryOf, attached ? id : "");

  useEffect(() => {
    const host = hostRef.current;
    if (!host || !page || error) return undefined;
    let lastY = null;
    let acc = 0;
    const STEP = 16;
    const target = () => host.querySelector(".xterm-screen") || host.querySelector(".xterm");
    const onStart = (e) => { lastY = e.touches.length === 1 ? e.touches[0].clientY : null; acc = 0; };
    const onMove = (e) => {
      if (lastY == null || e.touches.length !== 1) return;
      const y = e.touches[0].clientY;
      acc += lastY - y;
      lastY = y;
      const el = target();
      if (!el) return;
      while (Math.abs(acc) >= STEP) {
        const dir = acc > 0 ? 1 : -1;
        acc -= dir * STEP;
        el.dispatchEvent(new WheelEvent("wheel", { deltaY: dir * 40, deltaMode: 0, bubbles: true, cancelable: true }));
      }
      e.preventDefault();
    };
    const onEnd = () => { lastY = null; acc = 0; };
    host.addEventListener("touchstart", onStart, { passive: true });
    host.addEventListener("touchmove", onMove, { passive: false });
    host.addEventListener("touchend", onEnd);
    host.addEventListener("touchcancel", onEnd);
    return () => {
      host.removeEventListener("touchstart", onStart);
      host.removeEventListener("touchmove", onMove);
      host.removeEventListener("touchend", onEnd);
      host.removeEventListener("touchcancel", onEnd);
    };
  }, [page, error, id]);

  useEffect(() => {
    setPage(null);
    setError("");
    if (!id) return undefined;
    let stale = false;
    api("/api/terminals/" + encodeURIComponent(id) + "/open", { method: "POST" })
      .then((p) => { if (!stale) setPage(p); })
      .catch((e) => { if (!stale) setError(humanizeError((e && e.message) || String(e))); });
    return () => { stale = true; };
  }, [id, retry]);

  useEffect(() => {
    const e = entryOf();
    if (e) scheduleTermFit(e, true);
  }, [keys.visible, id]);

  if (!term) {
    return (
      <div className="m-screen">
        <ScreenHeader title="Terminal" onBack={onBack} />
        <div className="m-tool-state"><p>That terminal is gone.</p><button type="button" className="btn btn-sm" onClick={onBack}>Back to Work</button></div>
      </div>
    );
  }
  const live = page ? { ...term, ...page, ...(term.launchCli ? term : {}) } : term;
  const line = termLine(live);
  const canKeys = !!(page && !error && !(live.launchCli && !live.running));
  return (
    <div className="m-screen m-term-screen">
      <ScreenHeader
        title={live.name || "Terminal"}
        sub={line.text}
        onBack={onBack}
        right={(
          <div className="m-tool-toolbar" data-align-row>
            {live.launchCli && live.running ? (
              <button type="button" className="m-tool-icon" title="Attach" aria-label="Attach" onClick={() => setAttach(true)}>
                <IconClip size={16} />
              </button>
            ) : null}
            {canKeys ? (
              <button type="button" className={"m-tool-icon m-keys-btn" + (keys.visible ? " on" : "")} title={keys.visible ? "Hide keyboard" : "Show keyboard"} aria-label={keys.visible ? "Hide keyboard" : "Show keyboard"} aria-pressed={keys.visible} onPointerDown={(e) => { if (keys.visible) e.preventDefault(); }} onClick={() => { keys.visible ? keys.hide() : keys.show(); }}>
                <IconKeyboard size={16} />
              </button>
            ) : null}
            <button type="button" className="m-tool-icon" title="Terminal actions" aria-label="Terminal actions" aria-haspopup="dialog" aria-expanded={actions} onClick={() => setActions(true)}><IconMore size={18} /></button>
          </div>
        )}
      />
      <div className="m-term" ref={hostRef}>
        {error ? (
          <div className="m-tool-state" role="alert"><p>{error}</p><button type="button" className="btn btn-sm" onClick={() => setRetry((n) => n + 1)}>Retry</button></div>
        ) : page ? (
          <TermSurface term={live} hidden={false} cwdKind="term" />
        ) : (
          <div className="m-tool-state m-tool-loading" role="status" aria-label="Attaching terminal" aria-busy="true"><p>Attaching…</p><span className="gg-skel" /><span className="gg-skel" /><span className="gg-skel" /></div>
        )}
      </div>
      {canKeys && keys.visible ? <KeyBar armed={keys.armed} onArm={keys.armKey} onKey={keys.sendKey} onHide={keys.hide} /> : null}
      {live.launchCli && live.running ? <TermAttachSheet term={live} open={attach} onClose={() => setAttach(false)} /> : null}
      <Dialog.Root open={actions} onOpenChange={setActions}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg m-term-actions" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">Terminal actions</Dialog.Title>
            <Dialog.Description className="m-term-actions-name">{live.name || "Terminal"}</Dialog.Description>
            {live.git && onOpenChanges ? <button type="button" className="m-tool-action" onClick={() => { setActions(false); onOpenChanges("term", term.id, live.name || "Terminal"); }}><IconGit size={18} /><span>View changes</span>{live.git.dirty ? <span className="m-tool-action-count">{live.git.dirty}</span> : null}</button> : null}
            <button type="button" className="m-tool-action is-danger" disabled={busy} onClick={() => { setActions(false); onRemove(term); }}><IconTrash size={18} /><span>{busy ? "Removing…" : "Remove terminal"}</span></button>
            <Dialog.Close asChild><button type="button" className="btn btn-sm">Done</button></Dialog.Close>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  );
}
