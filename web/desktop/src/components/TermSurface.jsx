import { useState } from "react";
import ShellTerm from "./ShellTerm.jsx";
import { ChecklistLine } from "./WorkspaceRows.jsx";
import { bumpTermFontSize } from "@picode/shared/domain/termTheme.js";
import { api } from "@picode/shared/client/api.js";

const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

export default function TermSurface({ term, error, hidden, onOpenFile, cwdKind, checklist }) {
  const [resuming, setResuming] = useState(false);
  const [resumeError, setResumeError] = useState("");
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
      {/* The agent's plan above the pane (ADR-0055): one live line, the same
          projection the sidebar cards use. Nothing known → nothing shown. */}
      {checklist ? <div className="term-check"><ChecklistLine line={checklist} /></div> : null}
      {error ? (
        <p className="file-pane-msg">
          {error}{" "}
          <a href="#/system">Open System</a>
        </p>
      ) : term?.launchCli && !term.running ? (
        <div className="file-pane-msg" role="status">
          <span>This CLI terminal is stopped. </span>
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
        <ShellTerm agentId={term.id} session={term.session} active={!hidden} cwd={term.cwd} cwdKind={cwdKind} onOpenFile={onOpenFile} />
      )}
    </section>
  );
}
