import assert from "node:assert/strict";
import { test } from "node:test";
import { cliPackagesHash, cliPackagesLocation, supportsCliPackages, loadPiPackagesContext, packageContextKey } from "./cliPackages.js";
import { cliLocation } from "./cliLaunch.js";

test("canonical links round-trip CLI, package, scope and explicit context", () => {
  const context = { workspaceId: "w /&", agentId: "a /?", scope: "agent", pkg: "@scope/roles" };
  const route = cliPackagesLocation(cliPackagesHash("pi", context));
  for (const [key, value] of Object.entries(context)) assert.equal(route[key], value);
  assert.equal(route.legacy, false);
  assert.equal(route.redirect, "");
  assert.equal(cliLocation(cliPackagesHash("pi", context)).view, "packages");
});

test("legacy links adopt a pane only when they have no explicit context", () => {
  for (const prefix of ["#/packages", "#/more/packages"]) {
    const route = cliPackagesLocation(prefix + "/config/pi-roles", { workspaceId: "w", agentId: "a" });
    assert.equal(route.redirect, "#/clis/packages/pi/config/pi-roles?workspaceId=w&agentId=a");
    assert.equal(cliPackagesLocation(prefix + "?workspaceId=x", { agentId: "a" }).agentId, "");
    assert.equal(cliPackagesLocation(prefix + "?agentId=", { agentId: "a" }).agentId, "");
  }
});

test("canonical machine links never inherit the current pane", () => {
  const route = cliPackagesLocation("#/clis/packages/pi", { workspaceId: "w", agentId: "a" });
  assert.equal(route.workspaceId, ""); assert.equal(route.agentId, "");
  assert.equal(cliPackagesLocation("#/clis/packages").redirect, "#/clis/packages/pi");
  assert.equal(cliPackagesLocation("#/providers"), null);
});

test("unsupported CLIs and malformed links never become Pi package targets", () => {
  assert.equal(supportsCliPackages("pi"), true);
  assert.equal(supportsCliPackages("codex"), false);
  assert.equal(cliPackagesLocation("#/clis/packages/codex").id, "codex");
  for (const hash of ["#/clis/packages/%zz", "#/clis/packages/pi/config/", "#/packages/nope", "#/clis/packages/pi?scope=everyone"]) {
    assert.equal(cliPackagesLocation(hash).invalid, true, hash);
  }
});

const rows = [{ id: "w", name: "Workspace", agents: [{ id: "a", workspaceId: "w" }] }, { id: "other", agents: [] }];
const request = async path => {
  if (path === "/api/workspaces") return rows;
  if (path === "/api/agents?free=1") return [{ id: "free", workspaceId: "ws_free" }];
  throw new Error("Unexpected API: " + path);
};

test("machine packages need no native settings, fleet or terminal reads", async () => {
  assert.deepEqual(await loadPiPackagesContext({}, () => assert.fail("unexpected API")), { workspace: null, agent: null });
});

test("workspace and attached agent targets resolve independently of the selected pane", async () => {
  for (const input of [{ workspaceId: "w" }, { agentId: "a" }, { workspaceId: "w", agentId: "a", scope: "agent" }]) {
    const result = await loadPiPackagesContext(input, request);
    assert.equal(result.workspace.id, "w");
    assert.equal(result.agent?.id || "", input.agentId || "");
  }
  const free = await loadPiPackagesContext({ agentId: "free", scope: "agent" }, request);
  assert.equal(free.workspace, null); assert.equal(free.agent.id, "free");
});

test("missing identities and incompatible scopes block instead of falling back", async () => {
  for (const input of [{ workspaceId: "missing" }, { agentId: "missing" }, { workspaceId: "other", agentId: "a" },
    { workspaceId: "w", agentId: "free" }, { agentId: "free", scope: "project" }, { scope: "agent" }, { scope: "project" }, { scope: "invalid" }]) {
    await assert.rejects(loadPiPackagesContext(input, request), error => error.status === 404, JSON.stringify(input));
  }
});

test("transient read failure stays distinguishable from a deleted target", async () => {
  const error = Object.assign(new Error("offline"), { status: 503 });
  await assert.rejects(loadPiPackagesContext({ agentId: "a" }, async () => { throw error; }), err => err === error);
});

test("draft targets ignore display changes but include workspace and agent file locations", () => {
  const value = { workspace: { id: "w", path: "/w", name: "Old" }, agent: { id: "a", workPath: "/w/tree", mode: "stopped" } };
  assert.equal(packageContextKey(value), packageContextKey({ workspace: { ...value.workspace, name: "New" }, agent: { ...value.agent, mode: "managed" } }));
  assert.notEqual(packageContextKey(value), packageContextKey({ ...value, workspace: { ...value.workspace, path: "/other" } }));
  assert.notEqual(packageContextKey(value), packageContextKey({ ...value, agent: { ...value.agent, workPath: "/other" } }));
});
