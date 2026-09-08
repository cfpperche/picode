import { useEffect, useState } from "react";
import ShellTerm from "./ShellTerm.jsx";
import TermAttachBar from "./TermAttachBar.jsx";
import TermFindBar from "./TermFindBar.jsx";
import { bumpTermFontSize } from "@picode/shared/domain/termTheme.js";
import { scheduleTermFit } from "@picode/shared/domain/termFit.js";
import { terms } from "../lib/terms.js";
import { api } from "@picode/shared/client/api.js";

const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

export default function TermSurface({ term, error, hidden, onOpenFile, cwdKind, attach, onAttachClose, find, onFindClose }) {
  const [resuming, setResuming] = useState(false);
  const [resumeError, setResumeError] = useState("");
  // The message bar takes height from the pane; tmux hears about it in the
  // same frame instead of waiting out the resize observer's debounce.
  const open = !!attach;
  useEffect(() => {
    const entry = term && terms.get("sh:" + term.id);
    if (entry) scheduleTermFit(entry, true);
  }, [open, term && term.id]);
  if (!term && !error) return null;
  // One-click recovery of the pinned conversation (ADR-0084). The feed
  // flips the terminal to running when the launch lands.
  async function resumeLast() {
    setResuming(true);
    setResumeError("");
    try {
      await api(`/api/terminals/${encodeURIComponent(term.id)}/launch/start`, json("POST", { resume: true }));
    } catch (e) {
      setResumeError(e.message);
    } finally {
      setResuming(false);
    }
  }
  function onKey(e) {
    if (!(e.ctrlKey || e.metaKey) || e.shiftKey) return;
    if (e.key === "=" || e.key === "+") { e.preventDefault(); bumpTermFontSize(1); }
    else if (e.key === "-") { e.preventDefault(); bumpTermFontSize(-1); }
    else if (e.key === "0") { e.preventDefault(); bumpTermFontSize(0); }
  }
  return (
    <section className="term-surface" hidden={!!hidden} aria-label={term ? term.name : "Terminal"} onKeyDown={onKey}>
      {error ? (
        <p className="file-pane-msg">
          {error}{" "}
          <a href="#/system">Open System</a>
        </p>
      ) : term?.launchCli && !term.running ? (
        <div className="file-pane-msg" role="status">
          <span>{term.lostAtRestart ? "PiCode restarted while this terminal was running. " : "This CLI terminal is stopped. "}</span>
          {term.lastSession ? (
            <button type="button" className="btn btn-sm" disabled={resuming} onClick={resumeLast} title={term.lastSession.preview || undefined}>
              {resuming ? "Resuming…" : "Resume last session"}
            </button>
          ) : null}
          {" "}
          <a href="#/clis/terminals">Start from Agent CLIs</a>
          {resumeError ? <span> {resumeError}</span> : null}
        </div>
      ) : (
        <>
          <div className="term-body">
            <ShellTerm agentId={term.id} session={term.session} active={!hidden} cwd={term.cwd} cwdKind={cwdKind} onOpenFile={onOpenFile} />
            {find ? <TermFindBar termId={term.id} onClose={onFindClose} /> : null}
          </div>
          {term.launchCli && term.running && attach ? (
            <TermAttachBar term={term} seed={attach} onClose={onAttachClose} />
          ) : null}
        </>
      )}
    </section>
  );
}
