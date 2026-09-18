import { useCallback, useEffect, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { hideDevServer, listDevServers, stopDevServer, unhideDevServer } from "@picode/shared/client/devservers.js";
import { askConfirm } from "../lib/confirm.js";
import { IconEllipsis, IconExternal, IconGlobe, IconReload, IconTerminal } from "./Icons.jsx";
import {
  canHide, canStop, hiddenRows, isOpenable, otherRows, sections,
  serverMeta, serverName, serverNote, stopPrompt,
} from "../lib/devServers.js";

// The Servers panel: what is listening on this machine right now, what each
// port actually is, and the verbs to act on it — Open, Copy, Show terminal,
// Stop, Hide. It is the one rail panel that is not about the anchor's folder:
// a dev server belongs to a terminal or an agent, and the row says which.
//
// The read reports every listener it can see, including the ones the panel
// keeps behind a disclosure (an API on a usual dev port that nobody in PiCode
// started), so revealing the rest is free. The poll is 5 s rather than a feed
// event: what is listening is machine state, not a store mutation. A Stop is
// optimistic — the row says "Stopping…" at once, and the poll that follows is
// the same 5 s poll the panel already runs.
const POLL_MS = 5000;
const NOTICE_MS = 4000;

export default function InspectorServers({ onOpen, onOpenOwner, hidden }) {
  const [state, setState] = useState({ servers: [], hidden: 0, readable: true, total: 0, truncated: false, error: "", loaded: false });
  const [busy, setBusy] = useState(false);
  const [showOther, setShowOther] = useState(false);
  const [showHidden, setShowHidden] = useState(false);
  const [pending, setPending] = useState({});
  const [notice, setNotice] = useState("");
  const gen = useRef(0);

  const load = useCallback(async (refresh) => {
    const mine = ++gen.current;
    if (refresh) setBusy(true);
    try {
      const page = await listDevServers({ refresh: !!refresh });
      if (mine !== gen.current) return;
      setState({
        servers: (page && page.servers) || [],
        hidden: (page && page.hidden) || 0,
        readable: !page || page.readable !== false,
        total: (page && page.total) || 0,
        truncated: !!(page && page.truncated),
        error: "",
        loaded: true,
      });
    } catch (e) {
      if (mine !== gen.current) return;
      // An error keeps the last good list: a panel that empties itself on a
      // hiccup reads as "nothing is running", which would be a lie.
      setState((s) => ({ ...s, error: e.message || "Could not list servers.", loaded: true }));
    } finally {
      if (mine === gen.current) setBusy(false);
    }
  }, []);

  useEffect(() => {
    if (hidden) return undefined;
    void load(false);
    const tick = setInterval(() => {
      if (!document.hidden) void load(false);
    }, POLL_MS);
    return () => { clearInterval(tick); gen.current++; };
  }, [hidden, load]);

  useEffect(() => {
    if (!notice) return undefined;
    const timer = setTimeout(() => setNotice(""), NOTICE_MS);
    return () => clearTimeout(timer);
  }, [notice]);

  // pendingFor is keyed by the process, not by the port: a new server on the
  // same port is a new row, and it must not inherit "Still running".
  const pendingFor = (row) => {
    const p = pending[row.port];
    return p && p.pid === row.pid && p.startKey === row.startKey ? p : null;
  };
  const setRowPending = (row, value) => setPending((cur) => {
    const next = { ...cur };
    if (value) next[row.port] = { ...value, pid: row.pid, startKey: row.startKey };
    else delete next[row.port];
    return next;
  });

  async function stopServer(row, force) {
    setRowPending(row, { kind: force ? "forcing" : "stopping" });
    try {
      const answer = await stopDevServer({ port: row.port, pid: row.pid, startKey: row.startKey, force });
      if (answer && answer.stopped) {
        // The row leaves as soon as the process is gone; the next poll agrees.
        setRowPending(row, null);
        await load(false);
        return;
      }
      setRowPending(row, { kind: "failed" });
    } catch (e) {
      setRowPending(row, null);
      setNotice(e && e.message ? e.message : "Could not stop that server.");
    }
    await load(false);
  }

  async function askStop(row, force) {
    const prompt = stopPrompt(row);
    const ok = await askConfirm({
      title: force ? `Force stop port ${row.port}?` : prompt.title,
      message: force ? "PiCode kills the process outright — anything it was running stops now, without a chance to clean up." : prompt.message,
      confirmLabel: force ? "Force stop" : "Stop",
      danger: true,
    });
    if (ok) await stopServer(row, force);
  }

  async function hideServer(row) {
    setRowPending(row, { kind: "hiding" });
    try {
      await hideDevServer({ port: row.port, pid: row.pid, startKey: row.startKey });
      setRowPending(row, null);
    } catch (e) {
      setRowPending(row, null);
      setNotice(e && e.message ? e.message : "Could not hide that server.");
    }
    await load(false);
  }

  async function showServer(row) {
    try {
      await unhideDevServer(row.hideId);
      setNotice(`Port ${row.port} is back in the list.`);
    } catch (e) {
      setNotice(e && e.message ? e.message : "Could not show that server.");
    }
    await load(false);
  }

  async function copyURL(row) {
    try {
      await navigator.clipboard.writeText(row.url);
      setNotice(`Copied ${row.url}`);
    } catch {
      setNotice("Could not copy the address.");
    }
  }

  const { mine, other } = sections(state.servers);
  const extra = otherRows(state.servers);
  const hiddenList = hiddenRows(state.servers);
  const extraVisible = showOther ? extra : [];
  const hiddenVisible = showHidden ? hiddenList : [];
  const rows = [...mine, ...other, ...extraVisible, ...hiddenVisible];
  const moreCount = showOther ? 0 : extra.length;
  const hiddenCount = showHidden ? 0 : hiddenList.length;
  const outside = [...other, ...extraVisible];

  const rowProps = (row) => ({
    row,
    pending: pendingFor(row),
    onOpen,
    onOpenOwner,
    onAskStop: askStop,
    onHide: hideServer,
    onShow: showServer,
    onCopy: copyURL,
  });

  return (
    <div className="insp-servers">
      <div className="insp-servers-head">
        <span className="insp-servers-title">Running on this machine</span>
        <button type="button" className="insp-btn" title={busy ? "Refreshing…" : "Refresh"} aria-label="Refresh servers" disabled={busy} onClick={() => load(true)}>
          <IconReload size={14} className={busy ? "insp-spin" : undefined} />
        </button>
      </div>
      {state.error ? (
        <p className="insp-notice" role="status">
          <span>{state.error}</span>
          <button type="button" className="btn btn-sm" onClick={() => load(true)} disabled={busy}>Try again</button>
        </p>
      ) : null}
      {notice ? <p className="insp-note" role="status">{notice}</p> : null}
      {state.loaded && !state.readable ? (
        <p className="insp-note">Port owners are not readable on this machine, so only the usual dev ports that answer are listed.</p>
      ) : null}
      {!state.loaded && !state.error ? (
        <div className="ft-skeleton" aria-label="Looking for servers" aria-busy="true">
          {[0, 1, 2].map((i) => (
            <div key={i} className="skel-line" style={{ width: `${70 - i * 15}%` }} />
          ))}
        </div>
      ) : null}
      {rows.length === 0 && state.loaded && !state.error ? (
        <p className="insp-msg">
          <span>{moreCount || hiddenCount ? "Nothing you started is listening." : "Nothing is listening. Start a dev server in a terminal — it shows up here."}</span>
          {onOpen ? <button type="button" className="btn btn-sm" onClick={() => onOpen("")}>Open a URL…</button> : null}
        </p>
      ) : null}
      {mine.length ? (
        <ul className="insp-server-list">
          {mine.map((row) => <ServerRow key={row.port} {...rowProps(row)} />)}
        </ul>
      ) : null}
      {outside.length ? (
        <>
          <p className="insp-group">Other ports</p>
          <ul className="insp-server-list">
            {outside.map((row) => <ServerRow key={row.port} {...rowProps(row)} />)}
          </ul>
        </>
      ) : null}
      {hiddenVisible.length ? (
        <>
          <p className="insp-group">Hidden here</p>
          <ul className="insp-server-list">
            {hiddenVisible.map((row) => <ServerRow key={row.port} {...rowProps(row)} />)}
          </ul>
        </>
      ) : null}
      {moreCount || hiddenCount || showOther || showHidden ? (
        <div className="insp-server-more-row">
          {moreCount ? <button type="button" className="btn btn-sm btn-ghost" onClick={() => setShowOther(true)}>Show {moreCount} more</button> : null}
          {hiddenCount ? <button type="button" className="btn btn-sm btn-ghost" onClick={() => setShowHidden(true)}>Show {hiddenCount} hidden</button> : null}
          {showOther && extra.length ? <button type="button" className="btn btn-sm btn-ghost" onClick={() => setShowOther(false)}>Fewer</button> : null}
          {showHidden && hiddenList.length ? <button type="button" className="btn btn-sm btn-ghost" onClick={() => setShowHidden(false)}>Hide again</button> : null}
        </div>
      ) : null}
      {state.truncated ? (
        <p className="insp-note">Showing the first {state.servers.length} of {state.total} listening ports.</p>
      ) : null}
    </div>
  );
}

// ServerRow is one listener. Line one is the port and what the page calls
// itself, line two the address, line three whose process it is and how long it
// has been up. The name and the address are one click (Open) only when the
// port is a page: a control channel that answers an API gets a plain block,
// never a button that leads nowhere, and its verbs live in the menu.
function ServerRow({ row, pending, onOpen, onOpenOwner, onAskStop, onHide, onShow, onCopy }) {
  const openable = isOpenable(row);
  const note = serverNote(row);
  const name = serverName(row);
  const meta = serverMeta(row);
  const body = (
    <>
      <span className="insp-server-glyph" aria-hidden="true"><IconGlobe size={15} /></span>
      <span className="insp-server-main">
        <span className="insp-server-top">
          <span className="insp-server-port">{row.port}</span>
          {name ? <span className="insp-server-name" title={name}>{name}</span> : null}
          {note ? <span className="insp-server-note">{note}</span> : null}
          {openable ? <span className="insp-server-open">Open</span> : null}
        </span>
        <span className="insp-server-sub" title={row.url}>{row.url}</span>
      </span>
    </>
  );

  return (
    <li className="insp-server" data-kind={row.kind}>
      {openable ? (
        <button type="button" className="insp-server-hit" onClick={() => onOpen(row.url, serverName(row))} title={"Open " + row.url + " in PiCode"}>
          {body}
        </button>
      ) : (
        <div className="insp-server-hit insp-server-static">{body}</div>
      )}
      <div className="insp-server-meta">
        <span title={meta}>{meta}</span>
      </div>
      {pending ? (
        <p className={"insp-server-pending" + (pending.kind === "failed" ? " is-failed" : "")} role="status">
          <span>{pendingLabel(pending)}</span>
          {pending.kind === "failed" ? (
            <button type="button" className="btn btn-sm btn-danger" onClick={() => onAskStop(row, true)}>Force stop</button>
          ) : null}
        </p>
      ) : null}
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button type="button" className="insp-btn insp-server-menu" aria-label={`Actions for port ${row.port}`} title="Actions"><IconEllipsis size={14} /></button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className="composer-more-pop" side="bottom" align="end" sideOffset={6} collisionPadding={8}>
            {openable ? (
              <DropdownMenu.Item className="composer-more-item" onSelect={() => onOpen(row.url, serverName(row))}>Open in PiCode</DropdownMenu.Item>
            ) : null}
            {openable ? (
              <DropdownMenu.Item className="composer-more-item" onSelect={() => window.open(row.url, "_blank", "noopener,noreferrer")}>
                <span>Open in browser</span><span className="insp-menu-hint"><IconExternal size={12} /></span>
              </DropdownMenu.Item>
            ) : null}
            <DropdownMenu.Item className="composer-more-item" onSelect={() => onCopy(row)}>Copy address</DropdownMenu.Item>
            {row.ownerKind && row.ownerId && onOpenOwner ? (
              <DropdownMenu.Item className="composer-more-item" onSelect={() => onOpenOwner({ kind: row.ownerKind, id: row.ownerId })}>
                <span>Show {row.ownerKind === "agent" ? "agent" : "terminal"}</span><span className="insp-menu-hint"><IconTerminal size={12} /></span>
              </DropdownMenu.Item>
            ) : null}
            {canStop(row) ? <DropdownMenu.Separator className="um-divider" /> : null}
            {canStop(row) ? (
              <DropdownMenu.Item className="composer-more-item is-danger" onSelect={() => onAskStop(row, false)}>Stop server…</DropdownMenu.Item>
            ) : null}
            {canHide(row) && !row.hidden ? (
              <DropdownMenu.Item className="composer-more-item" onSelect={() => onHide(row)}>Hide from the list</DropdownMenu.Item>
            ) : null}
            {row.hidden ? (
              <DropdownMenu.Item className="composer-more-item" onSelect={() => onShow(row)}>Show again</DropdownMenu.Item>
            ) : null}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </li>
  );
}

function pendingLabel(pending) {
  switch (pending.kind) {
    case "stopping": return "Stopping…";
    case "forcing": return "Forcing…";
    case "hiding": return "Hiding…";
    case "failed": return "Still running.";
    default: return "Working…";
  }
}
