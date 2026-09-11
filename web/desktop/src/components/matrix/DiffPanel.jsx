import { memo, useEffect, useRef, useState } from "react";
import { gitTouches } from "@picode/shared/domain/canvas.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import WorkingDiff from "../WorkingDiff.jsx";

// DiffPanel — the loaded body of a `diff` panel (docs/plans/matrix-canvas.md
// §4.2): the working-tree patch for one path, the same `WorkingDiff` the
// file tab's Diff view and the Inspector's Changes rows show. Its own header
// carries **Open file**, so the pair ADR-0074 already uses stays the pair
// here; the panel's header opens the diff in a tab.
//
// It follows `git.updated` for the folder its owner works in — the fleet's
// watcher names the folder it inspected — exactly as the Inspector does, and
// nothing polls. `nonce` is what re-reads the patch: `WorkingDiff` owns the
// request.
const DiffPanel = memo(function DiffPanel({ owner, path, watch, onOpenFile }) {
  const [nonce, setNonce] = useState(0);
  const watchRef = useRef(watch);
  watchRef.current = watch;
  useEffect(() => subscribeFeed((ev) => {
    const hit = ev.type === "git.updated" && ev.data && gitTouches(watchRef.current, ev.data.path);
    if (hit || ev.type === "feed.open" || ev.type === "feed.reset") setNonce((n) => n + 1);
  }), []);
  return <WorkingDiff owner={owner} path={path} nonce={nonce} onOpenFile={onOpenFile} />;
});

export default DiffPanel;
