import { adoptOffer } from "@picode/shared/domain/cliLaunch.js";
import { useEffect, useRef, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import KeyBar from "../components/KeyBar.jsx";
import TermSurface from "../components/TermSurface.jsx";
import { terms } from "../lib/terms.js";
import { scheduleTermFit } from "@picode/shared/domain/termFit.js";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";
import { termLine } from "@picode/shared/domain/repoLine.js";
import { IconKeyboard, IconGit, IconFolder, IconClip, IconMore, IconPanelRight, IconTrash, IconFile, IconStop } from "../components/Icons.jsx";
import { useTermAccessory } from "../hooks/useTermAccessory.js";
import TermAttachSheet from "../components/TermAttachSheet.jsx";
import SnipRunSheet from "../components/SnipRunSheet.jsx";
import * as Dialog from "../components/MobileSheet.jsx";
import "../styles/mobile-tools.css";

// The pushed terminal screen (#/term/<id>): the same xterm the desktop
// attaches to the tmux session. Extra keys are an IME accessory — they
// open and close with the phone keyboard (ADR-0044). Attach does not
// focus xterm; a tap on the pane or the header icon does.

export default function TerminalScreen({ term, onBack, onRemove, busy, onOpenFiles, onOpenGit, onOpenInspector, owner, title, onStop }) {
  const [page, setPage] = useState(null);
  const [error, setError] = useState("");
  const [attach, setAttach] = useState(false);
  const [actions, setActions] = useState(false);
  const [snipRun, setSnipRun] = useState(null); // { onlyKind }
  const [retry, setRetry] = useState(0);
  const hostRef = useRef(null);
  const id = term && term.id;
  const targetOwner = owner || { kind: "term", id };
  const legacy = targetOwner.kind === "agent";
  const entryOf = () => terms.get("sh:" + id);
  const attached = !!(page && !error && id);
  const keys = useTermAccessory(hostRef, entryOf, attached ? id : "", () => {
    toast.warn("Terminal reconnecting — tap the pane, then try again.");
  });


  useEffect(() => {
    setPage(null);
    setError("");
    if (!id) return undefined;
    if (legacy) { setPage(term); return undefined; }
    let stale = false;
    api("/api/terminals/" + encodeURIComponent(id) + "/open", { method: "POST" })
      .then((p) => { if (!stale) setPage(p); })
      .catch((e) => { if (!stale) setError(humanizeError((e && e.message) || String(e))); });
    return () => { stale = true; };
  }, [id, legacy, retry]);

  useEffect(() => {
    const e = entryOf();
    if (e) scheduleTermFit(e, true);
  }, [keys.visible, id]);

  if (!term) {
    return (
      <div className="m-screen">
        <ScreenHeader title={title || "Terminal"} onBack={onBack} right={<>{onOpenInspector ? <button type="button" className="btn btn-sm" aria-label="Inspector" title="Inspector" aria-haspopup="dialog" onClick={() => onOpenInspector?.()}><IconPanelRight size={16} /></button> : null}{onStop ? <button type="button" className="btn btn-sm" disabled={busy} onClick={onStop}>Stop agent</button> : null}</>} />
        <div className="m-tool-state"><p>Terminal unavailable.</p><button type="button" className="btn btn-sm" onClick={() => location.reload()}>Reload</button></div>
      </div>
    );
  }
  const live = page ? { ...term, ...page, ...(term.launchCli ? term : {}) } : term;
  const line = termLine(live);
  const canKeys = !!(page && !error && !(live.launchCli && !live.running));
  return (
    <div className="m-screen m-term-screen">
      <ScreenHeader
        title={title || live.name || "Terminal"}
        sub={line.text}
        onBack={onBack}
        right={(
          <div className="m-tool-toolbar" data-align-row>
            {onOpenInspector ? (
              <button type="button" className="m-tool-icon" title="Inspector" aria-label="Inspector" aria-haspopup="dialog" onClick={() => onOpenInspector?.()}>
                <IconPanelRight size={16} />
              </button>
            ) : null}
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
          <TermSurface adopt={adoptOffer(live, targetOwner.kind === "agent" ? targetOwner : null)} term={live} hidden={false} cwdKind={targetOwner.kind} onOpenFile={(path) => onOpenFiles(targetOwner, { path })} />
        ) : (
          <div className="m-tool-state m-tool-loading" role="status" aria-label="Attaching terminal" aria-busy="true"><p>Attaching…</p><span className="gg-skel" /><span className="gg-skel" /><span className="gg-skel" /></div>
        )}
      </div>
      {canKeys && keys.visible ? <KeyBar armed={keys.armed} onArm={keys.armKey} onKey={keys.sendKey} onHide={keys.hide} /> : null}
      {live.launchCli && live.running ? <TermAttachSheet term={live} owner={targetOwner} open={attach} onClose={() => setAttach(false)} /> : null}
      <SnipRunSheet
        open={!!snipRun}
        mode="run"
        target={{ type: legacy ? "agent" : "terminal", id }}
        targetName={snipRun && snipRun.targetName}
        onlyKind={snipRun && snipRun.onlyKind}
        onRan={() => setSnipRun(null)}
        onClose={() => setSnipRun(null)}
      />
      <Dialog.Root open={actions} onOpenChange={setActions}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg m-term-actions" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">Terminal actions</Dialog.Title>
            <Dialog.Description className="m-term-actions-name">{live.name || "Terminal"}</Dialog.Description>
            <button type="button" className="m-tool-action" onClick={() => { setActions(false); onOpenFiles(targetOwner); }}><IconFolder size={18} /><span>Files</span></button>
            <button type="button" className="m-tool-action" onClick={() => { setActions(false); onOpenGit(targetOwner); }}><IconGit size={18} /><span>Git</span>{live.git?.dirty ? <span className="m-tool-action-count">{live.git.dirty}</span> : null}</button>
            {live.launchCli && live.running ? (
              <button type="button" className="m-tool-action" onClick={() => { setActions(false); setSnipRun({ onlyKind: "prompt", targetName: live.name }); }}><IconFile size={18} /><span>Send to terminal…</span></button>
            ) : !live.launchCli ? (
              <button type="button" className="m-tool-action" onClick={() => { setActions(false); setSnipRun({ onlyKind: "shell", targetName: live.name }); }}><IconFile size={18} /><span>Run command…</span></button>
            ) : null}
            {onStop ? <button type="button" className="m-tool-action is-danger" disabled={busy} onClick={() => { setActions(false); onStop(); }}><IconStop size={18} /><span>{busy ? "Stopping…" : "Stop agent"}</span></button> : null}
            {onRemove ? <button type="button" className="m-tool-action is-danger" disabled={busy} onClick={() => { setActions(false); onRemove(term); }}><IconTrash size={18} /><span>{busy ? "Removing…" : "Remove terminal"}</span></button> : null}
            <Dialog.Close asChild><button type="button" className="btn btn-sm">Done</button></Dialog.Close>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  );
}
