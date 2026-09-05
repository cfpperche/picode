import { useMemo, useState } from "react";
import { changeKinds, flattenTree } from "../lib/fileTree.js";
import { compactCount, groupChanges, totalsLabel } from "../lib/inspector.js";
import FileTree from "./FileTree.jsx";

const NO_DIRS = new Set();

// The working tree as a folder tree with counts (Paseo's Changes shape):
// every folder is open by default, its row sums what lies below, and a
// click opens the file's diff in the center. The scope chips narrow the
// list to what one agent's tools touched — an attribution no benchmark
// offers, and the one that matters when several agents share a folder.
export default function InspectorChanges({ changes, totals, scope, onScope, scopable, activePath, onOpen, onViewFiles }) {
  const [collapsed, setCollapsed] = useState(() => new Set());
  const grouped = useMemo(() => groupChanges(changes), [changes]);
  const expanded = useMemo(() => new Set([...grouped.dirs].filter((d) => !collapsed.has(d))), [grouped, collapsed]);
  const rows = useMemo(() => flattenTree(grouped.levels, expanded), [grouped, expanded]);
  const kinds = useMemo(() => changeKinds(changes), [changes]);
  const label = totalsLabel(totals);
  function toggle(path) {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }
  return (
    <>
      <div className="insp-sub" data-align-row>
        <span className="insp-stat" title={label ? `${totals.add} lines added, ${totals.del} removed` : undefined}>
          <span className="insp-stat-label">Uncommitted</span>
          {label ? <span className="insp-stat-counts">{totals.add ? <span className="gg-add">+{compactCount(totals.add)}</span> : null}{totals.del ? <span className="gg-del">−{compactCount(totals.del)}</span> : null}</span> : null}
        </span>
        {scopable ? (
          <div className="chip-group insp-scope" role="group" aria-label="Changes scope">
            <button type="button" className="cockpit-chip" aria-pressed={scope !== "agent"} onClick={() => onScope("all")}>All</button>
            <button type="button" className="cockpit-chip" aria-pressed={scope === "agent"} onClick={() => onScope("agent")}>This agent</button>
          </div>
        ) : null}
      </div>
      {changes.length === 0 ? (
        scope === "agent"
          ? <p className="insp-msg">No files from this agent yet. <button type="button" className="btn btn-sm" onClick={() => onScope("all")}>Show all</button></p>
          : <p className="insp-msg">No changes. <button type="button" className="btn btn-sm" onClick={onViewFiles}>View files</button></p>
      ) : (
        <FileTree
          rows={rows}
          kinds={kinds}
          dirtyDirs={NO_DIRS}
          selectedPath={activePath}
          onToggle={toggle}
          onOpen={onOpen}
          ariaLabel="Changed files"
          trailing={(row) => <Stat stat={grouped.stats.get(row.path)} />}
        />
      )}
    </>
  );
}

function Stat({ stat }) {
  if (!stat) return null;
  if (stat.binary) return <span className="insp-count insp-count-bin">bin</span>;
  if (!stat.add && !stat.del) return null;
  return (
    <span className="insp-count">
      {stat.add ? <span className="gg-add">+{compactCount(stat.add)}</span> : null}
      {stat.del ? <span className="gg-del">−{compactCount(stat.del)}</span> : null}
    </span>
  );
}
