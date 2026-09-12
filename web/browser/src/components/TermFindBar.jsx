import { useEffect, useRef, useState } from "react";
import { IconCase, IconChevronDown, IconChevronUp, IconRegex, IconWholeWord, IconX } from "./Icons.jsx";
import { terms } from "../lib/terms.js";
import { findKey, findLabel, findMode, findProblem, modeChanged, setFindMode, FIND_DECORATIONS } from "../lib/termFind.js";

const DEBOUNCE_MS = 120;

// Find inside the pane (@xterm/addon-search), floating over the terminal
// rather than sitting in the layout: a search must not resize the pane, which
// would send tmux a SIGWINCH and make the guest TUI redraw for nothing.
export default function TermFindBar({ termId, onClose }) {
  const inputRef = useRef(null);
  const [query, setQuery] = useState("");
  const [res, setRes] = useState({ index: -1, count: 0 });
  const [broken, setBroken] = useState(false);
  const [mode, setMode] = useState(findMode);
  const ranWith = useRef(findMode());
  const problem = findProblem(query, mode.regex);

  const addon = () => {
    const entry = terms.get("sh:" + termId);
    return (entry && entry.search) || null;
  };

  useEffect(() => {
    if (inputRef.current) inputRef.current.focus();
  }, []);

  // The addon reports what it found, including while typing.
  useEffect(() => {
    const search = addon();
    if (!search) return undefined;
    const sub = search.onDidChangeResults((e) => setRes({ index: e.resultIndex, count: e.resultCount }));
    return () => {
      sub.dispose();
      search.clearDecorations();
    };
  }, [termId]);

  // Incremental while typing: the current match survives each keystroke
  // instead of jumping to the first one on every letter.
  useEffect(() => {
    const search = addon();
    if (!search) return undefined;
    if (!query || problem) {
      search.clearDecorations();
      setRes({ index: -1, count: 0 });
      return undefined;
    }
    const t = setTimeout(() => {
      // The addon caches its match list per term and does not re-scan when
      // only the options change (@xterm/addon-search 0.16.0): toggling case
      // or regex on the same query keeps the old count and highlights.
      // Dropping the decorations drops that cache with them.
      if (modeChanged(ranWith.current, mode)) {
        search.clearDecorations();
        ranWith.current = mode;
      }
      run(search, "next", { incremental: true });
    }, DEBOUNCE_MS);
    return () => clearTimeout(t);
  }, [query, termId, mode.caseSensitive, mode.wholeWord, mode.regex, problem]);

  // The addon reaches xterm's proposed decoration API and throws when a
  // terminal was built without it (termTheme.js). Say so in the counter
  // rather than reporting an honest-looking "No results".
  function run(search, dir, extra) {
    try {
      const go = dir === "prev" ? search.findPrevious : search.findNext;
      go.call(search, query, { ...extra, ...mode, decorations: FIND_DECORATIONS });
      return true;
    } catch {
      setBroken(true);
      return false;
    }
  }

  function toggle(flag) {
    setMode(setFindMode({ [flag]: !mode[flag] }));
    if (inputRef.current) inputRef.current.focus();
  }

  function step(dir) {
    const search = addon();
    if (!search || !query || problem) return;
    run(search, dir);
    if (inputRef.current) inputRef.current.focus();
  }

  function onKeyDown(e) {
    const action = findKey(e);
    if (!action) return;
    e.preventDefault();
    e.stopPropagation();
    if (action === "close") onClose();
    else step(action);
  }

  const label = broken ? "Unavailable" : problem || findLabel(query, res.count, res.index);
  const empty = broken || !!problem || !query || !res.count;

  return (
    <div className="term-find" role="search" data-align-row onKeyDown={onKeyDown}>
      <input
        ref={inputRef}
        type="text"
        className="term-find-input"
        placeholder="Find in terminal"
        aria-label="Find in terminal"
        autoComplete="off"
        spellCheck="false"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <button type="button" className="icon-btn" title="Match case" aria-label="Match case" aria-pressed={mode.caseSensitive} onClick={() => toggle("caseSensitive")}>
        <IconCase />
      </button>
      <button type="button" className="icon-btn" title="Match whole word" aria-label="Match whole word" aria-pressed={mode.wholeWord} onClick={() => toggle("wholeWord")}>
        <IconWholeWord />
      </button>
      <button type="button" className="icon-btn" title="Use a regular expression" aria-label="Use a regular expression" aria-pressed={mode.regex} onClick={() => toggle("regex")}>
        <IconRegex />
      </button>
      <span className="term-find-count" aria-live="polite">{label}</span>
      <button type="button" className="icon-btn" title="Previous match (Shift+Enter)" aria-label="Previous match" disabled={empty} onClick={() => step("prev")}>
        <IconChevronUp size={14} />
      </button>
      <button type="button" className="icon-btn" title="Next match (Enter)" aria-label="Next match" disabled={empty} onClick={() => step("next")}>
        <IconChevronDown size={14} />
      </button>
      <button type="button" className="icon-btn" title="Close (Esc)" aria-label="Close find" onClick={onClose}>
        <IconX size={13} />
      </button>
    </div>
  );
}
