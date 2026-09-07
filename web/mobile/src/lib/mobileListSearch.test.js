import test from "node:test";
import assert from "node:assert/strict";
import { matchesListSearch, searchWorkspaceGroups, terminalMatchesSearch } from "./mobileListSearch.js";

const workspaces = [
  { id: "one", name: "Café project", path: "/work/coffee", agents: [{ id: "a", name: "Atlas", model: "sonnet" }, { id: "b", name: "Nova", model: "opus" }] },
  { id: "two", name: "Website", path: "/work/site", agents: [] },
];
const terminals = [{ id: "t", workspaceId: "one", name: "Build", cli: "codex", cwd: "/work/coffee" }, { id: "free", name: "Free", cwd: "/tmp" }];

test("list search matches all words across fields without case or accent sensitivity", () => {
  assert.equal(matchesListSearch("CAFE sonnet", "Café", "Sonnet"), true);
  assert.equal(matchesListSearch("cafe missing", "Café", "Sonnet"), false);
  assert.equal(matchesListSearch("   ", ""), true);
});

test("matching a workspace keeps its full group, including an empty workspace", () => {
  const result = searchWorkspaceGroups(workspaces, terminals, "cafe");
  assert.deepEqual(result.map(group => [group.workspace.id, group.agents.map(a => a.id), group.terminals.map(t => t.id)]), [["one", ["a", "b"], ["t"]]]);
  assert.equal(searchWorkspaceGroups(workspaces, terminals, "website")[0].agents.length, 0);
  assert.equal(searchWorkspaceGroups(workspaces, terminals, "").length, 2);
});

test("matching a child retains workspace context and hides unrelated children", () => {
  const agents = searchWorkspaceGroups(workspaces, terminals, "nova");
  assert.deepEqual(agents.map(group => [group.workspace.id, group.agents.map(a => a.id), group.terminals.length]), [["one", ["b"], 0]]);
  const terms = searchWorkspaceGroups(workspaces, terminals, "codex");
  assert.deepEqual(terms.map(group => [group.workspace.id, group.agents.length, group.terminals.map(t => t.id)]), [["one", 0, ["t"]]]);
  assert.deepEqual(searchWorkspaceGroups(workspaces, terminals, "absent"), []);
  assert.equal(terminalMatchesSearch(terminals[0], "codex build"), true);
  assert.deepEqual(searchWorkspaceGroups(workspaces, terminals, "cafe nova")[0].agents.map(a => a.id), ["b"]);
});
