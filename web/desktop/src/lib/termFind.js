// Find in a terminal pane — the parts that hold no browser.
// Study: docs/benchmarks/2026-09-07-terminal-context-menu.md (phase 2).

// What the counter says. An empty query says nothing at all rather than
// "0 results" for something nobody asked yet; a query with no match says so
// in words, because "0/0" reads like a bug. `index` is xterm's own 0-based
// active match, and it is -1 while the addon has matches but no active one
// (the search ran, the viewport has not landed on a match yet) — then only
// the total is honest.
export function findLabel(query, count, index) {
  if (!query) return "";
  if (!count) return "No results";
  if (index < 0) return String(count);
  return index + 1 + "/" + count;
}

// The three keys the field owns. Everything else belongs to the input.
export function findKey(ev) {
  if (!ev) return null;
  if (ev.key === "Escape") return "close";
  if (ev.key === "Enter") return ev.shiftKey ? "prev" : "next";
  return null;
}

// VS Code's terminal find colours, which are readable over both terminal
// themes we ship (its own light and dark themes use this one pair). The
// addon requires the two overview-ruler entries, so they are given even
// though `overviewRuler` is one pixel wide here (termTheme.js).
export const FIND_DECORATIONS = Object.freeze({
  matchBackground: "#ea5c0055",
  activeMatchBackground: "#f38518",
  matchOverviewRuler: "#d186167e",
  activeMatchColorOverviewRuler: "#a0a0a0cc",
});

// Match case, whole word and regular expression, remembered for the life of
// the page — the same span VS Code's find widget remembers them for.
// Deliberately not persisted: a mode that outlives a reload turns the next
// plain search into a mystery ("No results" for text that is plainly on the
// screen). The names are the addon's own ISearchOptions, so the object is
// handed to it as it stands.
export const FIND_FLAGS = ["caseSensitive", "wholeWord", "regex"];

const mode = { caseSensitive: false, wholeWord: false, regex: false };

export function findMode() {
  return { ...mode };
}

export function setFindMode(patch) {
  for (const flag of FIND_FLAGS) {
    if (patch && typeof patch[flag] === "boolean") mode[flag] = patch[flag];
  }
  return findMode();
}

// True when two modes would search differently — the addon caches its match
// list per term and re-scans only when told (see TermFindBar).
export function modeChanged(a, b) {
  return FIND_FLAGS.some((flag) => !!(a && a[flag]) !== !!(b && b[flag]));
}

// A half-typed pattern is the normal state of a regex field, and the addon
// throws on it. Say "Invalid pattern" — a real answer about the query —
// rather than letting it land in the counter's failure state.
export function findProblem(query, regex) {
  if (!regex || !query) return "";
  try {
    new RegExp(query);
    return "";
  } catch {
    return "Invalid pattern";
  }
}
