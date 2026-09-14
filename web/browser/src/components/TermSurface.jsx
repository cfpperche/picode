import { useEffect, useState } from "react";
import ShellTerm from "./ShellTerm.jsx";
import TermAttachBar from "./TermAttachBar.jsx";
import TermFindBar from "./TermFindBar.jsx";
import { IconPlay, IconReload, IconTerminal, IconWarn } from "./Icons.jsx";
import { bumpTermFontSize } from "@picode/shared/domain/termTheme.js";
import { scheduleTermFit } from "@picode/shared/domain/termFit.js";
import { absTime, relTime } from "@picode/shared/domain/relTime.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { terms } from "../lib/terms.js";
import { api, humanizeError } from "@picode/shared/client/api.js";

const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

// The catalog's preview is the conversation's opening words and can carry a
// paragraph; the meta row is one line (CSS ellipsis keeps it to one).
function oneLine(text) {
  return String(text || "").replace(/\s+/g, " ").trim();
}

// TermMessage — the pane with no terminal in it, the surface's own empty
// state (docs/benchmarks.md, "Empty states teach"). Two states land here: a
// CLI terminal that is stopped, and a terminal that refused to open. Both
// read the same way — a mark, what is true, what a Resume would continue,
// why we know, and the actions — instead of a sentence pinned to the
// top-left of a black well.
//
// The pinned conversation of ADR-0084 is the meta line: it was reachable
// only through the resume button's tooltip before, which is the one thing
// a reader decides on ("is that the session I was in?"). `why` is a fact we
// hold in the record (a failed launch); `alert` is one that just happened
// (this pane's own Resume came back with an error), which is the only live
// region — the rest is already true when the pane mounts.
function TermMessage({ tone, icon, head, meta, metaTitle, why, alert, children }) {
  return (
    <div className="term-msg" data-tone={tone || undefined}>
      {icon ? <span className="term-msg-mark" aria-hidden="true">{icon}</span> : null}
      <p className="term-msg-head" role="status">{head}</p>
      {meta ? <p className="term-msg-meta" title={metaTitle || undefined}>{meta}</p> : null}
      {why ? <p className="term-msg-why">{why}</p> : null}
      {alert ? <p className="term-msg-why is-danger" role="alert">{alert}</p> : null}
      <div className="term-msg-acts">{children}</div>
    </div>
  );
}

// TermWindow — the picture of a terminal window the empty state sits in
// (owner call, 2026-09-14: "a frame that simulates a terminal", after the
// ohmyz.sh mock): titlebar, traffic lights, a real title — the terminal's
// own name and folder — and the content hugged inside, centered on the pane
// instead of filling it. The chrome is decorative (aria-hidden); the state
// text inside keeps its own live region.
function TermWindow({ title, titleFull, children }) {
  return (
    <div className="term-empty">
      <div className="term-window">
        <div className="term-window-bar" aria-hidden="true">
          <span className="term-window-lights"><i /><i /><i /></span>
          <span className="term-window-title" title={titleFull || undefined}>{title}</span>
        </div>
        {children}
      </div>
    </div>
  );
}

// autoFocus (default true): the visible pane takes the keyboard. The Canvas
// passes false for every panel but the focused one (ShellTerm).
export default function TermSurface({ term, error, hidden, autoFocus = true, onOpenFile, cwdKind, attach, onAttachClose, find, onFindClose }) {
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
      setResumeError(humanizeError(e && e.message ? e.message : String(e)));
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
  const last = (term && term.lastSession) || null;
  const cli = term && term.launchCli ? terminalCliLabel(term.launchCli) : "";
  const session = last ? oneLine(last.name) || oneLine(last.preview) : "";
  // Nothing pinned (ADR-0084) is a reason, not an empty gap: the meta says
  // why there is no Resume to press instead of leaving the reader with a
  // button that is missing for no stated cause.
  const meta = [cli, last ? session : "no session to resume", last ? relTime(last.updatedAt) : ""].filter(Boolean).join(" · ");
  const metaTitle = last ? [oneLine(last.preview), absTime(last.updatedAt)].filter(Boolean).join(" · ") : "";
  const attempt = term && term.launchAttempt && term.launchAttempt.error ? term.launchAttempt.error : "";
  // The window's title bar carries the terminal's own name and folder, the
  // way a real terminal titles itself ("codex — term-stopped-state").
  const cwdBase = term && term.cwd ? String(term.cwd).split("/").filter(Boolean).pop() : "";
  // An empty pane is app chrome and follows the app theme; the class moves
  // the ground off the terminal's own (see .term-surface.is-empty).
  const empty = !!(error || (term && term.launchCli && !term.running));
  return (
    <section className={"term-surface" + (empty ? " is-empty" : "")} hidden={!!hidden} aria-label={term ? term.name : "Terminal"} onKeyDown={onKey}>
      {error ? (
        <TermWindow title="Terminal">
          <TermMessage tone="warn" icon={<IconWarn size={18} />} head={error}>
            <a className="btn" href="#/system">Open System</a>
          </TermMessage>
        </TermWindow>
      ) : term?.launchCli && !term.running ? (
        <TermWindow
          title={term.name + (cwdBase ? " — " + cwdBase : "")}
          titleFull={term.cwd}
        >
          <TermMessage
            head={term.lostAtRestart ? "PiCode restarted while this terminal was running." : "This CLI terminal is stopped."}
            meta={meta}
            metaTitle={metaTitle}
            why={attempt}
            alert={resumeError}
          >
            {last ? (
              <button type="button" className="btn btn-primary" disabled={resuming} onClick={resumeLast}>
                {resuming ? <IconReload size={13} className="term-msg-spin" /> : <IconPlay size={12} />}
                {resuming ? "Resuming…" : "Resume last session"}
              </button>
            ) : null}
            <a className="btn" href="#/clis">Start from Agent CLIs</a>
          </TermMessage>
        </TermWindow>
      ) : (
        <>
          <div className="term-body">
            <ShellTerm agentId={term.id} session={term.session} active={!hidden} autoFocus={autoFocus} cwd={term.cwd} cwdKind={cwdKind} onOpenFile={onOpenFile} />
            {find ? <TermFindBar termId={term.id} onClose={onFindClose} /> : null}
          </div>
          {attach ? (
            <TermAttachBar term={term} seed={attach} ownerKind={cwdKind === "agent" ? "agent" : "term"} onClose={onAttachClose} />
          ) : null}
        </>
      )}
    </section>
  );
}
