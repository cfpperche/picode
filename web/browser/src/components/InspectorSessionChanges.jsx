import { useMemo } from "react";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import { changeTotals, compactCount, normalizeTouched, scopeChanges } from "../lib/inspector.js";
import InspectorChanges, { InspectorChangeGroup } from "./InspectorChanges.jsx";

// The Changes tab over every dirty checkout of the anchor's repository:
// one group per checkout, following the session's work instead of the
// anchor folder alone (docs/plans/inspector-session-changes.md). The view
// ({mode, shown, pill, switcher}) is resolved by pure logic; this component
// only renders it: a Following pill with Back, a switcher line with View,
// or one tree per group with the scope chips rendered once above them.
export default function InspectorSessionChanges({
  view, truncated = 0, backLabel,
  scope, onScope, scopable, touchedPaths,
  activePath, activeWorktree = "",
  onOpen, onViewFiles, onFollow, onBack,
}) {
  const scoped = useMemo(() => {
    const agent = scopable && scope === "agent";
    return (view.shown || []).map((g) => {
      if (!agent) return { group: g, changes: g.changes || [], totals: g.totals || changeTotals(g.changes) };
      const set = normalizeTouched(touchedPaths, g.path);
      const changes = scopeChanges(g.changes, set);
      return { group: g, changes, totals: changeTotals(changes) };
    });
  }, [view.shown, scopable, scope, touchedPaths]);
  const visible = scopable && scope === "agent" ? scoped.filter((s) => s.changes.length > 0) : scoped;
  const pill = view.pill || null;

  if (view.mode === "groups") {
    return (
      <>
        {scopable ? (
          <div className="insp-sub" data-align-row>
            <span className="insp-stat"><span className="insp-stat-label">Uncommitted</span></span>
            <div className="chip-group insp-scope" role="group" aria-label="Changes scope">
              <button type="button" className="cockpit-chip" aria-pressed={scope !== "agent"} onClick={() => onScope("all")}>All</button>
              <button type="button" className="cockpit-chip" aria-pressed={scope === "agent"} onClick={() => onScope("agent")}>This agent</button>
            </div>
          </div>
        ) : null}
        {visible.map(({ group, changes, totals }) => (
          <InspectorChangeGroup
            key={group.key}
            group={group}
            changes={changes}
            totals={totals}
            activePath={(group.ref || "") === activeWorktree ? activePath : ""}
            onOpen={(path) => onOpen(group, path)}
          />
        ))}
        {visible.length === 0 ? (
          <p className="insp-msg">No files from this agent yet. <button type="button" className="btn btn-sm" onClick={() => onScope("all")}>Show all</button></p>
        ) : null}
        {truncated > 0 ? <p className="insp-msg insp-note">+{truncated} more worktree{truncated === 1 ? "" : "s"} not scanned.</p> : null}
      </>
    );
  }

  const single = scoped[0] || { group: null, changes: [], totals: { add: 0, del: 0, files: 0 } };
  return (
    <>
      {pill ? (
        <div className="insp-pill" role="status" title={pill.path}>
          <span className="insp-pill-label">Following <strong>{pill.branch || shortPath(pill.path)}</strong></span>
          {single.totals && (single.totals.add || single.totals.del) ? (
            <span className="insp-stat-counts">
              {single.totals.add ? <span className="gg-add">+{compactCount(single.totals.add)}</span> : null}
              {single.totals.del ? <span className="gg-del">−{compactCount(single.totals.del)}</span> : null}
            </span>
          ) : null}
          <button type="button" className="btn btn-sm" onClick={onBack}>{backLabel || "Back"}</button>
          {(view.switcher || []).map((g) => (
            <span key={g.key} className="insp-pill-also">
              <span>Also {g.branch || shortPath(g.path)} ({(g.changes || []).length})</span>
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => onFollow(g.ref)}>View</button>
            </span>
          ))}
        </div>
      ) : null}
      <InspectorChanges
        changes={single.changes}
        totals={single.totals}
        scope={scope}
        onScope={onScope}
        scopable={scopable}
        emptyWhere={view.mode === "root" && (view.switcher || []).length > 0 && single.group ? (single.group.branch || shortPath(single.group.path)) : ""}
        activePath={((single.group && single.group.ref) || "") === activeWorktree ? activePath : ""}
        onOpen={(path) => onOpen(single.group, path)}
        onViewFiles={onViewFiles}
      />
      {view.mode === "root" && (view.switcher || []).map((g) => (
        <p key={g.key} className="insp-msg" role="status" title={g.path}>
          <span>Touched in <strong>{g.branch || shortPath(g.path)}</strong> · {(g.changes || []).length} file{(g.changes || []).length === 1 ? "" : "s"}</span>
          <button type="button" className="btn btn-sm" onClick={() => onFollow(g.ref)}>View</button>
        </p>
      ))}
      {truncated > 0 ? <p className="insp-msg insp-note">+{truncated} more worktree{truncated === 1 ? "" : "s"} not scanned.</p> : null}
    </>
  );
}
