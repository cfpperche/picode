// The checks card (slice 4 of docs/benchmarks/2026-09-22-omp-helpers-placement.md):
// what a CLI resolves in one folder, where each value comes from, and the
// problems PiCode measured. `internal/clidoctor` is the server's half;
// TestJSListMatchesTheChecks keeps the two lists equal.

export const DOCTOR_CLIS = ["omp"];

export const supportsCliDoctor = (id) => DOCTOR_CLIS.includes(id);

// Where a value comes from, in the pane's own words. A value neither file sets
// is the CLI's default or comes from a source PiCode does not read (an
// overlay, a foreign settings file the CLI also merges), so it is not called
// "default" — that would be a claim PiCode cannot check.
export function sourceLabel(source) {
  if (source === "project") return "This workspace";
  if (source === "user") return "Global";
  return "Not in either file";
}

// One line for any value the listing holds. Lists and maps print as compact
// JSON, long values are cut with an ellipsis the tooltip completes.
export function formatValue(value, max = 80) {
  let text;
  // omp lists a key it knows but holds no value for as null; say so in
  // words, so it is not mistaken for a hidden credential.
  if (value === null || value === undefined) text = "not set";
  else if (typeof value === "string") text = value === "" ? '""' : value;
  else text = JSON.stringify(value);
  return text.length > max ? text.slice(0, max - 1) + "…" : text;
}

export function filterEntries(entries = [], { query = "", onlySet = false } = {}) {
  const q = query.trim().toLowerCase();
  return entries.filter((e) => {
    if (onlySet && !e.source) return false;
    if (!q) return true;
    // Key and value only: a match on the description the row does not show
    // listed keys with no visible reason (visual review, 2026-09-22).
    return e.key.toLowerCase().includes(q) || (!e.redacted && formatValue(e.value, 2000).toLowerCase().includes(q));
  });
}

// Warnings first, then information, each kept in the server's order.
export function sortFindings(findings = []) {
  const rank = (f) => (f.severity === "warn" ? 0 : 1);
  return findings.map((f, i) => [f, i]).sort((a, b) => rank(a[0]) - rank(b[0]) || a[1] - b[1]).map(([f]) => f);
}
