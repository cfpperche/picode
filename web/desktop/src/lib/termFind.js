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
