import { memo } from "react";
import FilePane from "../FilePane.jsx";

// FilePanel — the loaded body of a `file` panel (docs/plans/matrix-canvas.md
// §4.2): the editor the file tab and the tree already use, in its embedded
// layout — no width sizer, no expand, no Close, because the panel is the
// frame now. Reading, editing, saving, the preview toggle and the read
// failures are all FilePane's, unchanged.
//
// Two things are the canvas's:
//   * `docKey` — the open document lives in lib/fileDocs.js keyed by this
//     panel's ref, so maximizing the panel or switching layout mode moves
//     the body between hosts without throwing away unsaved text.
//   * `onDirty` — the surface pins a dirty editor (chunkLoader `keep`), so
//     panning away never unmounts work in progress.
//
// A file that is not on disk is *not* a gone binding: FilePane says what the
// read answered and offers Reload, and the panel keeps its own actions.
const FilePanel = memo(function FilePanel({ owner, path, refKey, hidden, onDirty }) {
  return (
    <FilePane
      agentId={owner.kind === "agent" ? owner.id : ""}
      termId={owner.kind === "term" ? owner.id : ""}
      wsId={owner.kind === "workspace" ? owner.id : ""}
      path={path}
      variant="embedded"
      hidden={hidden}
      docKey={refKey}
      onDirty={onDirty}
    />
  );
});

export default FilePanel;
