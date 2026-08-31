import { IconChevronRight, IconFile, IconFolder } from "./Icons.jsx";

// Pure render of the flattened tree (ADR-0028): rows come from
// lib/fileTree.flattenTree, decoration from the gitstatus-derived maps.
export default function FileTree({ rows, kinds, dirtyDirs, onToggle, onOpen }) {
  return (
    <ul className="ft-list" role="tree">
      {rows.map((row) => (
        <li key={row.path} role="treeitem" aria-expanded={row.isDir ? row.open : undefined}>
          <button
            type="button"
            className={"ft-row" + (row.isDir ? " ft-row-dir" : "")}
            style={{ paddingLeft: 8 + row.depth * 14 }}
            onClick={() => (row.isDir ? onToggle(row.path) : onOpen(row.path))}
            title={row.path}
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
        </li>
      ))}
    </ul>
  );
}
