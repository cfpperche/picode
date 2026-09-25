import FilePane from "./FilePane.jsx";
import WorkingDiff from "./WorkingDiff.jsx";

// One file tab, two views. Editor is FilePane as before; Diff renders the
// working-tree patch for the same path (the Inspector's Changes opens tabs
// here), and the two swap in place — "View diff" and "Open file" are the pair
// ADR-0074 already uses, so no new chrome is invented. The view is per-tab
// viewer state held by the app, not part of the tab id or the hash.
export default function FileSurface({ owner, path, error, onClose, view = "file", onView, changed = false, root = "", worktree = "", onOpenPath }) {
  if (error) {
    return (
      <section className="file-surface" aria-label="File">
        <p className="file-pane-msg">
          {error}{" "}
          <a href="#/">Back</a>
        </p>
      </section>
    );
  }
  if (!owner || !path) return null;
  if (view === "diff") {
    return (
      <section className="file-surface file-surface-diff" aria-label={`Changes to ${path}`}>
        <WorkingDiff owner={owner} path={path} root={root} worktree={worktree} onOpenFile={onView ? () => onView("file") : undefined} />
      </section>
    );
  }
  return (
    <section className="file-surface" aria-label={path}>
      <FilePane
        agentId={owner.kind === "agent" ? owner.id : ""}
        termId={owner.kind === "term" ? owner.id : ""}
        wsId={owner.kind === "workspace" ? owner.id : ""}
        path={path}
        onClose={onClose}
        variant="tab"
        root={root}
        worktree={worktree}
        onViewDiff={changed && onView ? () => onView("diff") : undefined}
        onOpenPath={onOpenPath}
      />
    </section>
  );
}
