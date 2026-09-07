import test from "node:test";
import assert from "node:assert/strict";
import { deliverGitCommand } from "./gitDelivery.js";
const conflict = message => Object.assign(new Error(message), { status: 409 });
const config = { owner: { kind: "workspace", id: "ws" }, root: "/project", command: "git fetch --prune", workspaceId: "ws", wait: async () => {}, terminals: [{ id: "idle", cwd: "/project" }] };
function recorder(fail) {
  const calls = [];
  return { calls, api: async (path, options) => {
    calls.push({ path, body: JSON.parse(options.body) });
    const error = fail?.(path, calls.length); if (error) throw error;
    return path === "/api/terminals" ? { id: "new" } : {};
  } };
}
for (const run of [false, true]) test("idle terminal delivers via " + (run ? "run" : "type") + " with the pinned folder", async () => {
  const io = recorder(); const result = await deliverGitCommand({ ...config, run, api: io.api });
  assert.deepEqual(io.calls, [{ path: "/api/terminals/idle/" + (run ? "run" : "type"), body: { text: config.command, root: config.root } }]);
  assert.equal(result.ran, run);
});
test("busy repository prepares once after a confirmed run refusal", async () => {
  const io = recorder(path => path.endsWith("/run") ? conflict("Atlas is working here.") : null);
  const result = await deliverGitCommand({ ...config, run: true, api: io.api });
  assert.equal(result.ran, false); assert.match(result.reason, /Atlas/);
  assert.deepEqual(io.calls.map(call => call.path), ["/api/terminals/idle/run", "/api/terminals/idle/type"]);
});
for (const message of ["This terminal moved to /elsewhere", "This terminal is running a CLI"]) test(message + " opens a separate shell", async () => {
  const io = recorder(path => path.includes("/idle/") ? conflict(message) : null);
  const result = await deliverGitCommand({ ...config, api: io.api });
  assert.equal(result.id, "new");
  assert.deepEqual(io.calls[1], { path: "/api/terminals", body: { name: "git", cwd: "/project", workspaceId: "ws" } });
  assert.equal(io.calls[2].path, "/api/terminals/new/type");
});
for (const error of [new Error("Network disconnected"), Object.assign(new Error("This terminal is running something"), { status: 502 }), Object.assign(new Error("Unauthorized"), { status: 401 })]) test("unknown or rejected run outcome is not retried: " + error.message, async () => {
  const io = recorder(() => error);
  await assert.rejects(deliverGitCommand({ ...config, run: true, api: io.api }), error);
  assert.equal(io.calls.length, 1);
});
for (const flag of ["tui", "cli", "launchCli", "state"]) test("never reuse a " + flag + " terminal", async () => {
  const io = recorder(); await deliverGitCommand({ ...config, terminals: [{ id: "busy", cwd: "/project", [flag]: "active" }], api: io.api });
  assert.equal(io.calls[0].path, "/api/terminals"); assert.equal(io.calls[1].path, "/api/terminals/new/type");
});
test("owner terminal is checked by the server even when fleet cwd is stale", async () => {
  const io = recorder(); await deliverGitCommand({ ...config, owner: { kind: "term", id: "own" }, api: io.api });
  assert.equal(io.calls[0].path, "/api/terminals/own/type");
});
test("missing root cannot create or type in a shell", async () => {
  const io = recorder(); await assert.rejects(deliverGitCommand({ ...config, root: "", api: io.api }), /project folder/); assert.equal(io.calls.length, 0);
});

for (const reason of ["cli", "closed"]) test("owner terminal " + reason + " refusal safely selects a plain shell", async () => {
  const io = recorder(path => path.includes("/own/") ? Object.assign(conflict("A plain shell is required."), { body: { reason } }) : null);
  const result = await deliverGitCommand({ ...config, owner: { kind: "term", id: "own" }, run: true, api: io.api });
  assert.equal(result.id, "new"); assert.equal(result.ran, true);
  assert.deepEqual(io.calls.map(call => call.path), ["/api/terminals/own/run", "/api/terminals", "/api/terminals/new/run"]);
});
test("known stopped shells are never reused", async () => {
  const io = recorder(); await deliverGitCommand({ ...config, terminals: [{ id: "closed", cwd: "/project", running: false }], api: io.api });
  assert.equal(io.calls[0].path, "/api/terminals");
});
