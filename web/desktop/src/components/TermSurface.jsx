import ShellTerm from "./ShellTerm.jsx";
import { ChecklistLine } from "./WorkspaceRows.jsx";
import { bumpTermFontSize } from "@picode/shared/domain/termTheme.js";

export default function TermSurface({ term, error, hidden, onOpenFile, cwdKind, checklist }) {
  if (!term && !error) return null;
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
        <p className="file-pane-msg">This CLI terminal is stopped. <a href="#/clis/terminals">Start from Agent CLIs</a></p>
      ) : (
        <ShellTerm agentId={term.id} session={term.session} active={!hidden} cwd={term.cwd} cwdKind={cwdKind} onOpenFile={onOpenFile} />
      )}
    </section>
  );
}
