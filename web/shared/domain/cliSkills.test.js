import test from "node:test";
import assert from "node:assert/strict";
import {
  alsoLoadedLine, cliSkillsHash, cliSkillsQuery, skillOrigin, skillStatus, skillsEmptyLine,
  skillsReportPath, skillsSummary, supportsCliSkills, tokensLabel, trustLine, visibleSkills,
} from "./cliSkills.js";
import { cliLocation, cliPanes } from "./cliLaunch.js";

const row = (over = {}) => ({ name: "pdf", description: "PDF tools", scope: "machine", root: "~/.agents/skills", status: "loaded", tokens: 12, ...over });

test("the address round-trips through cliLocation", () => {
  assert.equal(cliSkillsHash("codex"), "#/clis/codex/skills");
  assert.equal(cliSkillsHash("codex", { workspaceId: "w1", scope: "workspace" }), "#/clis/codex/skills?workspaceId=w1&scope=workspace");
  assert.equal(cliSkillsHash("codex", { scope: "machine" }), "#/clis/codex/skills");
  assert.equal(cliSkillsHash("pi", { workspaceId: "w1", agentId: "a1", scope: "agent" }), "#/clis/pi/skills?workspaceId=w1&agentId=a1&scope=agent");
  assert.equal(cliLocation("#/clis/pi/skills?workspaceId=w1&agentId=a1&scope=agent").agentId, "a1");
  const loc = cliLocation("#/clis/codex/skills?workspaceId=w1&scope=machine");
  assert.equal(loc.pane, "skills");
  assert.equal(loc.workspaceId, "w1");
  assert.equal(loc.scope, "machine");
  assert.equal(cliLocation("#/clis/codex/skills?scope=bogus").invalid, true);
  assert.equal(cliLocation("#/clis/codex/skills/extra").invalid, true);
  assert.deepEqual(cliSkillsQuery(new URLSearchParams("")), { workspaceId: "", agentId: "", scope: "machine", invalid: false });
  assert.equal(cliLocation("#/clis/codex/skills?scope=all").invalid, true);
});

test("every CLI carries the Skills tab after Packages", () => {
  const panes = cliPanes({ id: "muse", integrationCapable: false, launchable: true, sessions: { list: true } });
  assert.equal(panes[panes.indexOf("packages") + 1], "skills");
  for (const cli of ["pi", "omp", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy"]) assert.ok(supportsCliSkills(cli), cli);
  assert.equal(supportsCliSkills("nope"), false);
});

test("the report path names the CLI and the workspace", () => {
  assert.equal(skillsReportPath("grok"), "/api/skills/report?cli=grok");
  assert.equal(skillsReportPath("grok", "w9"), "/api/skills/report?cli=grok&workspace=w9");
  assert.equal(skillsReportPath("pi", "w9", "a1"), "/api/skills/report?cli=pi&workspace=w9&agent=a1");
  assert.equal(skillsReportPath("pi", "", "a1"), "/api/skills/report?cli=pi");
});

test("each status reads in words, with the reason where there is one", () => {
  assert.equal(skillStatus(row(), "codex").label, "Loaded");
  assert.match(skillStatus(row({ status: "shadowed", shadowedBy: ".agents/skills" }), "muse").detail, /\.agents\/skills wins/);
  assert.match(skillStatus(row({ status: "needs-trust" }), "pi").detail, /Pi skips/);
  assert.match(skillStatus(row({ status: "if-trusted" }), "hermes").detail, /trusted in Hermes/);
  assert.equal(skillStatus(row({ status: "invalid", problems: ["no description"] }), "pi").detail, "no description");
});

test("origin names the installer's source, or says the folder was added by hand", () => {
  assert.equal(skillOrigin(row()).label, "Added by hand");
  const o = skillOrigin(row({ provenance: { installer: "skills", lock: "skills-lock.json", source: "anthropics/skills", modified: true } }));
  assert.equal(o.label, "anthropics/skills");
  assert.ok(o.modified);
  assert.match(o.title, /changed since it was installed/);
  assert.match(skillOrigin(row({ provenance: { installer: "hermes", lock: "~/.hermes/skills/.hub/lock.json", source: "" } })).label, /Hermes hub/);
});

test("filters keep the scope and the text, loaded first", () => {
  const rows = [
    row({ name: "zeta", status: "shadowed", root: "~/.claude/skills" }),
    row({ name: "zeta" }),
    row({ name: "alpha", scope: "workspace", root: ".agents/skills", status: "if-trusted" }),
    row({ name: "beta", description: "deploys to prod" }),
  ];
  assert.deepEqual(visibleSkills(rows).map((r) => r.name + ":" + r.status), ["beta:loaded", "zeta:loaded", "zeta:shadowed"]);
  assert.deepEqual(visibleSkills(rows, { scope: "workspace" }).map((r) => r.name), ["alpha"]);
  assert.deepEqual(visibleSkills(rows, { text: "PROD" }).map((r) => r.name), ["beta"]);
  // The agent chip: what it loads across scopes, or nothing when isolated.
  assert.deepEqual(visibleSkills(rows, { scope: "agent", agent: { isolated: false } }).map((r) => r.name), ["beta", "zeta", "alpha"]);
  assert.deepEqual(visibleSkills(rows, { scope: "agent", agent: { isolated: true } }), []);
  assert.deepEqual(skillsSummary(rows), { total: 4, loaded: 3, shadowed: 1, off: 0, tokens: 24 });
});

test("small words for cost and sharing", () => {
  assert.equal(tokensLabel(0), "");
  assert.equal(tokensLabel(42), "~42 tokens");
  assert.equal(tokensLabel(1520), "~1.5k tokens");
  assert.equal(alsoLoadedLine(row({ alsoLoadedBy: ["pi", "claude-code"] })), "Also read by Pi, Claude Code");
  assert.equal(alsoLoadedLine(row()), "");
});

test("the trust line appears only when workspace skills wait on trust", () => {
  const report = { trust: { needed: true, command: "hermes skills trust", note: "Hermes loads this workspace's skills after hermes skills trust." } };
  assert.equal(trustLine(report, [row()]), null);
  const t = trustLine(report, [row({ scope: "workspace", status: "if-trusted" })]);
  assert.equal(t.command, "hermes skills trust");
  assert.equal(t.known, false);
  assert.equal(trustLine({ trust: { needed: false } }, [row({ scope: "workspace", status: "if-trusted" })]), null);
});

test("an empty pane says why in one line", () => {
  assert.match(skillsEmptyLine("codex"), /on this machine\.$/);
  assert.match(skillsEmptyLine("codex", { hasWorkspace: true, workspaceName: "picode" }), /in picode/);
});

import { canRemoveSkill, conflictChoices, installBlocker, installLine, sizeLabel, skillTargetBody, updatesByKey, updatesSummary } from "./cliSkills.js";
import { skillSourceSchema } from "../contracts/schemas.js";

test("sources the form accepts, and the ones it refuses with a sentence", () => {
  for (const s of ["anthropics/skills", "anthropics/skills/skills/pdf#main", "https://github.com/o/r/tree/v1/x", "https://skills.example.com", "/opt/skills/x", "~/my-skill"]) {
    assert.equal(skillSourceSchema.safeParse({ source: s }).success, true, s);
  }
  for (const s of ["", "http://example.com", "not a source", "https://user:pw@x.com"]) {
    assert.equal(skillSourceSchema.safeParse({ source: s }).success, false, s);
  }
});

test("install, remove and update helpers", () => {
  assert.equal(canRemoveSkill({ root: ".agents/skills" }), true);
  assert.equal(canRemoveSkill({ root: "~/.claude/skills" }), false);
  assert.match(installLine("workspace", "demo"), /demo's \.agents\/skills folder/);
  assert.match(installLine("machine"), /~\/\.agents\/skills/);
  assert.deepEqual(conflictChoices("exists").map((c) => Object.keys(c.body)[0]), ["adopt", "replace"]);
  assert.deepEqual(conflictChoices("update")[0].body, { replace: true });
  assert.deepEqual(conflictChoices("stale"), []);
  const crit = { problems: [], findings: [{ severity: "critical", text: "x" }] };
  assert.match(installBlocker(crit, false), /confirm/);
  assert.equal(installBlocker(crit, true), "");
  assert.match(installBlocker({ problems: ["no description"], findings: [] }, true), /breaks the format/);
  assert.equal(sizeLabel(512), "512 B");
  assert.equal(sizeLabel(2048), "2.0 KB");
  assert.equal(updatesByKey([{ scope: "machine", name: "a", status: "behind" }])["machine:a"].status, "behind");
  assert.equal(updatesSummary([]), "Nothing here was installed from a source PiCode can check.");
  assert.equal(updatesSummary([{ name: "pdf", status: "behind" }, { status: "unreachable" }]), "Update available for pdf · 1 could not be checked.");
  assert.deepEqual(skillTargetBody({ name: "a", scope: "workspace" }, "w1"), { name: "a", scope: "workspace", workspace: "w1" });
});

import { agentScopeLine, skillScopes } from "./cliSkills.js";

test("chips run Global, the workspace, then the agent where the CLI has one", () => {
  assert.deepEqual(skillScopes({}).map((s) => s.label), ["Global"]);
  assert.deepEqual(skillScopes({ workspacePath: "/w" }, "picode").map((s) => s.label), ["Global", "picode"]);
  assert.deepEqual(skillScopes({ workspacePath: "/w", agent: { name: "delivery" } }, "picode").map((s) => s.id + ":" + s.label), ["machine:Global", "workspace:picode", "agent:delivery"]);
  assert.match(agentScopeLine({ agent: { name: "delivery", isolated: true } }, "omp"), /runs isolated/);
  assert.match(agentScopeLine({ agent: { name: "delivery", isolated: false } }, "omp"), /What delivery loads/);
  assert.equal(agentScopeLine({}, "pi"), "");
});

// --- slice 4: the agent's own skills (ADR-0196) -----------------------------
import { AGENT_ROOT, installedLine, removeQuestion } from "./cliSkills.js";

test("the agent chip lists the agent's own skills, isolated or not; other chips never do", () => {
  const rows = [
    { name: "own", scope: "agent", status: "loaded", root: AGENT_ROOT },
    { name: "gone", scope: "agent", status: "missing", root: AGENT_ROOT },
    { name: "shared", scope: "machine", status: "loaded", root: "~/.agents/skills" },
  ];
  assert.deepEqual(visibleSkills(rows, { scope: "agent", agent: { isolated: false } }).map((r) => r.name), ["own", "shared", "gone"]);
  assert.deepEqual(visibleSkills(rows, { scope: "agent", agent: { isolated: true } }).map((r) => r.name), ["own", "gone"]);
  assert.deepEqual(visibleSkills(rows, { scope: "machine" }).map((r) => r.name), ["shared"]);
});

test("agent rows: removable, addressed by agent, their words", () => {
  const row = { name: "own", scope: "agent", status: "missing", root: AGENT_ROOT, provenance: { installer: "picode", source: "/home/me/skills/own/" } };
  assert.equal(canRemoveSkill(row), true);
  assert.deepEqual(skillTargetBody(row, "w1", "a1"), { name: "own", scope: "agent", agent: "a1" });
  assert.equal(skillStatus(row, "pi").label, "Missing");
  assert.equal(skillOrigin(row).label, "own");
  assert.equal(skillOrigin({ provenance: { installer: "picode", source: "anthropics/skills" } }).label, "anthropics/skills");
  assert.match(skillStatus({ status: "shadowed", shadowedBy: AGENT_ROOT }, "omp").detail, /agent's own copy wins/);
  assert.match(installLine("agent", "demo", "delivery"), /for delivery alone/);
  assert.match(installedLine({ name: "own", status: "installed" }, "agent", "delivery"), /added to delivery\. It loads at the next start/);
  assert.match(installedLine({ name: "own", status: "already" }, "agent", "delivery"), /already on delivery/);
  assert.match(removeQuestion(row, { agentName: "delivery" }), /from delivery\? It stops loading at the next start/);
  assert.match(removeQuestion({ name: "x", scope: "workspace" }, { workspaceName: "demo" }), /from demo\? Every CLI/);
  assert.match(agentScopeLine({ agent: { name: "delivery" } }, "claude-code"), /\/picode-agent:<name>/);
});

import { noAgentScopeLine } from "./cliSkills.js";

test("a CLI without an agent scope says so when opened for an agent", () => {
  assert.match(noAgentScopeLine({}, "codex", "a1"), /^Codex cannot take skills for one agent/);
  assert.equal(noAgentScopeLine({}, "codex", ""), "");
  assert.equal(noAgentScopeLine({ agent: { name: "x" } }, "pi", "a1"), "");
  assert.equal(noAgentScopeLine({}, "claude-code", "a1"), "");
});

test("per-skill switches: the row flips at once and says what decides (slice 3)", async () => {
  const m = await import("./cliSkills.js");
  const report = { toggle: "skillOverrides in Claude Code's settings", rows: [
    { name: "pdf", dir: "/h/.claude/skills/pdf", scope: "machine", status: "loaded", enabled: true },
    { name: "old", dir: "/h/.claude/skills/old", scope: "machine", status: "shadowed" },
  ] };
  assert.equal(m.hasSkillSwitch(report.rows[0]), true);
  assert.equal(m.hasSkillSwitch(report.rows[1]), false);
  const off = m.withSkillEnabled(report, "/h/.claude/skills/pdf", "machine", false);
  assert.equal(off.rows[0].status, "disabled");
  assert.equal(off.rows[0].enabled, false);
  assert.equal(off.rows[1], report.rows[1]);
  assert.equal(m.withSkillEnabled(off, "/h/.claude/skills/pdf", "machine", true).rows[0].status, "loaded");
  assert.equal(m.skillStatus({ status: "disabled" }, "claude-code").label, "Off");
  assert.equal(m.skillsSummary(off.rows).off, 1);
  assert.deepEqual(m.skillToggleBody("codex", report.rows[0], "ws1", false), { cli: "codex", scope: "machine", dir: "/h/.claude/skills/pdf", enabled: false, workspace: "ws1" });
  assert.equal(m.toggledLine({ name: "pdf", enabled: false }), "pdf is off.");
  assert.equal(m.toggledLine({ name: "pdf", enabled: true, note: "x decides" }), "pdf is on. x decides.");
  // Pi and Antigravity have no switch: one sentence, only when there are skills.
  assert.match(m.noSwitchLine({ rows: [{}] }, "pi"), /^Pi has no switch/);
  assert.equal(m.noSwitchLine(report, "claude-code"), "");
  assert.equal(m.noSwitchLine({ rows: [] }, "pi"), "");
});

// --- slice 5: the Marketplace ------------------------------------------------
import { catalogEmptyLine, catalogInstalled, catalogOrigin, installedSkillKeys, skillCatalogPath, sourceStateLine } from "./cliSkills.js";

test("marketplace words", () => {
  assert.equal(skillCatalogPath(""), "/api/skills/catalog");
  assert.equal(skillCatalogPath(" pdf "), "/api/skills/catalog?q=pdf");
  assert.equal(catalogOrigin({ origin: "skills.sh", installs: 200826, source: "a/b" }).label, "a/b · skills.sh · 201k installs");
  assert.equal(catalogOrigin({ origin: "skills.sh", installs: 2147 }).label, "skills.sh · 2.1k installs");
  assert.equal(catalogOrigin({ origin: "seed", source: "anthropics/skills" }).label, "anthropics/skills");
  const keys = installedSkillKeys({ rows: [{ name: "pdf", status: "loaded", provenance: { source: "anthropics/skills" } }, { name: "mine", status: "loaded" }] });
  assert.equal(catalogInstalled(keys, { name: "pdf", source: "anthropics/skills" }), true);
  assert.equal(catalogInstalled(keys, { name: "pdf", source: "openai/skills" }), false, "same name, other source");
  assert.equal(catalogInstalled(keys, { name: "mine", source: "a/b" }), false, "a hand-made folder names no source");
  assert.equal(sourceStateLine({ reading: true }), "Reading…");
  assert.equal(sourceStateLine({ count: 20 }), "20 skills");
  assert.equal(sourceStateLine({ count: 0, error: "not found" }), "not found");
  assert.equal(sourceStateLine({ count: 3, error: "limited" }), "3 skills · last read failed: limited");
  assert.match(catalogEmptyLine("pdf", false), /No skill matches “pdf”/);
  assert.match(catalogEmptyLine("", true), /Reading the sources/);
});
