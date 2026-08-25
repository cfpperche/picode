import * as Dialog from "@radix-ui/react-dialog";
import { shortPath } from "../lib/repoLine.js";
import { shortModel } from "../lib/chip.js";
import { toast } from "../lib/toast.js";

function row(k, v) {
  if (v == null || v === "") return null;
  return (
    <div className="session-fact">
      <dt>{k}</dt>
      <dd>{v}</dd>
    </div>
  );
}

export default function SessionInfo({ open, onClose, bar, agent, onRename, onNew, onCompact, onTree }) {
  const file = agent && agent.sessionPath;
  const model = [agent && agent.provider, shortModel((agent && agent.model) || ""), agent && agent.thinking].filter(Boolean).join(" · ");
  let git = "";
  if (bar && bar.branch) {
    git = bar.branch;
    if (bar.worktree) git += " / " + bar.worktree;
    if (bar.dirty) git += "*";
  }
  let tokens = "";
  if (bar && (bar.input || bar.output)) tokens = "↑" + (bar.input || 0) + "  ↓" + (bar.output || 0);
  let ctx = "";
  if (bar && bar.contextWindow) {
    ctx = bar.contextPercent == null ? String(bar.contextWindow) : bar.contextPercent.toFixed(1) + "% of " + bar.contextWindow;
  }
  const cost = bar && bar.cost > 0 ? "$" + bar.cost.toFixed(4) : "";

  function copyPath() {
    if (!file) { toast.info("No session file yet."); return; }
    navigator.clipboard.writeText(file).then(() => toast.ok("Path copied.")).catch(() => toast.error("Clipboard blocked."));
  }

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-session" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Session</Dialog.Title>
          <Dialog.Description className="dlg-body">Current JSONL and usage for this agent.</Dialog.Description>
          <dl className="session-facts">
            {row("Name", (bar && bar.sessionName) || "—")}
            <div className="session-fact">
              <dt>File</dt>
              <dd title={file || ""}>{file ? shortPath(file) : "—"}</dd>
            </div>
            {row("Folder", bar && bar.cwd)}
            {row("Git", git)}
            {row("Model", model)}
            {row("Tokens", tokens)}
            {row("Context", ctx)}
            {row("Cost", cost)}
          </dl>
          <div className="dlg-actions session-info-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={copyPath} disabled={!file}>Copy path</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => { onClose(); onRename && onRename(); }}>Rename</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => { onClose(); onNew && onNew(); }}>New</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => { onClose(); onCompact && onCompact(); }}>Compact</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => { onClose(); onTree && onTree(); }}>Tree</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
