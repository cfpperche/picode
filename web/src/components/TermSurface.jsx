import ShellTerm from "./ShellTerm.jsx";
import { bumpTermFontSize } from "../lib/termTheme.js";

export default function TermSurface({ term, error, onOpenFile }) {
  if (!term && !error) return null;
  function onKey(e) {
    if (!(e.ctrlKey || e.metaKey) || e.shiftKey) return;
    if (e.key === "=" || e.key === "+") { e.preventDefault(); bumpTermFontSize(1); }
    else if (e.key === "-") { e.preventDefault(); bumpTermFontSize(-1); }
    else if (e.key === "0") { e.preventDefault(); bumpTermFontSize(0); }
  }
  return (
    <section className="term-surface" aria-label={term ? term.name : "Terminal"} onKeyDown={onKey}>
      {error ? (
        <p className="file-pane-msg">
          {error}{" "}
          <a href="#/system">Open System</a>
        </p>
      ) : (
        <ShellTerm agentId={term.id} session={term.session} active cwd={term.cwd} onOpenFile={onOpenFile} />
      )}
    </section>
  );
}
