import assert from "node:assert/strict";
import { test } from "node:test";
import { repoLine, termLine, shortPath } from "./repoLine.js";

test("a repo shows path / branch, spaced", () => {
  const r = repoLine({ workPath: "/home/goat/picode", git: { branch: "main" } }, null);
  assert.equal(r.text, "~/picode / main");
  assert.equal(r.git.branch, "main"); // the object, not a boolean — tooltips read it
});

test("a worktree's name is already in the path, not appended again", () => {
  const r = repoLine({ workPath: "/home/goat/picode/.worktrees/fix-x", git: { branch: "fix-x", worktree: "fix-x" } }, null);
  assert.equal(r.text, "~/picode/.worktrees/fix-x / fix-x");
});

test("workspace git backs an agent without its own", () => {
  const r = repoLine({}, { path: "/home/goat/picode", git: { branch: "trunk" } });
  assert.equal(r.text, "~/picode / trunk");
});

test("not a repo shows the dir alone", () => {
  assert.deepEqual(
    repoLine({ workPath: "/home/goat/.picode/work/grok" }, null),
    { git: null, text: "~/.picode/work/grok" },
  );
});

test("a detached head still reads as git (branch is the short hash)", () => {
  const r = repoLine({ workPath: "/tmp/x", git: { branch: "a1b2c3d" } }, null);
  assert.equal(r.git.branch, "a1b2c3d");
  assert.equal(r.text, "/tmp/x / a1b2c3d");
});

test("termLine mirrors the agent line, from the live cwd", () => {
  assert.equal(termLine({ cwd: "/home/goat/picode", git: { branch: "main" } }).text, "~/picode / main");
  assert.deepEqual(termLine({ cwd: "/tmp/scratch" }), { git: null, text: "/tmp/scratch" });
  assert.equal(termLine({ cwd: "/home/goat" }).text, "~");
});

test("an agent on a non-repo workPath never borrows the workspace's branch", () => {
  const r = repoLine(
    { workPath: "/home/goat/notes" },                     // its dir, not a repo
    { path: "/home/goat/picode", git: { branch: "main" } }, // the workspace is
  );
  assert.deepEqual(r, { git: null, text: "~/notes" });
});

test("shortPath", () => {
  assert.equal(shortPath(""), "—");
  assert.equal(shortPath("/tmp/x"), "/tmp/x");
  assert.equal(shortPath("/home/goat"), "~"); // the bare home is home too
  assert.equal(shortPath("/home/goat/"), "~");
  assert.equal(shortPath("/Users/goat/dev"), "~/dev");
});
