import { extraSlash, filterSlash } from "./slash.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("slash only when leading /", () => {
  assert.equal(filterSlash("hello").length, 0);
  assert.ok(filterSlash("/").length > 3);
});

test("filters by prefix", () => {
  const hits = filterSlash("/log");
  assert.equal(hits[0].id, "login");
});

test("/scoped-models opens settings", () => {
  const hits = filterSlash("/scoped");
  assert.equal(hits[0].id, "scoped-models");
  assert.equal(hits[0].run, "go-scoped");
});

test("/settings is a PiCode route, not a TUI proxy", () => {
  const hits = filterSlash("/settings");
  assert.equal(hits[0].id, "settings");
  assert.equal(hits[0].run, "go-settings");
});

test("copy quit reload logout session trust are PiCode UI", () => {
  const run = (q) => filterSlash(q)[0].run;
  assert.equal(run("/copy"), "copy");
  assert.equal(run("/quit"), "quit");
  assert.equal(run("/reload"), "reload");
  assert.equal(run("/login"), "go-providers-new");
  assert.equal(run("/logout"), "go-providers");
  assert.equal(run("/session"), "session-info");
  assert.equal(run("/trust"), "trust");
  assert.equal(run("/export"), "export");
  assert.equal(run("/import"), "import");
  assert.equal(run("/hotkeys"), "hotkeys");
  assert.equal(run("/changelog"), "changelog");
  assert.equal(run("/share"), "share");
  assert.equal(run("/llama"), "go-llama");
});

test("skills and templates insert into composer", () => {
  const extra = extraSlash([{ name: "brave-search", hint: "Web" }], [{ name: "review", hint: "Diff" }]);
  const skill = filterSlash("/skill:br", extra)[0];
  assert.equal(skill.run, "insert");
  assert.equal(skill.insert, "/skill:brave-search ");
  const tpl = filterSlash("/rev", extra).find((c) => c.id === "tpl:review");
  assert.ok(tpl);
  assert.equal(tpl.insert, "/review ");
});
