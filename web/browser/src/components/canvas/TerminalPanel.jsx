import { memo, useEffect, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import TermSurface from "../TermSurface.jsx";
import { claimPane, releasePane } from "./paneOwnership.js";
import { resolveInteractiveTerminal } from "../../lib/agentTerminalView.js";

// TerminalPanel — the loaded body of a terminal or interactive-agent panel:
// the real xterm through TermSurface, the same instance the tab shows
// (docs/plans/matrix-app.md §2.2). A terminal starts the way its tab does:
// POST /open ensures the tmux shell and answers the live record (session
// name, running) that fleet rows do not carry, and the placeholder stays
// up until it does — no blank flash. An agent's TUI resolves the same runtime
// as its tab. Unmounting is unloading (paneOwnership.js): a pane nobody's
// tab holds has its socket suspended; loading again mounts TermSurface,
// whose ShellTerm kicks the suspended socket itself.
// The last answer of `POST /open` per terminal. Maximizing moves the body
// to another host, which is a remount: without the cache the panel would
// fall back to the placeholder for the round trip and flash. The POST runs
// again anyway, so a record that went stale is corrected in the same frame
// the server answers.
const liveRecords = new Map();

const TerminalPanel = memo(function TerminalPanel({ kind, target, terminals, cwd, hidden, focused, owned, onOpenFile, attach, onAttachClose, find, onFindClose, placeholder }) {
  const resolved = kind === "agent" ? resolveInteractiveTerminal(target, terminals, cwd) : null;
  const id = kind === "agent" ? resolved?.id : target.id;
  const [live, setLive] = useState(() => liveRecords.get(id) || null);
  const [error, setError] = useState("");
  // The tab closing while this panel shows the pane disposes the xterm
  // (App.closeTab → closeShellTerm): a fresh epoch mounts a fresh one.
  const [epoch, setEpoch] = useState(0);
  const ownedRef = useRef(owned);
  useEffect(() => {
    if (ownedRef.current && !owned) setEpoch((e) => e + 1);
    ownedRef.current = owned;
  }, [owned]);
  useEffect(() => {
    if (kind !== "terminal") return undefined;
    let stop = false;
    setError("");
    api("/api/terminals/" + encodeURIComponent(id) + "/open", { method: "POST" })
      .then((page) => { liveRecords.set(id, page); if (!stop) setLive(page); })
      .catch((e) => { liveRecords.delete(id); if (!stop) { setLive(null); setError(humanizeError(e && e.message ? e.message : String(e))); } });
    return () => { stop = true; };
  }, [kind, id, epoch]);
  useEffect(() => {
    if (!id) return undefined;
    claimPane(id);
    return () => releasePane(id, ownedRef.current);
  }, [id]);

  const term = kind === "agent"
    ? resolved?.term
    : live ? { ...target, ...live } : null;
  if (error) {
    return (
      <div className="cv-placeholder" role="status">
        <span>{error}</span>
        {/* PanelBody's rule: the action that starts work is accented. This
            one re-runs the open that failed, so it is the panel's primary. */}
        <button type="button" className="btn btn-sm btn-primary" onClick={() => setEpoch((e) => e + 1)}>Try again</button>
      </div>
    );
  }
  if (!term) return placeholder;
  return (
    <TermSurface
      key={epoch}
      term={term}
      hidden={!!hidden}
      autoFocus={!!focused}
      cwdKind={resolved?.cwdKind}
      onOpenFile={onOpenFile}
      attach={attach}
      onAttachClose={onAttachClose}
      find={find}
      onFindClose={() => onFindClose?.(id)}
    />
  );
});

export default TerminalPanel;
