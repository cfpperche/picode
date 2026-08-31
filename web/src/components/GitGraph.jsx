import { Fragment, useEffect, useMemo, useRef } from "react";
import { layout, layoutUncommitted, UNCOMMITTED, branchPath, colourAt, GRID } from "../lib/gitgraph.js";
import { IconRemote } from "./Icons.jsx";

// One row per commit, with the branch lines drawn as a single SVG layer behind
// them. Row height is GRID.y so the drawing and the text stay aligned without
// measuring anything.

function shortDate(at) {
  if (!at) return "";
  const d = new Date(at * 1000);
  const now = new Date();
  const opts = d.getFullYear() === now.getFullYear()
    ? { day: "2-digit", month: "short" }
    : { day: "2-digit", month: "short", year: "numeric" };
  return d.toLocaleDateString(undefined, opts);
}

function refKindClass(kind) {
  if (kind === "remote") return "gg-ref-remote";
  if (kind === "tag") return "gg-ref-tag";
  return "gg-ref-head";
}

// A remote name is "origin/feat-x": the remote is context, the branch is the
// point. One span keeps them glued against the pill's flex gap.
function refLabel(ref) {
  if (ref.kind !== "remote") return ref.name;
  const cut = ref.name.indexOf("/");
  if (cut < 0) return ref.name;
  return (
    <span className="gg-ref-name">
      <span className="gg-ref-origin">{ref.name.slice(0, cut)}</span>
      {ref.name.slice(cut + 1)}
    </span>
  );
}

export default function GitGraph({
  graph, selected, onSelect, matches, activeMatch, detail, detailHeight, onSizerDown,
}) {
  const commits = graph.commits || [];
  const listRef = useRef(null);

  // The detail opens below the clicked row, possibly past the bottom edge.
  // Panel first, row second: if both cannot fit, the row wins — it is the
  // thing the reader clicked. "nearest" keeps an already-visible row still.
  useEffect(() => {
    if (!selected || !listRef.current) return;
    const inline = listRef.current.querySelector(".gg-inline");
    if (inline) inline.scrollIntoView({ block: "nearest" });
    const row = listRef.current.querySelector(".gg-row-on");
    if (row) row.scrollIntoView({ block: "nearest" });
    // commits is a dependency because a parent link can select a row that only
    // exists after the window grows — the scroll has to wait for the data.
  }, [selected, commits]);

  // Walking the matches jumps anywhere in the window, so centre the target;
  // "nearest" would leave a far match glued to an edge.
  useEffect(() => {
    if (!activeMatch || !listRef.current) return;
    const row = listRef.current.querySelector(".gg-row-match-on");
    if (row) row.scrollIntoView({ block: "center" });
  }, [activeMatch]);

  // A dirty working tree prepends a pseudo row over HEAD (ADR-0038). `rows` is
  // the single source for every index: dots, head dot, selection, expandAt —
  // never commits[i] with a scattered +1.
  const dirty = graph.uncommitted ? graph.uncommitted.count : 0;
  const { placed, rows, dashed } = useMemo(
    () =>
      dirty > 0
        ? layoutUncommitted(commits, graph.head)
        : { placed: layout(commits), rows: commits, dashed: null },
    [commits, graph.head, dirty],
  );

  // The inline detail is a row in the flow, so the list below it moves by
  // itself; the SVG is absolute, so every line and dot below the selected row
  // detours by the same amount through expandAt/expandY.
  const selIdx = detail ? rows.findIndex((r) => r.hash === selected) : -1;
  const expandY = selIdx > -1 ? detailHeight : 0;

  // A commit carries every ref that points at it, and a branch carries the
  // agents living in the worktree checked out on it — the one thing this graph
  // can show that a git client cannot.
  const refsByHash = useMemo(() => {
    const map = new Map();
    for (const ref of graph.refs || []) {
      if (!map.has(ref.hash)) map.set(ref.hash, []);
      map.get(ref.hash).push(ref);
    }
    for (const list of map.values()) {
      list.sort((a, b) => a.kind.localeCompare(b.kind) || a.name.localeCompare(b.name));
    }
    return map;
  }, [graph.refs]);

  const agentsByBranch = useMemo(() => {
    const map = new Map();
    for (const wt of graph.worktrees || []) {
      if (wt.branch && (wt.agents || []).length) map.set(wt.branch, wt.agents);
    }
    return map;
  }, [graph.worktrees]);

  const width = Math.max(1, placed.columns) * GRID.x + GRID.offsetX;
  const height = Math.max(rows.length, 1) * GRID.y + expandY;

  // With a query active every row still draws — non-matches dim, the walked
  // match lights up. The uncommitted row is not a commit, so it only dims.
  const rowClass = (hash, extra = "") => {
    let cls = "gg-row" + extra;
    if (selected === hash) cls += " gg-row-on";
    if (matches) {
      if (hash === activeMatch) cls += " gg-row-match-on";
      else if (!matches.has(hash)) cls += " gg-row-dim";
    }
    return cls;
  };

  const inlineDetail = (
    <li className="gg-inline" style={{ height: detailHeight }}>
      {detail}
      <div className="gg-inline-sizer" onPointerDown={onSizerDown} aria-hidden="true" />
    </li>
  );

  return (
    <div className="gg-rows" style={{ "--gg-graph-w": width + "px", "--gg-row-h": GRID.y + "px" }}>
      <svg
        className="gg-svg"
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        aria-hidden="true"
        focusable="false"
      >
        {placed.branches.map((branch, i) => (
          <path
            key={i}
            className="gg-line"
            d={branchPath(branch.lines, { expandAt: selIdx, expandY })}
            stroke={colourAt(branch.colour)}
          />
        ))}
        {dashed && dashed.lines.length ? (
          <path
            className="gg-line gg-line-dashed"
            d={branchPath(dashed.lines, { expandAt: selIdx, expandY })}
            stroke={colourAt(dashed.colour)}
          />
        ) : null}
        {placed.vertices.map((v, i) => {
          const isHead = rows[i].hash === graph.head;
          const isDirty = rows[i].hash === UNCOMMITTED;
          return (
            <circle
              key={i}
              className={"gg-dot" + (isHead ? " gg-dot-head" : "")}
              cx={v.x * GRID.x + GRID.offsetX}
              cy={i * GRID.y + GRID.offsetY + (selIdx > -1 && i > selIdx ? expandY : 0)}
              r={isHead ? 4.5 : 3.5}
              fill={isDirty ? "var(--bg-base)" : colourAt(v.colour)}
              style={isDirty ? { stroke: colourAt(v.colour) } : undefined}
            />
          );
        })}
      </svg>

      <ol className="gg-list" ref={listRef}>
        {rows.map((c, i) => {
          if (c.hash === UNCOMMITTED) {
            return (
              <Fragment key={UNCOMMITTED}>
                <li>
                  <button
                    type="button"
                    className={rowClass(UNCOMMITTED, " gg-row-uncommitted")}
                    aria-current={selected === UNCOMMITTED ? "true" : undefined}
                    onClick={() => onSelect && onSelect(UNCOMMITTED)}
                  >
                    Uncommitted Changes ({dirty})
                  </button>
                </li>
                {i === selIdx ? inlineDetail : null}
              </Fragment>
            );
          }
          const refs = refsByHash.get(c.hash) || [];
          return (
            <Fragment key={c.hash}>
            <li>
              <button
                type="button"
                className={rowClass(c.hash)}
                aria-current={selected === c.hash ? "true" : undefined}
                onClick={() => onSelect && onSelect(c.hash)}
              >
                <span className="gg-refs">
                  {refs.map((ref) => {
                    const agents = ref.kind === "head" ? agentsByBranch.get(ref.name) : null;
                    return (
                      <span key={ref.kind + ref.name} className={"gg-ref " + refKindClass(ref.kind)}>
                        {ref.kind === "remote" ? <IconRemote /> : null}
                        {refLabel(ref)}
                        {agents
                          ? agents.map((a) => (
                              <span key={a.id} className="gg-occupant" title={`Agent ${a.name} works here`}>
                                {a.name}
                              </span>
                            ))
                          : null}
                      </span>
                    );
                  })}
                </span>
                <span className="gg-subject" title={c.subject}>{c.subject}</span>
                <span className="gg-author">{c.author}</span>
                <span className="gg-date">{shortDate(c.at)}</span>
                <span className="gg-hash">{c.hash.slice(0, 7)}</span>
              </button>
            </li>
            {i === selIdx ? inlineDetail : null}
            </Fragment>
          );
        })}
      </ol>
    </div>
  );
}
