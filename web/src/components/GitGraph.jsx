import { useMemo } from "react";
import { layout, branchPath, colourAt, GRID } from "../lib/gitgraph.js";

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

export default function GitGraph({ graph, selected, onSelect }) {
  const commits = graph.commits || [];

  const placed = useMemo(() => layout(commits), [commits]);

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
  const height = Math.max(commits.length, 1) * GRID.y;

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
            d={branchPath(branch.lines)}
            stroke={colourAt(branch.colour)}
          />
        ))}
        {placed.vertices.map((v, i) => (
          <circle
            key={i}
            className={"gg-dot" + (commits[i].hash === graph.head ? " gg-dot-head" : "")}
            cx={v.x * GRID.x + GRID.offsetX}
            cy={i * GRID.y + GRID.offsetY}
            r={commits[i].hash === graph.head ? 4.5 : 3.5}
            fill={colourAt(v.colour)}
          />
        ))}
      </svg>

      <ol className="gg-list">
        {commits.map((c) => {
          const refs = refsByHash.get(c.hash) || [];
          return (
            <li key={c.hash}>
              <button
                type="button"
                className={"gg-row" + (selected === c.hash ? " gg-row-on" : "")}
                aria-current={selected === c.hash ? "true" : undefined}
                onClick={() => onSelect && onSelect(c.hash)}
              >
                <span className="gg-refs">
                  {refs.map((ref) => {
                    const agents = ref.kind === "head" ? agentsByBranch.get(ref.name) : null;
                    return (
                      <span key={ref.kind + ref.name} className={"gg-ref " + refKindClass(ref.kind)}>
                        {ref.name}
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
          );
        })}
      </ol>
    </div>
  );
}
