// The Instructions tab's pure half: how GET /api/workspaces/{id}/instructions
// (internal/cliinstructions) becomes rows, columns and one line for the
// Create dialog. No DOM, no fetch — `node --test` covers every branch.

export const STATUS_LABEL = {
  reads: "Reads",
  "on-demand": "On demand",
  shadowed: "Skipped",
  "not-read": "—",
  untrusted: "Needs trust",
  unknown: "Unknown",
};

// What a cell says when a person hovers it: the verdict in words, the
// winner for a skipped file, and the limit it crosses.
export function cellTitle(cell) {
  if (!cell) return "";
  const parts = [cell.why || ""];
  if (cell.status === "shadowed" && cell.by) parts.unshift(cell.by + " wins.");
  if (cell.status === "on-demand") parts.unshift("Read once the agent works in that folder.");
  if (cell.cut) parts.push(cell.cut + ".");
  return parts.filter(Boolean).join(" ");
}

const GROUPS = [
  { id: "project", title: "This workspace" },
  { id: "above", title: "Folders above this workspace" },
  { id: "personal", title: "Personal files on this machine" },
];

// groupFiles splits the rows by where they live, dropping empty groups.
export function groupFiles(files = []) {
  return GROUPS.map((g) => ({ ...g, files: files.filter((f) => f && f.scope === g.id) })).filter((g) => g.files.length > 0);
}

// visibleClis is the columns: the installed CLIs, or all nine when asked
// (and when none is installed, since an empty table explains nothing).
export function visibleClis(clis = [], showAll = false) {
  const installed = clis.filter((c) => c && c.installed);
  if (showAll || installed.length === 0) return clis;
  return installed;
}

// A row matters to a column set when at least one of those CLIs does
// something with the file; a row every visible CLI ignores is noise unless
// the reader asked for everything.
export function rowMatters(file, clis) {
  return clis.some((c) => {
    const st = file && file.cells && file.cells[c.id] && file.cells[c.id].status;
    return st && st !== "not-read";
  });
}

// shortPath keeps a long path's last folders, so a row outside the workspace
// (an absolute path above it) does not push the columns off the card.
export function shortPath(path = "", max = 36) {
  if (path.length <= max) return path;
  const parts = path.split("/");
  let out = parts.pop();
  while (parts.length && (parts[parts.length - 1] + "/" + out).length <= max - 2) out = parts.pop() + "/" + out;
  return "…/" + out;
}

export function sizeLabel(bytes = 0) {
  if (bytes < 1024) return bytes + " B";
  const kb = bytes / 1024;
  return (kb < 10 ? kb.toFixed(1) : Math.round(kb)) + " KB";
}

// createLine is the one line the Create dialog shows for a workspace and a
// CLI: which project files it reads and what it leaves out.
export function createLine(report, cli) {
  if (!report || !Array.isArray(report.files)) return "";
  const col = (report.clis || []).find((c) => c.id === cli);
  if (!col) return "";
  const project = report.files.filter((f) => f.scope === "project" && !f.folder);
  const cell = (f) => (f.cells && f.cells[cli]) || {};
  const reads = project.filter((f) => cell(f).status === "reads").map((f) => f.path);
  const skipped = project.filter((f) => cell(f).status === "shadowed").map((f) => f.path);
  if (project.some((f) => cell(f).status === "untrusted")) {
    return col.name + " ignores this folder's instructions until you trust it in " + col.name + ".";
  }
  if (reads.length === 0 && skipped.length === 0) {
    return project.length === 0 ? "No instruction file in this workspace." : col.name + " reads none of this workspace's instruction files.";
  }
  let line = col.name + " reads " + (reads.length ? joinNames(reads) : "none of them");
  if (skipped.length) line += "; " + joinNames(skipped) + (skipped.length === 1 ? " is" : " are") + " left out";
  return line + ".";
}

function joinNames(names) {
  if (names.length <= 1) return names.join("");
  return names.slice(0, -1).join(", ") + " and " + names[names.length - 1];
}

// lineDiff is the change a fix makes, as lines marked " ", "-" or "+".
// Fixes are small (a line added on top, a sentence replaced, a line appended,
// a new file), so the common head and tail are kept as context — at most
// `context` lines each side — and the middle is what changes.
export function lineDiff(before = "", after = "", context = 2) {
  const split = (t) => (t === "" ? [] : t.replace(/\n$/, "").split("\n"));
  const a = split(before);
  const b = split(after);
  let head = 0;
  while (head < a.length && head < b.length && a[head] === b[head]) head++;
  let tail = 0;
  while (tail < a.length - head && tail < b.length - head && a[a.length - 1 - tail] === b[b.length - 1 - tail]) tail++;
  const out = [];
  if (head > context) out.push({ kind: "…", text: head - context + " unchanged line" + (head - context === 1 ? "" : "s") });
  for (const t of a.slice(Math.max(0, head - context), head)) out.push({ kind: " ", text: t });
  for (const t of a.slice(head, a.length - tail)) out.push({ kind: "-", text: t });
  for (const t of b.slice(head, b.length - tail)) out.push({ kind: "+", text: t });
  const kept = a.slice(a.length - tail);
  for (const t of kept.slice(0, context)) out.push({ kind: " ", text: t });
  if (kept.length > context) out.push({ kind: "…", text: kept.length - context + " unchanged line" + (kept.length - context === 1 ? "" : "s") });
  return out;
}
