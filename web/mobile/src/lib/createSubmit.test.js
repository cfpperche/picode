import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { createRequest } from "./createSubmit.js";

const cfg = { provider: "xai", model: "grok-4.6", thinking: "medium" };

describe("createRequest", () => {
  it("builds a workspace create", () => {
    const r = createRequest("workspace", { name: "App", path: "~/code/app", source: "local" }, null, "");
    assert.equal(r.path, "/api/workspaces");
    assert.equal(r.body.name, "App");
  });
  it("builds a clone when the source is remote", () => {
    const r = createRequest("workspace", { name: "app", path: "~/code/app", source: "remote", url: "https://github.com/o/app" }, null, "");
    assert.equal(r.path, "/api/workspaces/clone");
    assert.equal(r.clone, true);
  });
  it("builds a free agent and a workspace agent", () => {
    const f = createRequest("free", { name: "solo", path: "" }, cfg, "");
    assert.equal(f.path, "/api/agents");
    assert.equal(f.body.model, "grok-4.6");
    const a = createRequest("agent", { name: "helper" }, cfg, "ws 1");
    assert.equal(a.path, "/api/workspaces/ws%201/agents");
    assert.deepEqual(Object.keys(a.body).sort(), ["model", "name", "provider", "thinking"]);
  });
  it("reports validation errors instead of posting", () => {
    assert.ok(createRequest("workspace", { name: "", path: "" }, null, "").error);
    assert.ok(createRequest("agent", { name: "x" }, { provider: "", model: "", thinking: "" }, "w").error);
    assert.equal(createRequest("agent", { name: "x" }, cfg, "").error, "Pick a workspace first.");
  });
});
