import { useEffect, useRef, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import KeyBar from "../components/KeyBar.jsx";
import TermSurface from "../components/TermSurface.jsx";
import { terms } from "../lib/terms.js";
import { scheduleTermFit } from "@picode/shared/domain/termFit.js";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { termLine } from "@picode/shared/domain/repoLine.js";
import { IconKeyboard, IconGit, IconClip } from "../components/Icons.jsx";
import { useTermAccessory } from "../hooks/useTermAccessory.js";
import TermAttachSheet from "../components/TermAttachSheet.jsx";

// The pushed terminal screen (#/term/<id>): the same xterm the desktop
// attaches to the tmux session. Extra keys are an IME accessory — they
// open and close with the phone keyboard (ADR-0044).

export default function TerminalScreen({ term, onBack, onRemove, busy, onOpenChanges }) {
  const [page, setPage] = useState(null);
  const [error, setError] = useState("");
  const [attach, setAttach] = useState(false);
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
  }, [id]);

  useEffect(() => {
    const e = entryOf();
    if (e) scheduleTermFit(e, true);
  }, [keys.visible, id]);

  if (!term) {
    return (
      <div className="m-screen">
        <ScreenHeader title="Terminal" onBack={onBack} />
        <p className="m-empty-line m-pad">That terminal is gone.</p>
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
          <>
            {live.git && live.git.dirty ? (
              <button type="button" className="btn btn-sm m-changes-btn" title="Uncommitted changes" onClick={() => onOpenChanges("term", term.id, live.name || "Terminal")}><IconGit size={13} /> {live.git.dirty}</button>
            ) : null}
            {live.launchCli && live.running ? (
              <button type="button" className="btn btn-sm" title="Attach" aria-label="Attach" onClick={() => setAttach(true)}>
                <IconClip size={13} />
              </button>
            ) : null}
            {canKeys ? (
              <button type="button" className={"btn btn-sm m-keys-btn" + (keys.visible ? " on" : "")} title={keys.visible ? "Hide keyboard" : "Show keyboard"} aria-label={keys.visible ? "Hide keyboard" : "Show keyboard"} aria-pressed={keys.visible} onPointerDown={(e) => e.preventDefault()} onClick={() => { keys.visible ? keys.hide() : keys.show(); }}>
                <IconKeyboard size={16} />
              </button>
            ) : null}
            <button type="button" className="btn btn-sm" disabled={busy} onClick={() => onRemove(term)}>Remove</button>
          </>
        )}
      />
      <div className="m-term" ref={hostRef}>
        {error ? (
          <p className="m-empty-line m-pad">{error}</p>
        ) : page ? (
          <TermSurface term={live} hidden={false} cwdKind="term" />
        ) : (
          <p className="m-empty-line m-pad">Attaching…</p>
        )}
      </div>
      {canKeys && keys.visible ? <KeyBar armed={keys.armed} onArm={keys.armKey} onKey={keys.sendKey} onHide={keys.hide} /> : null}
      {live.launchCli && live.running ? <TermAttachSheet term={live} open={attach} onClose={() => setAttach(false)} /> : null}
    </div>
  );
}
