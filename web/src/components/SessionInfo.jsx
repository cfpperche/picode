import * as Dialog from "@radix-ui/react-dialog";

function row(k, v) {
  if (v == null || v === "") return null;
  return (
    <div className="session-fact">
      <dt>{k}</dt>
      <dd>{v}</dd>
    </div>
  );
}

export default function SessionInfo({ open, onClose, bar, agent }) {
  const file = agent && agent.sessionPath;
  const cwd = bar && bar.cwd;
  const name = (bar && bar.sessionName) || "";
  const cost = bar && bar.cost > 0 ? "$" + bar.cost.toFixed(2) : "";
  let tokens = "";
  if (bar && (bar.input || bar.output)) {
    tokens = "↑" + (bar.input || 0) + " ↓" + (bar.output || 0);
  }
  let ctx = "";
  if (bar && bar.contextWindow) {
    ctx = bar.contextPercent == null ? "? / " + bar.contextWindow : bar.contextPercent.toFixed(1) + "% · " + bar.contextWindow;
  }
  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Session</Dialog.Title>
          <Dialog.Description className="dlg-body">This agent's current session file and usage.</Dialog.Description>
          <dl className="session-facts">
            {row("File", file || "—")}
            {row("Name", name)}
            {row("Folder", cwd)}
            {row("Tokens", tokens)}
            {row("Context", ctx)}
            {row("Cost", cost)}
          </dl>
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
