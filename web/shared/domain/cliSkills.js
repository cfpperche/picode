// The Skills pane's browser-side contract (ADR-0196, docs/plans/skills.md):
// the address, the words for each status, and the filters. Pure, tested in
// cliSkills.test.js; both apps render from it.
import { SKILL_WORDS, readScope, writeScope } from "./scopes.js";
import { terminalCliLabel } from "./terminalCli.js";

// The CLIs with a skills declaration, in the server's order
// (internal/skills/specs.go; TestSkillsJSListMatchesTheDeclarations pins it).
export const SKILLS_CLIS = ["pi", "omp", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy"];

export function supportsCliSkills(cli) {
  return SKILLS_CLIS.includes(cli);
}

export function cliSkillsHash(cli, { workspaceId = "", agentId = "", scope = "" } = {}) {
  const q = new URLSearchParams();
  if (workspaceId) q.set("workspaceId", workspaceId);
  if (agentId) q.set("agentId", agentId);
  writeScope(q, scope);
  const qs = q.toString();
  return "#/clis/" + encodeURIComponent(cli) + "/skills" + (qs ? "?" + qs : "");
}

// The query half of the pane's address, as cliLocation reads it.
export function cliSkillsQuery(params) {
  const read = readScope(params, SKILL_WORDS);
  return {
    workspaceId: params.get("workspaceId") || "",
    agentId: params.get("agentId") || "",
    scope: read.value || "machine",
    ...(read.kind ? { scopeKind: read.kind } : {}),
    invalid: read.invalid,
  };
}

export function skillsReportPath(cli, workspaceId = "", agentId = "") {
  const q = new URLSearchParams({ cli });
  if (workspaceId) q.set("workspace", workspaceId);
  if (workspaceId && agentId) q.set("agent", agentId);
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
      return { label: "Shadowed", tone: "muted", detail: row.shadowedBy === AGENT_ROOT ? "This agent's own copy wins" : "The copy in " + row.shadowedBy + " wins" };
    case "missing":
      return { label: "Missing", tone: "warn", detail: "Its copy is gone from PiCode's cache, so the next start leaves it out. Add it again" };
    case "needs-trust":
      return { label: "Needs trust", tone: "warn", detail: name + " skips this workspace's skills until you trust the folder" };
    case "if-trusted":
      return { label: "If trusted", tone: "info", detail: name + " loads it once the folder is trusted in " + name };
    case "invalid":
      return { label: "Not loaded", tone: "warn", detail: (row.problems || []).join("; ") || "SKILL.md is not a valid skill" };
    case "disabled":
      return { label: "Off", tone: "muted", detail: "Switched off in " + name + "'s own settings, so it does not load" };
    default:
      return { label: row.status || "Unknown", tone: "muted" };
  }
}

// A source as a short label: owner/repo, a site, or a folder's name.
export function sourceLabel(src = {}) {
  if (src.kind === "github") return src.owner + "/" + src.repo + (src.sub ? "/" + src.sub : "");
  if (src.kind === "well-known") return (src.origin || "").replace(/^https:\/\//, "");
  return (src.dir || src.input || "").replace(/\/+$/, "").split("/").pop();
}

// Where a skill came from, as one short phrase: the installer's source, or
// the folder when no lock names it.
export function skillOrigin(row) {
  const p = row.provenance;
  if (!p) return { label: "Added by hand", title: "No installer's lock names this folder" };
  if (p.installer === "picode") {
    const src = (p.source || "").replace(/\/+$/, "");
    const label = src.startsWith("/") || src.startsWith("~") ? src.split("/").pop() : src;
    return { label: label || "PiCode", title: "Added to this agent by PiCode from " + (p.source || "a source") };
  }
  const via = p.installer === "hermes" ? "Hermes hub" : "skills CLI";
  // A local source is a long path whose useful part is its last folder.
  const local = p.sourceType === "local" && p.source ? p.source.replace(/\/+$/, "").split("/").pop() : "";
  const label = local || p.source || via;
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

// The agent's own list, as the reader names its folder (internal/skills).
export const AGENT_ROOT = "This agent only";

// The chips a report offers: Global, the workspace by its name, and the
// agent by its name when the CLI takes an agent's own skills at launch (Pi,
// Omp, Claude Code — ADR-0196 slice 4).
export function skillScopes(report, workspaceName = "") {
  const out = [{ id: "machine", label: "Global" }];
  if (report?.workspacePath) out.push({ id: "workspace", label: workspaceName || "This workspace" });
  if (report?.agent) out.push({ id: "agent", label: report.agent.name || "This agent" });
  return out;
}

// Under the agent chip: what that agent loads — its own skills and the
// folders' (none of the folders' when it runs isolated), and how a skill
// added here reaches it.
export function agentScopeLine(report, cli) {
  const a = report?.agent;
  if (!a) return "";
  const name = terminalCliLabel(cli);
  const own = cli === "claude-code" ? " Its own skills appear as /picode-agent:<name>." : "";
  if (a.isolated) return a.name + " runs isolated: " + name + " loads only its own skills and its packages' skills, none from these folders." + own;
  return "What " + a.name + " loads. A skill you add here is for " + a.name + " alone and loads at its next start." + own;
}

// The CLIs that take an agent's own skills at launch (internal/skills specs
// AgentScope; TestSkillsAgentCLIsMatch pins it).
export const AGENT_SKILL_CLIS = ["pi", "omp", "claude-code"];

// Opened for an agent whose CLI has no way to take skills of its own at
// launch: one line saying so, instead of a missing chip nobody explains.
export function noAgentScopeLine(report, cli, agentId = "") {
  if (!agentId || report?.agent || AGENT_SKILL_CLIS.includes(cli)) return "";
  return terminalCliLabel(cli) + " cannot take skills for one agent when it starts; skills here reach every " + terminalCliLabel(cli) + " agent. Pi, Omp and Claude Code can.";
}

export function visibleSkills(rows = [], { scope = "machine", text = "", agent = null } = {}) {
  const needle = text.trim().toLowerCase();
  const inScope = (r) => {
    if (r.scope === "agent") return scope === "agent";
    if (scope === "agent") return !agent?.isolated && (r.status === "loaded" || r.status === "if-trusted");
    return r.scope === scope;
  };
  return rows
    .filter(inScope)
    .filter((r) => !needle || (r.name + " " + (r.description || "") + " " + (r.provenance?.source || "")).toLowerCase().includes(needle))
    .slice()
    .sort((a, b) => (RANK[a.status] ?? 9) - (RANK[b.status] ?? 9) || a.name.localeCompare(b.name) || a.root.localeCompare(b.root));
}

// The counts the header states, so an empty filter is never mistaken for an
// empty installation.
export function skillsSummary(rows = []) {
  const loaded = rows.filter((r) => r.status === "loaded" || r.status === "if-trusted").length;
  const shadowed = rows.filter((r) => r.status === "shadowed").length;
  const off = rows.filter((r) => r.status === "disabled").length;
  const tokens = rows.filter((r) => r.status === "loaded").reduce((n, r) => n + (r.tokens || 0), 0);
  return { total: rows.length, loaded, shadowed, off, tokens };
}

// Per-skill switches (ADR-0196 slice 3). A row carries `enabled` only where
// the CLI keeps a switch of its own; Pi and Antigravity have none, and say so
// once instead of drawing a control that cannot work.
export function hasSkillSwitch(row) {
  return typeof row?.enabled === "boolean";
}

export function noSwitchLine(report, cli) {
  if (!report || report.toggle || !(report.rows || []).length) return "";
  return terminalCliLabel(cli) + " has no switch for one skill: remove it to stop it loading.";
}

// The body for POST /api/skills/toggle: the row as the report named it.
export function skillToggleBody(cli, row, workspaceId, enabled) {
  const body = { cli, scope: row.scope, dir: row.dir, enabled };
  if (workspaceId) body.workspace = workspaceId;
  return body;
}

// The toast after a flip: what the CLI does now, and why when another layer
// or a machine-wide switch decides.
export function toggledLine(res) {
  const now = res.name + (res.enabled ? " is on." : " is off.");
  return res.note ? now + " " + res.note + "." : now;
}

// The switch applied optimistically, before the server answers.
export function withSkillEnabled(report, dir, scope, enabled) {
  if (!report) return report;
  const rows = (report.rows || []).map((r) => {
    if (r.dir !== dir || r.scope !== scope) return r;
    const status = enabled ? (r.status === "disabled" ? "loaded" : r.status) : (r.status === "loaded" ? "disabled" : r.status);
    return { ...r, enabled, status };
  });
  return { ...report, rows };
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

// --- slice 2: install, remove, update (ADR-0196) ---------------------------

// Rows PiCode can remove: the canonical folder an install writes. A skill in
// a CLI's own folder (~/.claude/skills, .grok/skills…) is that CLI's to manage.
export function canRemoveSkill(row) {
  return row.scope === "agent" || row.root === ".agents/skills" || row.root === "~/.agents/skills";
}

// Where an install lands, in one sentence the dialog shows before consent.
export function installLine(scope, workspaceName = "", agentName = "") {
  if (scope === "agent") {
    return "Keeps a copy in PiCode for " + (agentName || "this agent") + " alone, loaded at its next start. Nothing is written to the workspace or your home folder.";
  }
  if (scope === "workspace") {
    return "Installs into " + (workspaceName || "this workspace") + "'s .agents/skills folder, where most agent CLIs look, and links it for Claude Code.";
  }
  return "Installs into ~/.agents/skills on this computer, linked for Claude Code, Hermes and Antigravity where they are installed.";
}

// The answers a refusal offers: the code comes from the server (409).
export function conflictChoices(code) {
  if (code === "exists") {
    return [
      { label: "Keep mine", body: { adopt: true } },
      { label: "Replace it", body: { replace: true }, danger: true },
    ];
  }
  if (code === "update") return [{ label: "Use this version", body: { replace: true } }];
  return [];
}

export function criticalFindings(c) {
  return (c?.findings || []).filter((f) => f.severity === "critical");
}

// Why Install cannot run yet, in words (a disabled button never explains
// itself — .pi/skills/uiux-review).
export function installBlocker(c, accepted) {
  if (!c) return "Pick a skill.";
  if ((c.problems || []).length) return "This skill breaks the format: " + c.problems.join("; ") + ".";
  if (criticalFindings(c).length && !accepted) return "Read the findings, then confirm you reviewed the files.";
  return "";
}

export function sizeLabel(n = 0) {
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(n < 10240 ? 1 : 0) + " KB";
  return (n / 1024 / 1024).toFixed(1) + " MB";
}

// Update rows by "scope:name", so a report row finds its check.
export function updatesByKey(rows = []) {
  const m = {};
  for (const r of rows) m[r.scope + ":" + r.name] = r;
  return m;
}

export function updatesSummary(rows = []) {
  if (!rows.length) return "Nothing here was installed from a source PiCode can check.";
  const behind = rows.filter((r) => r.status === "behind").length;
  const failed = rows.filter((r) => r.status === "unreachable").length;
  const parts = [];
  const names = rows.filter((r) => r.status === "behind").map((r) => r.name);
  parts.push(behind ? "Update available for " + names.slice(0, 3).join(", ") + (names.length > 3 ? " and " + (names.length - 3) + " more" : "") : "Everything checked is current");
  if (failed) parts.push(failed + " could not be checked");
  return parts.join(" · ") + ".";
}

// The body for DELETE /api/skills and POST /api/skills/update.
export function skillTargetBody(row, workspaceId, agentId = "") {
  const body = { name: row.name, scope: row.scope };
  if (row.scope === "workspace") body.workspace = workspaceId;
  if (row.scope === "agent") body.agent = agentId;
  return body;
}

// The toast after an install, by where it landed.
export function installedLine(res, scope, agentName = "") {
  if (res.status === "already") return res.name + (scope === "agent" ? " is already on " + (agentName || "this agent") + "." : " is already installed.");
  if (res.status === "adopted") return "Recorded where " + res.name + " came from.";
  if (scope === "agent") return res.name + " added to " + (agentName || "this agent") + ". It loads at the next start.";
  return res.name + " installed.";
}

// The question before a remove, by scope.
export function removeQuestion(row, { workspaceName = "", agentName = "" } = {}) {
  if (row.scope === "agent") return "Remove " + row.name + " from " + (agentName || "this agent") + "? It stops loading at the next start.";
  return "Remove " + row.name + " from " + (row.scope === "workspace" ? (workspaceName || "this workspace") : "this computer") + "? Every CLI that reads this folder loses it.";
}

// --- slice 5: the Marketplace (ADR-0196) ------------------------------------

export function skillCatalogPath(q = "") {
  const query = new URLSearchParams();
  if (q.trim()) query.set("q", q.trim());
  const qs = query.toString();
  return "/api/skills/catalog" + (qs ? "?" + qs : "");
}

// Where a card comes from, in one short phrase: a built-in source, one the
// person added, or skills.sh with its own install count.
export function catalogOrigin(item = {}) {
  if (item.origin === "skills.sh") {
    const n = item.installs || 0;
    const count = n >= 1000 ? (n / 1000).toFixed(n >= 10000 ? 0 : 1) + "k" : String(n);
    return { label: (item.source ? item.source + " · " : "") + "skills.sh · " + count + " installs", title: "Listed by skills.sh (Vercel); its audits are on the skill's page there" };
  }
  if (item.origin === "yours") return { label: item.source, title: "From a source you added" };
  return { label: item.source, title: "From a built-in source" };
}

// A card is Installed when this CLI already has that skill from that source
// (the installer's lock names it). Another skill of the same name from
// elsewhere is not this one: the card still offers Install, and the dialog
// asks before replacing it.
export function installedSkillKeys(report) {
  const keys = new Set();
  for (const r of report?.rows || []) {
    if (r.status === "invalid") continue;
    const src = (r.provenance?.source || "").toLowerCase();
    if (src) keys.add(r.name + "|" + src);
  }
  return keys;
}

export function catalogInstalled(keys, item = {}) {
  const src = (item.source || "").toLowerCase();
  if (keys.has(item.name + "|" + src)) return true;
  // A lock records a GitHub source as owner/repo, and a sub-folder install
  // as owner/repo too; a site as its host.
  return [...keys].some((k) => k.startsWith(item.name + "|") && src && k.slice(item.name.length + 1).startsWith(src));
}

// One line per source under the list.
export function sourceStateLine(s = {}) {
  if (s.reading) return "Reading…";
  if (s.error && !s.count) return s.error;
  const n = s.count === 1 ? "1 skill" : (s.count || 0) + " skills";
  return s.error ? n + " · last read failed: " + s.error : n;
}

export const SKILLSSH_NOTE = "What you type is sent to skills.sh (Vercel), which lists far more skills than the sources above.";

// The empty Marketplace, by why it is empty.
export function catalogEmptyLine(q, reading) {
  if (reading) return "Reading the sources. Their skills appear here as they arrive.";
  if (q.trim()) return "No skill matches “" + q.trim() + "”.";
  return "The sources list no skills yet.";
}

// --- slice 6: the lifecycle (ADR-0196) --------------------------------------

// An agent's own skill can move into the agent's workspace once it works.
export function canPromoteSkill(row, report) {
  return row?.scope === "agent" && row.status !== "missing" && !!report?.workspacePath;
}

// Under an agent's own skill: where it is in its life, and the next step.
export function trialLine(row, { agentName = "", workspaceName = "" } = {}) {
  if (row?.scope !== "agent") return "";
  const a = agentName || "this agent";
  return "Trying in " + a + " only. Outcomes compares its runs with and without it; when it works, promote it to " + (workspaceName || "the workspace") + ".";
}

export function promoteQuestion(row, { agentName = "", workspaceName = "" } = {}) {
  const ws = workspaceName || "this workspace";
  return "Promote " + row.name + " to " + ws + "? Every agent CLI in " + ws + " then loads it, and it leaves " + (agentName || "this agent") + "'s own list.";
}

export function promotedLine(res, workspaceName = "") {
  const ws = workspaceName || "the workspace";
  if (res?.status === "already") return res.name + " was already in " + ws + "; it left the agent's own list.";
  return res.name + " is now in " + ws + ": every agent CLI there loads it.";
}

// A promote refusal the person can answer, and what the retry sends.
export function promoteRetry(code) {
  if (code === "exists" || code === "update") return { replace: true };
  if (code === "critical") return { acceptCritical: true };
  return null;
}

// The confirm button's words, by verb and what the retry sends.
export function askLabel(ask) {
  if (!ask) return "";
  if (ask.verb === "remove") return ask.extra?.confirm ? "Remove anyway" : "Remove";
  if (ask.verb === "promote") return ask.extra?.replace ? "Replace it" : ask.extra?.acceptCritical ? "Promote anyway" : "Promote";
  return "Update anyway";
}

// The agent row's warning when some of its own skills lost their cached
// copy (the launch leaves them out): one line and where to fix it.
export function agentSkillsMissing(agent) {
  const gone = ((agent && agent.skills) || []).filter((s) => s && s.missing).map((s) => s.name);
  if (!gone.length) return null;
  const cli = (agent && agent.cli) || "pi";
  const workspaceId = agent.workspaceId && agent.workspaceId !== "ws_free" ? agent.workspaceId : "";
  return {
    text: gone.length === 1 ? gone[0] + " skill missing" : gone.length + " skills missing",
    title: "PiCode's copy of " + gone.join(", ") + " is gone, so the next start leaves " + (gone.length === 1 ? "it" : "them") + " out. Add " + (gone.length === 1 ? "it" : "them") + " again.",
    href: cliSkillsHash(cli, { workspaceId, agentId: agent.id, scope: "agent" }),
  };
}
