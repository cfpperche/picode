import * as Dialog from "@radix-ui/react-dialog";

function flatten(nodes, depth, out) {
  for (const n of nodes || []) {
    out.push({ ...n, depth });
    flatten(n.children, depth + 1, out);
  }
  return out;
}

export default function SessionTree({ open, onClose, mode, tree, onFork, onClone }) {
  const rows = flatten((tree && tree.tree) || [], 0, []);
  const leaf = tree && tree.leafId;
  const forkOnly = mode === "fork";
  const shown = forkOnly ? rows.filter((r) => r.role === "user") : rows;
  const title = forkOnly ? "Fork from a prompt" : "Session tree";

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-tree" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            {forkOnly ? "New session from that user message." : "Same file. Fork creates a new session from a prompt. Clone copies this branch."}
          </Dialog.Description>
          <div className="tree-list">
            {shown.length === 0 ? (
              <p className="side-empty">No messages yet</p>
            ) : (
              shown.map((r) => {
                const user = r.role === "user";
                const label = r.text || r.kind || r.id;
                return (
                  <button
                    key={r.id}
                    type="button"
                    className={"tree-row" + (r.id === leaf ? " leaf" : "") + (user ? " user" : "")}
                    style={{ paddingLeft: 8 + r.depth * 14 }}
                    disabled={!user}
                    onClick={() => user && onFork(r.id)}
                    title={user ? "Fork from here" : undefined}
                  >
                    <span className="tree-kind">{user ? "You" : (r.role || r.kind)}</span>
                    <span className="tree-text">{label}</span>
                  </button>
                );
              })
            )}
          </div>
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Close</button>
            {!forkOnly ? (
              <button type="button" className="btn btn-primary btn-sm" onClick={onClone} disabled={shown.length === 0}>Clone</button>
            ) : null}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
