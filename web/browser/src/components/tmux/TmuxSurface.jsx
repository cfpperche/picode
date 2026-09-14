import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { IconChevronDown, IconChevronRight, IconExternal, IconReload, IconTmux, IconTrash, IconWarn } from "../Icons.jsx";
import { termTabId } from "../../lib/routes.js";
import { askConfirm } from "../../lib/confirm.js";
import { toast, toastError } from "../../lib/toast.js";
import "../../styles/tmux.css";

// TmuxSurface — the tmux app's native surface (ADR-0109; plan
// docs/plans/tmux-app.md; API internal/server/tmux.go). One tab, `x:tmux` /
// #/app/tmux: every session on the tmux server this daemon talks to — this
// daemon's own sessions, the leftovers no record claims, and the user's own
// tmux beside them.
//
// Three rules this component exists to keep, each one measured rather than
// imagined:
//
//  1. **No polling.** The read is refetched on the change feed
//     (`tmux.changed` after a reap, `terminal.*` when a session comes or
//     goes) and after this app's own actions — never on a timer. The tmux
//     read spawns subprocesses; a 3s interval against it would be a
//     subprocess every 3s per open tab, which is why the Tachyon inspector
//     this app learned from is *not* the model here.
//  2. **The app never says a session is nobody's.** A PiCode-shaped name the
//     store does not resolve is reported as exactly that ("not in PiCode's
//     records"), because a second PiCode instance on the same machine's tmux
//     server looks identical from here. The row is offered for removal with
//     the uncertainty written on it — never with a promise.
//  3. **Removal carries a receipt.** Every kill sends the session id, its
//     creation time and the pane pid the human was looking at; the server
//     re-reads tmux and refuses if any of the three moved. A row that
//     changed under the human cannot be killed by a stale click.
//
// The app reads the desktop only through `host` (fleet, openTab,
// openInteractive, feed) — never App state.

const TABS = [
  ["sessions", "Sessions"],
  ["server", "Server"],
];

const SCOPE_LABEL = {
  yours: "PiCode",
  unclaimed: "no record",
  notPiCode: "not PiCode's",
};

const KIND_LABEL = { agent: "agent", terminal: "terminal", other: "other" };

// ago is the same shape the rest of the desktop uses for an age: coarse,
// never a countdown, and empty when the fact is missing.
function ago(iso) {
  const at = Date.parse(iso || "");
  if (!at) return "";
  const sec = Math.max(0, Math.floor((Date.now() - at) / 1000));
  if (sec < 60) return sec + "s";
  if (sec < 3600) return Math.floor(sec / 60) + "m";
  if (sec < 86400) return Math.floor(sec / 3600) + "h";
  return Math.floor(sec / 86400) + "d";
}

function plural(n, one, many) {
  return n + " " + (n === 1 ? one : many);
}

// sessionRoute is the door a claimed session opens through: a terminal tab or
// the agent's shell. Nothing else on this screen opens anything — an unclaimed
// session has no owner surface to open, which is the point of the screen.
function sessionRoute(host, row) {
  if (row.scope !== "yours" || !row.ownerId) return null;
  if (row.kind === "agent") return { kind: "agent", id: row.ownerId };
  if (row.kind === "terminal") return { kind: "terminal", id: row.ownerId };
  return null;
}

export default function TmuxSurface({ manifest, host }) {
  const [tab, setTab] = useState("sessions");
  const [view, setView] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [scope, setScope] = useState("all");
  const [openRow, setOpenRow] = useState("");
  const [busy, setBusy] = useState("");
  const inFlight = useRef(false);

  const load = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    try {
      const next = await api("/api/tmux/server");
      setView(next);
      setError("");
    } catch (err) {
      setError(humanizeError(err));
    } finally {
      inFlight.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  // The feed doors this app listens to: its own reap notice and the terminal
  // lifecycle events that add or remove a session. Everything else in the
  // stream is somebody else's business.
  useEffect(() => {
    if (!host || typeof host.feed !== "function") return undefined;
    return host.feed((ev) => {
      if (!ev || typeof ev.type !== "string") return;
      if (ev.type === "tmux.changed" || ev.type.startsWith("terminal.")) load();
    });
  }, [host, load]);

  const server = view?.server;
  const sessions = view?.sessions || [];
  const absent = view?.absent || [];

  const rows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return sessions.filter((row) => {
      if (scope !== "all" && row.scope !== scope) return false;
      if (!needle) return true;
      return [row.name, row.ownerName, row.command, row.cwd, row.sessionId]
        .some((v) => String(v || "").toLowerCase().includes(needle));
    });
  }, [sessions, scope, search]);

  const openSession = useCallback((row) => {
    const route = sessionRoute(host, row);
    if (!route) return;
    if (route.kind === "agent") host.openInteractive?.(route.id);
    else host.openTab?.(termTabId(route.id));
  }, [host]);

  const reap = useCallback(async (row) => {
    const ok = await askConfirm({
      title: "Remove " + row.name + "?",
      message: "No terminal or agent in PiCode's records claims this session. Removing it stops whatever is still running inside it, and cannot be undone.",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    setBusy(row.name);
    try {
      const result = await api(`/api/tmux/sessions/${encodeURIComponent(row.name)}/reap`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          confirm: true,
          sessionId: row.sessionId,
          created: Math.floor(Date.parse(row.created) / 1000),
          panePid: row.panePid || 0,
        }),
      });
      if (result?.audit === "failed") {
        toast("Session removed, but the audit record could not be written.", "warn");
      } else {
        toast("Session removed.");
      }
      await load();
    } catch (err) {
      toastError(err);
      await load();
    } finally {
      setBusy("");
    }
  }, [load]);

  return (
    <div className="tmux-root" data-app={manifest?.id || "tmux"}>
      <header className="tmux-head">
        <div className="tmux-title">
          <IconTmux />
          <h1 title="tmux keeps terminals running after you close the window you were watching them in">
            {manifest?.name || "tmux"}
          </h1>
          <span className="tmux-sub">
            Every session on this machine's tmux server — PiCode's, and anything else running there.
          </span>
        </div>
        <div className="tmux-head-actions">
          {server ? (
            <span className="tmux-summary">
              {plural(server.sessions, "session", "sessions")}
              {server.clients > 0 ? " · " + plural(server.clients, "client", "clients") : ""}
            </span>
          ) : null}
          <button className="tmux-btn" onClick={load} disabled={loading}>
            <IconReload /> Refresh
          </button>
        </div>
      </header>

      <nav className="tmux-tabs" role="tablist">
        {TABS.map(([id, label]) => (
          <button
            key={id}
            role="tab"
            aria-selected={tab === id}
            className={"tmux-tab" + (tab === id ? " is-on" : "")}
            onClick={() => setTab(id)}
          >
            {label}
          </button>
        ))}
      </nav>

      {error ? (
        <div className="tmux-state">
          <IconWarn />
          <p>The tmux server could not be read. {error}</p>
          <button className="tmux-btn" onClick={load}>Try again</button>
        </div>
      ) : null}

      {!error && loading && !view ? (
        <div className="tmux-state">
          <p>Reading the tmux server…</p>
        </div>
      ) : null}

      {!error && view && tab === "server" ? (
        <section className="tmux-server">
          <dl className="tmux-facts">
            <dt>tmux</dt>
            <dd>{server.version || "not installed"}</dd>
            <dt>State</dt>
            <dd><span className={"tmux-health is-" + (server.running ? "ok" : "off")}>{server.running ? "running" : "not running"}</span></dd>
            <dt>Socket</dt>
            <dd><code>{server.socketPath || "—"}</code></dd>
            <dt>Sessions</dt>
            <dd>{server.sessions} · {server.attachedSessions} attached</dd>
            <dt>Clients</dt>
            <dd>{server.clients}</dd>
            <dt>Keyboard mode</dt>
            <dd>
              {server.extendedKeysFormat === "xterm"
                ? "xterm (Shift+Enter reaches your agents)"
                : server.extendedKeysFormat
                  ? server.extendedKeysFormat + " — PiCode sets xterm per attach; check Preferences → Terminal for the server value"
                  : "—"}
            </dd>
          </dl>

          <h2>In PiCode's records, not on this server</h2>
          {absent.length === 0 ? (
            <p className="tmux-empty">Every terminal PiCode knows about has its session running.</p>
          ) : (
            <ul className="tmux-absent">
              {absent.map((row) => (
                <li key={row.id}>
                  <span className="tmux-absent-name">{row.name}</span>
                  <code>{row.session}</code>
                  {row.lostAtRestart ? <span className="tmux-badge is-warn">lost at restart</span> : <span className="tmux-badge">stopped</span>}
                </li>
              ))}
            </ul>
          )}
        </section>
      ) : null}

      {!error && view && tab === "sessions" ? (
        <section className="tmux-sessions">
          <div className="tmux-filters">
            <input
              className="tmux-input"
              type="search"
              value={search}
              placeholder="Search sessions"
              aria-label="Search sessions"
              onChange={(e) => setSearch(e.target.value)}
            />
            <select className="tmux-input" value={scope} aria-label="Owner" onChange={(e) => setScope(e.target.value)}>
              <option value="all">All sessions</option>
              <option value="yours">PiCode's</option>
              <option value="unclaimed">Not in PiCode's records</option>
              <option value="notPiCode">Not PiCode's</option>
            </select>
          </div>

          {rows.length === 0 ? (
            <div className="tmux-state">
              <p>
                {sessions.length === 0
                  ? "No tmux sessions on this machine. Start an agent or open a terminal, then refresh."
                  : "No session matches that filter."}
              </p>
              {sessions.length === 0 ? (
                <button className="tmux-btn" onClick={load}>Refresh</button>
              ) : (
                <button className="tmux-btn" onClick={() => { setSearch(""); setScope("all"); }}>Clear filters</button>
              )}
            </div>
          ) : (
            <ul className="tmux-list">
              {rows.map((row) => {
                const open = openRow === row.name;
                const route = sessionRoute(host, row);
                return (
                  <li key={row.name} className={"tmux-row is-" + row.scope}>
                    <div className="tmux-row-main">
                      <button
                        className="tmux-row-toggle"
                        aria-expanded={open}
                        onClick={() => setOpenRow(open ? "" : row.name)}
                      >
                        {open ? <IconChevronDown /> : <IconChevronRight />}
                        <span className="tmux-row-name">{row.ownerName || row.name}</span>
                        <span className="tmux-badge is-kind">{KIND_LABEL[row.kind] || row.kind}</span>
                        {row.scope !== "yours" ? <span className="tmux-badge is-dim">{SCOPE_LABEL[row.scope]}</span> : null}
                        {row.attached > 0 ? <span className="tmux-badge is-ok">{plural(row.attached, "client", "clients")}</span> : null}
                        {row.panes > 1 ? <span className="tmux-badge">{plural(row.panes, "pane", "panes")}</span> : null}
                        {row.dead ? <span className="tmux-badge is-warn">{typeof row.exitCode === "number" ? "exited " + row.exitCode : "exited"}</span> : null}
                      </button>
                      <span className="tmux-row-facts">
                        {row.command ? <code>{row.command}</code> : null}
                        <span className="tmux-when">{ago(row.created)}</span>
                      </span>
                      <span className="tmux-row-actions">
                        {route ? (
                          <button className="tmux-btn" onClick={() => openSession(row)}>
                            <IconExternal /> Open
                          </button>
                        ) : null}
                      </span>
                    </div>
                    {open ? (
                      <dl className="tmux-details">
                        <dt>Session</dt>
                        <dd><code>{row.name}</code> · {row.sessionId}</dd>
                        <dt>Started</dt>
                        <dd>{new Date(row.created).toLocaleString()} ({ago(row.created)} ago)</dd>
                        <dt>Folder</dt>
                        <dd>{row.cwd ? <code>{row.cwd}</code> : "—"}</dd>
                        <dt>Windows</dt>
                        <dd>{row.windows} · {plural(row.panes, "pane", "panes")}</dd>
                        <dt>Command</dt>
                        <dd><code>{row.command || "—"}</code></dd>
                        {row.scope === "unclaimed" ? (
                          <>
                            <dt>Owner</dt>
                            <dd>
                              No terminal or agent with this name is in PiCode's records. It may be a leftover from a
                              previous run, or a session belonging to another PiCode on this machine — PiCode cannot
                              tell those apart, so nothing removes it on its own.
                            </dd>
                            <dt>Remove</dt>
                            <dd>
                              <button
                                className={"tmux-btn is-danger" + (busy === row.name ? " is-busy" : "")}
                                disabled={busy === row.name}
                                onClick={() => reap(row)}
                              >
                                <IconTrash /> {busy === row.name ? "Removing…" : "Remove this session"}
                              </button>
                            </dd>
                          </>
                        ) : null}
                        {row.scope === "notPiCode" ? (
                          <>
                            <dt>Owner</dt>
                            <dd>This session is not in PiCode's namespace. It is yours, and it is read-only here.</dd>
                          </>
                        ) : null}
                      </dl>
                    ) : null}
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      ) : null}
    </div>
  );
}
