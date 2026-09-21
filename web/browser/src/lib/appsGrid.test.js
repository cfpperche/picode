import { test } from "node:test";
import assert from "node:assert/strict";
import { visibleApps } from "./appsGrid.js";

const REGISTRATION = [
  { id: "inbox", name: "Inbox" },
  { id: "docker", name: "Docker" },
  { id: "canvas", name: "Canvas" },
  { id: "tmux", name: "tmux" },
];

test("empty query keeps every app, alphabetical regardless of registration order", () => {
  assert.deepEqual(visibleApps(REGISTRATION, "").map((a) => a.name), ["Canvas", "Docker", "Inbox", "tmux"]);
  assert.deepEqual(visibleApps(REGISTRATION, "   ").map((a) => a.name), ["Canvas", "Docker", "Inbox", "tmux"]);
});

test("sort is case-insensitive (base sensitivity), like the free-agents list", () => {
  const mixed = [{ id: "a", name: "canvas" }, { id: "b", name: "Inbox" }, { id: "c", name: "DOCKER" }];
  assert.deepEqual(visibleApps(mixed, "").map((a) => a.name), ["canvas", "DOCKER", "Inbox"]);
});

test("query filters by name, case-insensitive, result still sorted", () => {
  assert.deepEqual(visibleApps(REGISTRATION, "do").map((a) => a.name), ["Docker"]);
  assert.deepEqual(visibleApps(REGISTRATION, "CAN").map((a) => a.name), ["Canvas"]);
});

test("query with no match answers an empty list (the pane shows its own empty line)", () => {
  assert.deepEqual(visibleApps(REGISTRATION, "zzz"), []);
});

test("the input list is never mutated", () => {
  const copy = REGISTRATION.map((a) => ({ ...a }));
  visibleApps(REGISTRATION, "");
  assert.deepEqual(REGISTRATION, copy);
});

test("a saved order leads, and apps missing from it append by name", () => {
  assert.deepEqual(visibleApps(REGISTRATION, "", ["tmux", "inbox"]).map((a) => a.id), ["tmux", "inbox", "canvas", "docker"]);
  assert.deepEqual(visibleApps(REGISTRATION, "in", ["tmux", "inbox"]).map((a) => a.id), ["inbox"]);
});

test("null-safe: missing list or junk rows answer empty, never throw", () => {
  assert.deepEqual(visibleApps(null, ""), []);
  assert.deepEqual(visibleApps(undefined, "do"), []);
  assert.deepEqual(visibleApps([{ id: "x" }, null, "junk"], "").length, 3);
});
