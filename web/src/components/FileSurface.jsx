import FilePane from "./FilePane.jsx";

export default function FileSurface({ owner, path, error, onClose }) {
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
  return (
    <section className="file-surface" aria-label={path}>
      <FilePane
        agentId={owner.kind === "agent" ? owner.id : ""}
        termId={owner.kind === "term" ? owner.id : ""}
        wsId={owner.kind === "workspace" ? owner.id : ""}
        path={path}
        onClose={onClose}
        variant="tab"
      />
    </section>
  );
}
