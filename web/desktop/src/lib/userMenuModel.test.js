import test from "node:test";
import assert from "node:assert/strict";
import { MENU_SECTIONS, MENU_GROUPS, MENU_ACTIONS, menuGroups, menuActions, menuHasResults } from "./userMenuModel.js";

test("every group id resolves to a section; every section is reachable", () => {
  const grouped = new Set(MENU_GROUPS.flatMap(([, ids]) => ids));
  for (const id of grouped) assert.ok(MENU_SECTIONS.find(row => row[0] === id), id);
  const reachable = new Set([...grouped, ...MENU_ACTIONS.map(a => a.id), "packages"]);
  for (const [id] of MENU_SECTIONS) assert.ok(reachable.has(id), id);
});

test("blank query returns all groups and actions (footer visible)", () => {
  assert.equal(menuGroups("").length, MENU_GROUPS.length);
  assert.equal(menuGroups("").flatMap(g => g.rows).length, MENU_SECTIONS.length);
  assert.equal(menuActions("").length, MENU_ACTIONS.length);
  assert.equal(menuHasResults(""), true);
});

test("a query filters groups and actions by title, subtitle and id", () => {
  const groups = menuGroups("keys");
  assert.deepEqual(groups.map(g => g.title), ["Agents and connections"]);
  assert.deepEqual(groups[0].rows.map(r => r[0]), ["providers"]);
  assert.deepEqual(menuActions("qr").map(a => a.id), ["share"]);
  assert.equal(menuHasResults("keys"), true);
});

test("a query with no matches empties the menu (empty state row shows)", () => {
  assert.deepEqual(menuGroups("zzz"), []);
  assert.deepEqual(menuActions("zzz"), []);
  assert.equal(menuHasResults("zzz"), false);
});
