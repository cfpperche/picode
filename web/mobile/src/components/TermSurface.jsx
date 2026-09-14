import { useState } from "react";
import ShellTerm from "./ShellTerm.jsx";
import { IconPlay, IconReload, IconTerminal, IconWarn } from "./Icons.jsx";
import { bumpTermFontSize } from "@picode/shared/domain/termTheme.js";
import { absTime, relTime } from "@picode/shared/domain/relTime.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { api, humanizeError } from "@picode/shared/client/api.js";

const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

// The catalog's preview is the conversation's opening words and can carry a
// paragraph; the meta row is one line (CSS ellipsis keeps it to one).
function oneLine(text) {
  return String(text || "").replace(/\s+/g, " ").trim();
}

// TermMessage — the pane with no terminal in it: a stopped CLI terminal or a
// terminal that refused to open. The phone draws the same anatomy the
// desktop pane does (web/browser TermSurface), one column wide: a mark, what
// is true, what a Resume would continue (the ADR-0084 pin, which the phone
// only had in a tooltip), why we know, and the actions.
function TermMessage({ tone, icon, head, meta, metaTitle, why, alert, children }) {
  return (
    <div className="term-msg" data-tone={tone || undefined}>
      <span className="term-msg-mark" aria-hidden="true">{icon}</span>
      <p className="term-msg-head" role="status">{head}</p>
      {meta ? <p className="term-msg-meta" title={metaTitle || undefined}>{meta}</p> : null}
      {why ? <p className="term-msg-why">{why}</p> : null}
      {alert ? <p className="term-msg-why is-danger" role="alert">{alert}</p> : null}
      <div className="term-msg-acts">{children}</div>
    </div>
  );
}

export default function TermSurface({ term, error, hidden, onOpenFile, cwdKind }) {
  const [resuming, setResuming] = useState(false);
  const [resumeError, setResumeError] = useState("");
  if (!term && !error) return null;
  // One-click recovery of the pinned conversation (ADR-0084).
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
  // An empty pane is app chrome and follows the app theme; the class moves
  // the ground off the terminal's own (see .term-surface.is-empty).
  const empty = !!(error || (term && term.launchCli && !term.running));
  return (
    <section className={"term-surface" + (empty ? " is-empty" : "")} hidden={!!hidden} aria-label={term ? term.name : "Terminal"} onKeyDown={onKey}>
      {error ? (
        <TermMessage tone="warn" icon={<IconWarn size={18} />} head={error}>
          <a className="btn" href="#/system">Open System</a>
        </TermMessage>
      ) : term?.launchCli && !term.running ? (
        <TermMessage
          icon={<IconTerminal size={18} />}
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
      ) : (
        <ShellTerm agentId={term.id} session={term.session} active={!hidden} cwd={term.cwd} cwdKind={cwdKind} onOpenFile={onOpenFile} />
      )}
    </section>
  );
}