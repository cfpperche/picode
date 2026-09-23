// The Skills pane's browser-side contract (ADR-0196, docs/plans/skills.md):
// the address, the words for each status, and the filters. Pure, tested in
// cliSkills.test.js; both apps render from it.
import { terminalCliLabel } from "./terminalCli.js";

// The CLIs with a skills declaration, in the server's order
// (internal/skills/specs.go; TestSkillsJSListMatchesTheDeclarations pins it).
export const SKILLS_CLIS = ["pi", "omp", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy"];

export function supportsCliSkills(cli) {
  return SKILLS_CLIS.includes(cli);
}

export const SKILL_SCOPES = ["all", "workspace", "machine"];

export function cliSkillsHash(cli, { workspaceId = "", scope = "" } = {}) {
  const q = new URLSearchParams();
  if (workspaceId) q.set("workspaceId", workspaceId);
  if (scope && scope !== "all" && SKILL_SCOPES.includes(scope)) q.set("scope", scope);
  const qs = q.toString();
  return "#/clis/" + encodeURIComponent(cli) + "/skills" + (qs ? "?" + qs : "");
}

// The query half of the pane's address, as cliLocation reads it.
export function cliSkillsQuery(params) {
  const scope = params.get("scope") || "all";
  return {
    workspaceId: params.get("workspaceId") || "",
    scope: SKILL_SCOPES.includes(scope) ? scope : "all",
    invalid: !SKILL_SCOPES.includes(scope),
  };
}

export function skillsReportPath(cli, workspaceId = "") {
  const q = new URLSearchParams({ cli });
  if (workspaceId) q.set("workspace", workspaceId);
  return "/api/skills/report?" + q;
}

// What the CLI does with one folder, in the reader's words. `cli` names the
// CLI whose pane this is, so the trust line says whose trust it needs.
export function skillStatus(row, cli) {
  const name = terminalCliLabel(cli);
  switch (row.status) {
    case "loaded":
      return { label: "Loaded", tone: "ok" };
    case "shadowed":
      return { label: "Shadowed", tone: "muted", detail: "The copy in " + row.shadowedBy + " wins" };
    case "needs-trust":
      return { label: "Needs trust", tone: "warn", detail: name + " skips this workspace's skills until you trust the folder" };
    case "if-trusted":
      return { label: "If trusted", tone: "info", detail: name + " loads it once the folder is trusted in " + name };
    case "invalid":
      return { label: "Not loaded", tone: "warn", detail: (row.problems || []).join("; ") || "SKILL.md is not a valid skill" };
    default:
      return { label: row.status || "Unknown", tone: "muted" };
  }
}

// Where a skill came from, as one short phrase: the installer's source, or
// the folder when no lock names it.
export function skillOrigin(row) {
  const p = row.provenance;
  if (!p) return { label: "Added by hand", title: "No installer's lock names this folder" };
  const via = p.installer === "hermes" ? "Hermes hub" : "skills CLI";
  const label = p.source || via;
  const title = "Recorded by the " + via + " in " + p.lock + (p.modified ? " — changed since it was installed" : "");
  return { label, title, modified: !!p.modified };
}

export function tokensLabel(n) {
  if (!n) return "";
  return "~" + (n >= 1000 ? (n / 1000).toFixed(1) + "k" : String(n)) + " tokens";
}

export function alsoLoadedLine(row) {
  const list = (row.alsoLoadedBy || []).map(terminalCliLabel);
  if (!list.length) return "";
  return "Also read by " + list.join(", ");
}

// Rows the pane shows for a scope and a filter; loaded rows first, then the
// rest, each group by name.
const RANK = { loaded: 0, "if-trusted": 1, "needs-trust": 2, invalid: 3, shadowed: 4 };

export function visibleSkills(rows = [], { scope = "all", text = "" } = {}) {
  const needle = text.trim().toLowerCase();
  return rows
    .filter((r) => scope === "all" || r.scope === scope)
    .filter((r) => !needle || (r.name + " " + (r.description || "") + " " + (r.provenance?.source || "")).toLowerCase().includes(needle))
    .slice()
    .sort((a, b) => (RANK[a.status] ?? 9) - (RANK[b.status] ?? 9) || a.name.localeCompare(b.name) || a.root.localeCompare(b.root));
}

// The counts the header states, so an empty filter is never mistaken for an
// empty installation.
export function skillsSummary(rows = []) {
  const loaded = rows.filter((r) => r.status === "loaded" || r.status === "if-trusted").length;
  const shadowed = rows.filter((r) => r.status === "shadowed").length;
  const tokens = rows.filter((r) => r.status === "loaded").reduce((n, r) => n + (r.tokens || 0), 0);
  return { total: rows.length, loaded, shadowed, tokens };
}

// The trust line: shown when the workspace has skills this CLI will not load
// until the folder is trusted in it.
export function trustLine(report, rows = []) {
  const t = report?.trust;
  if (!t?.needed) return null;
  const gated = rows.filter((r) => r.scope === "workspace" && (r.status === "needs-trust" || r.status === "if-trusted"));
  if (!gated.length) return null;
  return { text: t.note || "", command: t.command || "", known: t.trusted !== undefined && t.trusted !== null, trusted: !!t.trusted };
}

// One line for an empty pane, by why it is empty.
export function skillsEmptyLine(cli, { workspaceName = "", hasWorkspace = false } = {}) {
  const name = terminalCliLabel(cli);
  if (!hasWorkspace) return name + " has no skills in the folders it reads on this machine.";
  return name + " has no skills in the folders it reads here or in " + (workspaceName || "this workspace") + ".";
}
