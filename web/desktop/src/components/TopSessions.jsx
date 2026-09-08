import { formatMoney } from "@picode/shared/domain/providerUsage.js";
import { sessionsHash } from "../lib/routes.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";
import { folderLabel } from "@picode/shared/domain/dashboardStats.js";
import { terminalCliMark, terminalCliLabel } from "@picode/shared/domain/terminalCli.js";

// The five costliest sessions of the period — which CLI ran it, its name,
// where it ran, what it cost. Never a preview: this is the aggregate layer,
// the transcript view is one click away. Clicking a row opens the sessions
// page for that workspace (or the machine-wide one when no workspace claims
// the folder), which is where opening/resuming already lives.
//
// The row carries its CLI mark rather than routing to that CLI's list.
// `#/clis/sessions/<wsId>` is ADR-0079's shape and does not name a CLI;
// widening it is that ADR's call, not this panel's. The sessions view opens
// on Pi with its own CLI select one control away, so the mark tells the
// reader which one to pick instead of landing them somewhere silently wrong.
export default function TopSessions({ items }) {
  if (!items || items.length === 0) {
    return <p className="dash-empty">No sessions in this period.</p>;
  }
  return (
    <ul className="top-sessions">
      {items.map((s) => {
        // pi rarely names a session, so the folder is the primary label
        // when there is no name and the "where" slot shows when instead.
        const where = s.workspace || folderLabel(s.cwd);
        const name = s.name || where;
        const sub = [s.name ? where : "", relTime(s.lastAt)].filter(Boolean).join(" · ");
        return (
          <li key={s.path}>
            <button
              type="button"
              className="top-session-row"
              title={(s.cli ? terminalCliLabel(s.cli) + " · " : "") + (s.name ? s.name + " · " : "") + s.cwd + " · " + s.messages.toLocaleString() + " msgs · " + absTime(s.lastAt)}
              onClick={() => { location.hash = sessionsHash(s.workspaceId); }}
            >
              <span className="top-session-name">
                {s.cli ? <span className="top-session-cli">{terminalCliMark(s.cli)}</span> : null}
                {name}
              </span>
              <span className="top-session-where">{sub}</span>
              <span className="top-session-cost">{formatMoney(s.cost, "usd")}</span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
