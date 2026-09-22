import { useState } from "react";
import ShellTerm from "./ShellTerm.jsx";
import { IconPlay, IconReload, IconTerminal, IconWarn } from "./Icons.jsx";
import { bumpTermFontSize } from "@picode/shared/domain/termTheme.js";
import { absTime, relTime } from "@picode/shared/domain/relTime.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";

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
      {icon ? <span className="term-msg-mark" aria-hidden="true">{icon}</span> : null}
      <p className="term-msg-head" role="status">{head}</p>
      {meta ? <p className="term-msg-meta" title={metaTitle || undefined}>{meta}</p> : null}
      {why ? <p className="term-msg-why">{why}</p> : null}
      {alert ? <p className="term-msg-why is-danger" role="alert">{alert}</p> : null}
      <div className="term-msg-acts">{children}</div>
    </div>
  );
}

// TermWindow — the same simulated terminal window the desktop pane draws
// (owner call, 2026-09-14): titlebar, traffic lights, the terminal's own
// name and folder as the title, content hugged inside and the window
// centered on the pane. The chrome is decorative (aria-hidden).
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

export default function TermSurface({ adopt = "", term, error, hidden, onOpenFile, cwdKind }) {
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
  // The window's title bar carries the terminal's own name and folder, the
  // way a real terminal titles itself.
  const cwdBase = term && term.cwd ? String(term.cwd).split("/").filter(Boolean).pop() : "";
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
          {adopt ? <AdoptBar term={term} cli={adopt} /> : null}
          <ShellTerm agentId={term.id} session={term.session} active={!hidden} cwd={term.cwd} cwdKind={cwdKind} onOpenFile={onOpenFile} />
        </>
      )}
    </section>
  );
}

// Make agent (ADR-0184): a catalog CLI typed into this shell can become an
// agent bound to it — named, granted and in the fleet. Only on a click.
function AdoptBar({ term, cli }) {
  const [busy, setBusy] = useState(false);
  async function adopt() {
    setBusy(true);
    try {
      const a = await api("/api/terminals/" + encodeURIComponent(term.id) + "/adopt", { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" });
      toast.ok((a && a.name ? a.name : terminalCliLabel(cli)) + " is an agent now.");
    } catch (e) {
      toast.error(humanizeError(e && e.message ? e.message : String(e)));
      setBusy(false);
    }
  }
  return (
    <div className="term-adopt" role="status">
      <span>{terminalCliLabel(cli)} is running in this shell.</span>
      <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={adopt}>{busy ? "Making agent…" : "Make agent"}</button>
    </div>
  );
}
