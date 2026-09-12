import { formatMoney } from "@picode/shared/domain/providerUsage.js";
import { sessionsHash } from "../lib/routes.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";
import { folderLabel } from "@picode/shared/domain/dashboardStats.js";
import { terminalCliMark, terminalCliLabel } from "@picode/shared/domain/terminalCli.js";

// The five costliest sessions of the period — which CLI ran it, its name,
// where it ran, what it cost. Never a preview: this is the aggregate layer,
// the transcript view is one click away. Clicking a row opens that CLI's
// Sessions pane for the workspace (or every folder when none claims it).
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
              onClick={() => { location.hash = sessionsHash(s.workspaceId, s.cli); }}
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
