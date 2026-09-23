import { useEffect, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { lineDiff } from "@picode/shared/domain/instructions.js";
import { toast } from "../lib/toast.js";

// Review a proposed instruction-file fix (ADR-0204): the exact change per
// file, written only on "Write change". The server refuses a write when a
// file moved since this diff was read (409 carries the fresh change, shown
// in place), and it never runs git: the edit waits in the Git tab.
export default function InstructionsFix({ workspaceId, fixId, onClose, onWritten }) {
  const [state, setState] = useState({ fix: null, error: "", busy: true, note: "" });

  useEffect(() => {
    if (!fixId) return;
    let live = true;
    setState({ fix: null, error: "", busy: true, note: "" });
    api("/api/workspaces/" + encodeURIComponent(workspaceId) + "/instructions/fix?id=" + encodeURIComponent(fixId))
      .then((fix) => { if (live) setState({ fix, error: "", busy: false, note: "" }); })
      .catch((e) => { if (live) setState({ fix: null, error: e.message || "failed", busy: false, note: "" }); });
    return () => { live = false; };
  }, [workspaceId, fixId]);

  async function write() {
    const fix = state.fix;
    if (!fix) return;
    setState((s) => ({ ...s, busy: true, note: "" }));
    const hashes = {};
    for (const c of fix.changes) hashes[c.path] = c.hash;
    try {
      await api("/api/workspaces/" + encodeURIComponent(workspaceId) + "/instructions/fix", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: fix.id, hashes }),
      });
    } catch (e) {
      if (e.status === 409 && e.body && e.body.fix) {
        setState({ fix: e.body.fix, error: "", busy: false, note: "These files changed since. This is the change now." });
        return;
      }
      setState((s) => ({ ...s, busy: false, error: e.message || "failed" }));
      return;
    }
    onWritten && onWritten();
    toast.ok("Written. The change waits in the Git tab.");
  }

  const fix = state.fix;
  return (
    <Dialog.Root open={!!fixId} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg instr-fix-dlg">
          <Dialog.Title className="dlg-title">{fix ? fix.title : "Review change"}</Dialog.Title>
          <Dialog.Description className="dlg-body">Nothing is written until you choose Write change. Nothing is committed.</Dialog.Description>
          {state.busy && !fix ? <div className="skel-line" style={{ height: 96, width: "100%" }} aria-label="Reading the files" /> : null}
          {state.note ? <p className="instr-fix-note" role="status">{state.note}</p> : null}
          {fix ? fix.changes.map((c) => (
            <div key={c.path} className="instr-fix-file">
              <div className="instr-fix-path">{c.path}{c.exists ? null : <span className="instr-tag">new file</span>}</div>
              <pre className="instr-diff">
                {c.after === "" && !c.exists ? <span className="d-ctx">(empty file)</span> : lineDiff(c.before, c.after).map((l, i) => (
                  <span key={i} className={l.kind === "+" ? "d-add" : l.kind === "-" ? "d-del" : "d-ctx"}>{(l.kind === "…" ? "  " : l.kind + " ") + l.text + "\n"}</span>
                ))}
              </pre>
            </div>
          )) : null}
          {state.error ? <p className="form-error" role="alert">{state.error}</p> : null}
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={state.busy && !!fix}>Cancel</button>
            {fix ? <button type="button" className="btn btn-primary btn-sm" onClick={write} disabled={state.busy}>{state.busy ? "Writing…" : "Write change"}</button> : null}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
