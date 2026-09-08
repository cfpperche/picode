import { useLayoutEffect, useMemo, useRef, useState } from "react";
import { layoutUncommitted, isUncommittedHash } from "../lib/git/layout.js";
import { measuredBranchPath, rowGeometry } from "../lib/git/geometry.js";
import { historyDate, shortHash, worktreeName } from "../lib/git/model.js";

const STEP = 12, OFFSET = 10;
const colours = ["var(--accent)", "var(--ok)", "var(--warn)", "var(--danger)"];
const colour = i => colours[i % colours.length];

// A press held for half a second, without moving, is the phone's right-click
// (ADR-0096 on mobile). The click that follows a fired press is swallowed so
// the row does not also open. A real contextmenu (the desktop preview, a
// mouse) takes the same door.
function usePress(onLongPress) {
  const timer = useRef(0);
  const fired = useRef(false);
  const start = useRef(null);
  const cancel = () => { clearTimeout(timer.current); timer.current = 0; };
  return {
    onPointerDown: (e, payload) => {
      if (e.pointerType === "mouse" && e.button !== 0) return;
      fired.current = false;
      start.current = { x: e.clientX, y: e.clientY };
      cancel();
      timer.current = setTimeout(() => { fired.current = true; onLongPress(payload); }, 500);
    },
    onPointerMove: (e) => {
      if (!timer.current || !start.current) return;
      if (Math.abs(e.clientX - start.current.x) > 8 || Math.abs(e.clientY - start.current.y) > 8) cancel();
    },
    onPointerUp: cancel,
    onPointerCancel: cancel,
    onPointerLeave: cancel,
    onContextMenu: (e, payload) => { e.preventDefault(); cancel(); fired.current = true; onLongPress(payload); },
    swallowClick: () => { const f = fired.current; fired.current = false; return f; },
  };
}

export default function GitAncestry({ graph, refs, query, matches, activeMatch, onCommit, onWorktree, onActions }) {
  const press = usePress((payload) => onActions?.(payload));
  const list = useRef(null);
  const [geometry, setGeometry] = useState({ centers: [], height: 0 });
  const anchors = useMemo(() => (graph.worktrees || []).filter(wt => wt.uncommitted?.count > 0 && !wt.bare && !wt.prunable).map(wt => ({ hash: wt.head, wt })), [graph.worktrees]);
  const { placed, rows, dashed } = useMemo(() => layoutUncommitted(graph.commits || [], anchors), [graph.commits, anchors]);
  const width = Math.max(1, placed.columns) * STEP + OFFSET;
  useLayoutEffect(() => {
    const element = list.current;
    if (!element) return;
    const measure = () => {
      const children = [...element.children];
      if (!children.length || !children[0].getBoundingClientRect().height) return;
      const next = rowGeometry(children.map(child => child.getBoundingClientRect().height));
      setGeometry(previous => previous.height === next.height && previous.centers.every((n, i) => n === next.centers[i]) ? previous : next);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [rows, width]);
  useLayoutEffect(() => {
    if (!activeMatch) return;
    list.current?.querySelector(`[data-hash="${CSS.escape(activeMatch)}"]`)?.scrollIntoView({ block: "nearest" });
  }, [activeMatch]);
  return <div className="m-git-graph" role="region" aria-label="Commit ancestry graph" tabIndex={0}>
    <div className="m-git-graph-content" style={{ "--m-git-lanes": Math.min(width, 72) + "px" }}>
      <div className="m-git-ancestry-viewport" tabIndex={width > 72 ? 0 : undefined} aria-label={width > 72 ? "Scroll ancestry lanes" : undefined} style={{ height: geometry.height || 1 }}><svg className="m-git-ancestry" width={width} height={geometry.height || 1} aria-hidden="true" focusable="false">
        {placed.branches.map((branch, i) => <path key={i} d={measuredBranchPath(branch.lines, geometry, STEP, OFFSET)} stroke={colour(branch.colour)} />)}
        {dashed.map((branch, i) => <path key={"dirty:" + i} d={measuredBranchPath(branch.lines, geometry, STEP, OFFSET)} stroke={colour(branch.colour)} strokeDasharray="3 3" />)}
        {placed.vertices.map((vertex, i) => <circle key={rows[i].hash} cx={vertex.x * STEP + OFFSET} cy={geometry.centers[i] ?? 0} r={rows[i].hash === graph.head ? 4 : 3} fill={isUncommittedHash(rows[i].hash) ? "var(--bg-base)" : colour(vertex.colour)} stroke={colour(vertex.colour)} />)}
      </svg></div>
      <ol className="m-git-commits" ref={list}>{rows.map(row => {
        const dirty = isUncommittedHash(row.hash), wt = row.anchor?.wt;
        const dim = query && !matches.has(row.hash);
        return <li key={row.hash} data-hash={row.hash} className={"m-git-graph-row" + (dim ? " is-dim" : "") + (row.hash === activeMatch ? " is-match" : "")}>
          <button
            type="button"
            className="m-git-commit-row"
            onClick={() => { if (press.swallowClick()) return; dirty ? onWorktree(wt) : onCommit(row.hash); }}
            onPointerDown={(e) => press.onPointerDown(e, dirty ? { worktree: wt, uncommitted: true } : { commit: row, refs })}
            onPointerMove={press.onPointerMove}
            onPointerUp={press.onPointerUp}
            onPointerCancel={press.onPointerCancel}
            onPointerLeave={press.onPointerLeave}
            onContextMenu={(e) => press.onContextMenu(e, dirty ? { worktree: wt, uncommitted: true } : { commit: row, refs })}
          >
            <span className="m-git-commit-subject">{dirty ? `Uncommitted · ${worktreeName(wt)}` : row.subject || shortHash(row.hash)}</span>
            {dirty ? <span className="m-git-commit-meta">{wt.uncommitted.count} changed {wt.uncommitted.count === 1 ? "file" : "files"}{wt.self ? " · this worktree" : ""}{wt.agents?.length ? " · " + wt.agents.map(a => a.name).join(", ") : ""}</span> : <>
              {refs.some(ref => ref.hash === row.hash) ? <span className="m-git-refs">{refs.filter(ref => ref.hash === row.hash).map(ref => <span key={ref.kind + ref.name}>{ref.name}</span>)}</span> : null}
              <span className="m-git-commit-meta"><code>{shortHash(row.hash)}</code><span>{row.author}</span><time>{historyDate(row.at)}</time></span>
              {(graph.worktrees || []).filter(w => w.detached && !w.bare && !w.prunable && w.head === row.hash).map(w => <span key={w.path} className="m-git-hint">{worktreeName(w)}{w.agents?.length ? " · " + w.agents.map(a => a.name).join(", ") : ""}</span>)}
            </>}
          </button>
        </li>;
      })}</ol>
    </div>
  </div>;
}
