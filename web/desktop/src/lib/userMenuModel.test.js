import test from "node:test";
import assert from "node:assert/strict";
import { MENU_SECTIONS, MENU_GROUPS, MENU_ACTIONS, menuGroups, menuActions, menuHasResults } from "./userMenuModel.js";

test("every group id resolves to a section; every section is reachable", () => {
  const grouped = new Set(MENU_GROUPS.flatMap(([, ids]) => ids));
  for (const id of grouped) assert.ok(MENU_SECTIONS.find(row => row[0] === id), id);
  const reachable = new Set([...grouped, ...MENU_ACTIONS.map(a => a.id), "packages", "providers", "clis"]);
  for (const [id] of MENU_SECTIONS) assert.ok(reachable.has(id), id);
});

test("blank query returns all groups and actions (footer visible)", () => {
  assert.equal(menuGroups("").length, MENU_GROUPS.length);
  assert.equal(menuGroups("").flatMap(g => g.rows).length, MENU_SECTIONS.length - 3);
  assert.equal(menuActions("").length, MENU_ACTIONS.length);
  assert.equal(menuHasResults(""), true);
});

test("a query filters groups and actions by title, subtitle and id", () => {
  const groups = menuGroups("keys");
  assert.deepEqual(groups.map(g => g.title), ["Agent CLIs"]);
  assert.deepEqual(groups[0].rows.map(r => r[0]), ["providers"]);
  assert.deepEqual(menuActions("qr").map(a => a.id), ["share"]);
  assert.equal(menuHasResults("keys"), true);
});

test("a query with no matches empties the menu (empty state row shows)", () => {
  assert.deepEqual(menuGroups("zzz"), []);
  assert.deepEqual(menuActions("zzz"), []);
  assert.equal(menuHasResults("zzz"), false);
});

test("Packages is searchable inside Agent CLIs", () => {
  assert.ok(!menuGroups("").flatMap(g => g.rows).some(row => row[0] === "packages"));
  assert.ok(menuGroups("packages").some(group => group.title === "Agent CLIs" && group.rows.some(row => row[0] === "packages")));
});

test("Providers is searchable inside Agent CLIs", () => {
  assert.ok(!menuGroups("").flatMap(g => g.rows).some(row => row[0] === "providers"));
  assert.ok(menuGroups("providers").some(group => group.title === "Agent CLIs" && group.rows.some(row => row[0] === "providers")));
});

test("Tools holds Integrations and llama.cpp; Agents and connections is gone", () => {
  assert.deepEqual(MENU_GROUPS.map(([title]) => title), ["Tools", "PiCode"]);
  assert.deepEqual(MENU_GROUPS[0][1], ["automations", "llama", "integrations"]);
  assert.deepEqual(menuGroups("").map(g => g.title), ["Tools", "PiCode"]);
  assert.deepEqual(menuGroups("").find(g => g.title === "Tools").rows.map(r => r[0]), ["automations", "llama", "integrations"]);
});

test("Agent CLIs is searchable, not a default Tools row", () => {
  assert.ok(!menuGroups("").flatMap(g => g.rows).some(row => row[0] === "clis"));
  assert.ok(menuGroups("clis").some(group => group.rows.some(row => row[0] === "clis")));
  const tools = menuGroups("a").filter(g => g.title === "Tools");
  assert.equal(tools.length, 1);
  assert.ok(tools[0].rows.some(row => row[0] === "clis"));
});

test("llama.cpp is reachable from Tools and searchable", () => {
  assert.ok(!menuGroups("").flatMap(g => g.rows).some(row => row[0] === "providers"));
  assert.ok(menuGroups("").find(g => g.title === "Tools").rows.some(row => row[0] === "llama"));
  assert.ok(menuGroups("llama").some(group => group.title === "Tools" && group.rows.some(row => row[0] === "llama")));
});
