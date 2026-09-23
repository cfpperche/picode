import test from "node:test";
import assert from "node:assert/strict";
import { MORE_SECTIONS, MORE_GROUPS, MORE_ACTIONS, moreGroups, moreActions, moreHasResults } from "./moreMenuModel.js";

test("every group id resolves to a section; every section is reachable", () => {
  const grouped = new Set(MORE_GROUPS.flatMap(([, ids]) => ids));
  for (const id of grouped) assert.ok(MORE_SECTIONS.find(row => row[0] === id), id);
  const reachable = new Set([...grouped, ...MORE_ACTIONS.map(a => a.id), "packages", "providers", "connectors"]);
  for (const [id] of MORE_SECTIONS) assert.ok(reachable.has(id), id);
});

test("Tools holds Agent CLIs and llama.cpp; Webhooks sit under PiCode", () => {
  assert.deepEqual(MORE_GROUPS.map(([title]) => title), ["Tools", "PiCode"]);
  assert.deepEqual(MORE_GROUPS[0][1], ["pins", "snippets", "outcomes", "clis", "automations", "apps", "llama"]);
  assert.deepEqual(moreGroups("").find(g => g.title === "Tools").rows.map(r => r[0]), ["pins", "snippets", "outcomes", "clis", "automations", "apps", "llama"]);
  assert.ok(moreGroups("").find(g => g.title === "PiCode").rows.some(row => row[0] === "integrations"));
});

test("blank query returns all groups and actions", () => {
  assert.equal(moreGroups("").length, MORE_GROUPS.length);
  assert.equal(moreGroups("").flatMap(g => g.rows).length, MORE_SECTIONS.length - 3);
  assert.equal(moreActions("").length, MORE_ACTIONS.length);
  assert.equal(moreHasResults(""), true);
});

test("a query filters groups and actions by title, subtitle and id", () => {
  const groups = moreGroups("keys");
  assert.deepEqual(groups.map(g => g.title), ["Agent CLIs"]);
  assert.deepEqual(groups[0].rows.map(r => r[0]), ["pi-providers"]);
  assert.deepEqual(moreActions("qr").map(a => a.id), ["pair"]);
  assert.equal(moreHasResults("keys"), true);
});

test("a query with no matches empties the menu", () => {
  assert.deepEqual(moreGroups("zzz"), []);
  assert.deepEqual(moreActions("zzz"), []);
  assert.equal(moreHasResults("zzz"), false);
});

test("Packages and Providers stay searchable inside Agent CLIs", () => {
  assert.ok(!moreGroups("").flatMap(g => g.rows).some(row => row[0] === "packages" || row[0] === "providers" || row[0] === "connectors"));
  assert.ok(moreGroups("packages").some(group => group.title === "Agent CLIs" && group.rows.some(row => row[0] === "pi-packages")));
  assert.ok(moreGroups("providers").some(group => group.title === "Agent CLIs" && group.rows.some(row => row[0] === "pi-providers")));
  assert.ok(moreGroups("mcp").some(group => group.rows.some(row => row[0] === "connectors")));
});

test("CLI shortcuts share one Agent CLIs group", () => {
  const groups = moreGroups("s").filter((g) => g.title === "Agent CLIs");
  assert.equal(groups.length, 1);
  assert.deepEqual(groups[0].rows.map((r) => r[1]), ["CLI settings", "Packages", "Providers", "Connectors"]);
});
