import { useState } from "react";
import { useDebounced } from "../lib/useDebounced.js";
import { filterRows } from "../lib/inspector.js";
import FileTree from "./FileTree.jsx";

// The project tree (Orca's Files shape), lazy one level at a time like the
// folder tab, with a filter over what is already loaded — labeled as such,
// because a complete search is a server matter this rail does not have yet.
export default function InspectorFiles({ rows, kinds, dirtyDirs, activePath, onToggle, onOpen, onRefresh }) {
  const [text, setText] = useState("");
  const query = useDebounced(text, 150);
  const shown = filterRows(rows, query);
  return (
    <>
      <div className="insp-sub">
        <input
          type="search"
          className="insp-filter"
          placeholder="Filter loaded files"
          aria-label="Filter loaded files"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Escape" && text) { e.preventDefault(); setText(""); } }}
        />
      </div>
      {rows.length === 0 ? (
        <p className="insp-msg">Empty folder. <button type="button" className="btn btn-sm" onClick={onRefresh}>Refresh</button></p>
      ) : shown.length === 0 ? (
        <p className="insp-msg">No loaded file matches. <button type="button" className="btn btn-sm" onClick={() => setText("")}>Clear</button></p>
      ) : (
        <FileTree rows={shown} kinds={kinds} dirtyDirs={dirtyDirs} selectedPath={activePath} onToggle={onToggle} onOpen={onOpen} onRefresh={onRefresh} />
      )}
    </>
  );
}
