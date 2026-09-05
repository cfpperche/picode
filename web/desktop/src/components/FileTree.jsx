import { useRef, useState } from "react";
import { treeKeyAction } from "../lib/fileTree.js";
import { IconChevronRight, IconFile, IconFolder } from "./Icons.jsx";

// Rows come from lib/fileTree.flattenTree and decoration from gitstatus.
// Only keyboard focus is local; document selection belongs to the surface.
export default function FileTree({ rows, kinds, dirtyDirs, selectedPath, onToggle, onOpen, onRefresh }) {
  const [focused, setFocused] = useState("");
  const listRef = useRef(null);
  const active = rows.some((r) => r.path === focused) ? focused : rows.find((r) => r.path === selectedPath)?.path || rows[0]?.path;
  function onKey(event, index) {
    const action = treeKeyAction(rows, index, event.key);
    if (!action) return;
    event.preventDefault();
    if (action.toggle) onToggle(action.toggle);
    if (action.focus) listRef.current?.querySelector(`[data-path="${CSS.escape(action.focus)}"]`)?.focus();
  }
  return (
    <ul className="ft-list" role="tree" aria-label="Project files" ref={listRef}>
      {rows.map((row, index) => (
        <li key={row.path} role="none">
          <button
            type="button"
            role="treeitem"
            aria-level={row.depth + 1}
            aria-expanded={row.isDir ? row.open : undefined}
            aria-selected={selectedPath === row.path}
            tabIndex={active === row.path ? 0 : -1}
            data-path={row.path}
            className={"ft-row" + (row.ignored ? " ft-row-ignored" : "") + (row.isDir ? " ft-row-dir" : "") + (selectedPath === row.path ? " ft-row-on" : "")}
            style={{ paddingLeft: 8 + row.depth * 14 }}
            onFocus={() => setFocused(row.path)}
            onKeyDown={(e) => onKey(e, index)}
            onClick={() => (row.isDir ? onToggle(row.path) : onOpen(row.path))}
            title={row.path + (row.ignored ? " — Ignored by Git" : "")}
            aria-label={row.ignored ? row.name + ", ignored by Git" : undefined}
          >
            <span className={"ft-chev" + (row.isDir && row.open ? " ft-chev-open" : "")}>
              {row.isDir ? <IconChevronRight /> : null}
            </span>
            <span className="ft-icon">{row.isDir ? <IconFolder /> : <IconFile />}</span>
            <span className="ft-name">{row.name}</span>
            {!row.isDir && kinds.has(row.path) ? (
              <span className={"ft-dot ft-dot-" + kinds.get(row.path)} title={kinds.get(row.path)} />
            ) : null}
            {row.isDir && dirtyDirs.has(row.path) ? <span className="ft-dot ft-dot-dir" title="contains changes" /> : null}
            {row.isDir && row.open && !row.loaded ? <span className="ft-loading">…</span> : null}
          </button>
          {row.empty ? <p className="ft-empty" style={{ paddingLeft: 30 + row.depth * 14 }}>Empty folder. <button type="button" className="btn btn-sm btn-ghost" onClick={onRefresh}>Refresh</button></p> : null}
        </li>
      ))}
    </ul>
  );
}
