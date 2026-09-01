import { formatMoney } from "../lib/providerUsage.js";
import { sessionsHash } from "../lib/routes.js";
import { relTime, absTime } from "../lib/relTime.js";
import { folderLabel } from "../lib/dashboardStats.js";

// The five costliest sessions of the period — name, where it ran, what it
// cost. Never a preview: this is the aggregate layer, the transcript view
// is one click away. Clicking a row opens the sessions page for that
// workspace (or the machine-wide one when no workspace claims the folder),
// which is where opening/resuming already lives.
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
              title={(s.name ? s.name + " · " : "") + s.cwd + " · " + s.messages.toLocaleString() + " msgs · " + absTime(s.lastAt)}
              onClick={() => { location.hash = s.workspaceId ? sessionsHash(s.workspaceId) : "#/sessions"; }}
            >
              <span className="top-session-name">{name}</span>
              <span className="top-session-where">{sub}</span>
              <span className="top-session-cost">{formatMoney(s.cost, "usd")}</span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
