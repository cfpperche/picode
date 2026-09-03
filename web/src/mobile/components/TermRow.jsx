import { IconTerminal, IconChevronRight, IconTrash } from "../../components/Icons.jsx";
import { termLine } from "../../lib/repoLine.js";

// One terminal, same 56px row as an agent: name, where it is (live cwd
// and branch), a live dot, and Remove on the right edge.
export default function TermRow({ term, onOpen, onRemove, busy }) {
  const line = termLine(term);
  return (
    <li className={"m-row m-term-row" + (term.running ? " is-live" : "")}>
      <button type="button" className="m-row-main" onClick={() => onOpen(term)}>
        <span className="m-row-face"><IconTerminal size={18} /></span>
        <span className="m-row-text">
          <span className="m-row-title">{term.name || "Terminal"}</span>
          <span className="m-row-sub">{line.text}</span>
        </span>
        {term.state === "needs-you" ? <span className="m-state is-waiting">Needs you</span>
          : term.state === "working" ? <span className="m-state is-working">Working</span>
          : term.running ? <span className="m-state is-working">Live</span> : null}
        <IconChevronRight size={16} className="m-row-chev" />
      </button>
      <button type="button" className="m-row-act is-remove" title="Remove" aria-label={"Remove " + (term.name || "terminal")} disabled={busy} onClick={() => onRemove(term)}><IconTrash size={17} /></button>
    </li>
  );
}
