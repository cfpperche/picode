// Copy for Work's full-list empty well. Chrome carries one line + one
// action (docs/benchmarks.md); "free" is ADR-0011 jargon, not UI copy.
const COPY = {
  workspaces: { line: "No workspaces yet.", action: "Add workspace", match: "No matching workspaces." },
  agents: { line: "No agents yet.", action: "New agent", match: "No matching agents." },
  terminals: { line: "No terminals yet.", action: "New terminal", match: "No matching terminals." },
};

export function workEmpty(section, searching) {
  const copy = COPY[section] || COPY.workspaces;
  if (searching) return { line: copy.match, action: "Clear search", page: false };
  return { line: copy.line, action: copy.action, page: true };
}
