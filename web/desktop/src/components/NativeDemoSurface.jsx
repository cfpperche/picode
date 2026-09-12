import { useEffect, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import AppIcon from "./AppIcon.jsx";
import TermSurface from "./TermSurface.jsx";
import { toastError } from "../lib/toast.js";

// The hidden QA app for the native surface (ADR-0109; PICODE_DEMO_APP=1).
// It is the smallest consumer of the surface kind before the Canvas
// (docs/plans/matrix-app.md): the host's chrome shape — a header with the
// app name and Close, like AppSurface — and the fleet's first terminal
// rendered live through TermSurface, which is the pane hand-off case (one
// xterm per terminal follows the visible host; ShellTerm). It reads the
// fleet only through `host`, the client twin of Go's apps.Host, never App
// state; server calls go through the shared api() like any component.
export default function NativeDemoSurface({ manifest, hidden, onClose, host }) {
  const title = (manifest && manifest.name) || "Native demo";
  const term = ((host && host.fleet && host.fleet.terminals) || [])[0] || null;
  const [creating, setCreating] = useState(false);
  // Showing a terminal starts with the step the terminal tab takes
  // (openTermTab): POST /open ensures the tmux shell and answers the live
  // record — session name, running — that fleet rows and feed events do
  // not carry. Kept here, not in App state: the tab owns its own copy.
  const [opened, setOpened] = useState({});
  const [openError, setOpenError] = useState("");
  const termId = term ? term.id : "";
  useEffect(() => {
    if (!termId) return undefined;
    let stop = false;
    setOpenError("");
    api("/api/terminals/" + encodeURIComponent(termId) + "/open", { method: "POST" })
      .then((page) => { if (!stop) setOpened((m) => ({ ...m, [termId]: page })); })
      .catch((e) => { if (!stop) setOpenError(humanizeError(e && e.message ? e.message : String(e))); });
    return () => { stop = true; };
  }, [termId]);
  const live = term ? { ...term, ...(opened[term.id] || {}) } : null;
  async function newTerminal() {
    setCreating(true);
    try {
      // The store announces the new terminal on the feed (ADR-0048); the
      // shell's fleet state picks it up and it arrives here as
      // host.fleet.terminals[0] — no navigation, the pane appears in place.
      await api("/api/terminals", { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" });
    } catch (e) {
      toastError(e);
    } finally {
      setCreating(false);
    }
  }
  return (
    <section className="app-surface native-surface" aria-label={title} hidden={!!hidden}>
      <header className="ft-head">
        <span className="app-head-icon"><AppIcon name={manifest ? manifest.icon : "grid"} label={title} size={14} /></span>
        <h2 className="ft-title" title={title}>{title}</h2>
        {term ? <span className="ft-count" title="The fleet's first terminal, live">{term.name}</span> : null}
        <span className="ft-spacer" />
        <div className="app-head-right" data-align-row>
          {onClose ? (
            <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>
              Close
            </button>
          ) : null}
        </div>
      </header>
      {term ? (
        <TermSurface term={live} error={openError} hidden={!!hidden} onOpenFile={(p) => host.openFileTab("term", term.id, p)} />
      ) : (
        <div className="app-blank">
          <AppIcon name="grid" label={title} size={24} />
          <p className="app-blank-title">No terminals yet.</p>
          <button type="button" className="btn btn-sm" disabled={creating} onClick={newTerminal}>{creating ? "Creating…" : "New terminal"}</button>
        </div>
      )}
    </section>
  );
}
