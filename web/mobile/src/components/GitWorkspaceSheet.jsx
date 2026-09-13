import * as Sheet from "./MobileSheet.jsx";
import { IconCheck, IconGit } from "./Icons.jsx";
import { pickerOptions } from "@picode/shared/domain/workspacePicker.js";

// Which workspace's folder this Git screen reads through (ADR-0022, ADR-0095).
// The phone has no tab strip to rename, so a pick is a navigation: the host
// opens the Git screen for that workspace and Back returns to this one.
//
// Only folders that are repositories are listed — the same rule the desktop
// picker and the sidebar use, so no row can answer "not a Git repository".
export default function GitWorkspaceSheet({ open, workspaces = [], currentId = "", onPick, onClose }) {
  const options = pickerOptions(workspaces, { reposOnly: true });
  return (
    <Sheet.Root open={!!open} onOpenChange={value => { if (!value) onClose?.(); }}>
      <Sheet.Portal>
        <Sheet.Overlay className="dlg-overlay" />
        <Sheet.Content className="dlg m-git-sheet" aria-describedby="m-git-ws-description">
          <div className="m-git-sheet-heading">
            <Sheet.Title className="dlg-title">Workspace</Sheet.Title>
            <button type="button" className="btn btn-ghost" onClick={onClose}>Close</button>
          </div>
          <Sheet.Description id="m-git-ws-description" className="dlg-body">
            The folder this history is read through.
          </Sheet.Description>
          {options.length ? (
            <div className="m-git-action-list">
              {options.map(o => (
                <button
                  type="button"
                  key={o.id}
                  className="m-git-action m-git-ws-row"
                  aria-current={o.id === currentId ? "true" : undefined}
                  onClick={() => { if (o.id === currentId) onClose?.(); else onPick?.(o.id); }}
                >
                  <span className="m-git-ws-name">{o.label}</span>
                  {o.hint ? <span className="m-git-ws-hint">{o.hint}</span> : null}
                  <span className="m-git-ws-check" aria-hidden="true">{o.id === currentId ? <IconCheck size={15} /> : null}</span>
                </button>
              ))}
            </div>
          ) : (
            <div className="m-git-state">
              <p>No workspace here is a Git repository.</p>
              <button type="button" className="btn" onClick={onClose}>Close</button>
            </div>
          )}
          <p className="m-git-ws-note"><IconGit size={12} /> Git reads through the folder it was picked from.</p>
        </Sheet.Content>
      </Sheet.Portal>
    </Sheet.Root>
  );
}
