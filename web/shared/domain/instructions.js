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
