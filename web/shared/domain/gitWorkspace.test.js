import assert from "node:assert/strict";
import { test } from "node:test";
import { pickerOptions, workspaceForOwner, triggerLabel } from "./gitWorkspace.js";

const wsRepo = (name, path, branch, id) => ({
  id: id || "w-" + name,
  name,
  path,
  git: branch ? { branch } : null,
  agents: [],
});

const desktop = wsRepo("desktop", "/home/goat/picode", "main");
const comms = wsRepo("communication", "/home/goat/picode/.worktrees/comms", "feat/comms");
const notes = wsRepo("notes", "/home/goat/notes", ""); // folder, not a repository
const orikami = wsRepo("ORIKAMI", "/home/goat/orikami", "main");
const workspaces = [desktop, comms, notes, orikami];

test("pickerOptions lists repositories only, sorted, with the branch as the hint", () => {
  const opts = pickerOptions(workspaces);
  assert.deepEqual(opts.map((o) => o.id), ["w-communication", "w-desktop", "w-ORIKAMI"]);
  assert.deepEqual(opts.map((o) => o.label), ["communication", "desktop", "ORIKAMI"]);
  assert.equal(opts[0].hint, "feat/comms");
  assert.equal(opts[1].hint, "main");
  // The folder is the row's title, not the hint: a deep worktree path elides
  // to `~/picode/.worktrees/gg-wo…`, which disambiguates nothing.
  assert.equal(opts[0].title, "/home/goat/picode/.worktrees/comms");
  // A folder that is not a repository is never offered: the route would 404.
  assert.equal(opts.some((o) => o.id === "w-notes"), false);
});

test("pickerOptions is empty for no workspaces at all", () => {
  assert.deepEqual(pickerOptions([]), []);
  assert.deepEqual(pickerOptions(null), []);
});

test("workspaceForOwner answers for every owner kind, and null when there is none", () => {
  const terminals = [
    { id: "t-desktop", workspaceId: "w-desktop" },
    { id: "t-free" },
    { id: "t-reserved", workspaceId: "ws_free" },
  ];
  const freeAgents = [{ id: "a-free" }];
  const withAgents = [{ ...desktop, agents: [{ id: "a-desktop" }] }, comms];
  const fleet = { workspaces: withAgents, freeAgents, terminals };
  assert.equal(workspaceForOwner({ kind: "workspace", id: "w-desktop" }, fleet), withAgents[0]);
  assert.equal(workspaceForOwner({ kind: "agent", id: "a-desktop" }, fleet), withAgents[0]);
  assert.equal(workspaceForOwner({ kind: "term", id: "t-desktop" }, fleet), withAgents[0]);
  // Neither a free agent nor a free terminal belongs to a workspace, and the
  // reserved id is nobody's.
  assert.equal(workspaceForOwner({ kind: "agent", id: "a-free" }, fleet), null);
  assert.equal(workspaceForOwner({ kind: "term", id: "t-free" }, fleet), null);
  assert.equal(workspaceForOwner({ kind: "term", id: "t-reserved" }, fleet), null);
  // A workspace that was removed while the surface stayed open names nothing.
  assert.equal(workspaceForOwner({ kind: "workspace", id: "w-gone" }, fleet), null);
  assert.equal(workspaceForOwner(null, fleet), null);
  assert.equal(workspaceForOwner({ kind: "workspace", id: "w-desktop" }, {}), null);
});

test("triggerLabel prefers the workspace, falls back to the repository name", () => {
  assert.equal(triggerLabel(desktop, "picode"), "desktop");
  assert.equal(triggerLabel(null, "picode"), "picode");
  // An unnamed workspace still names something: its folder.
  assert.equal(triggerLabel({ id: "w", path: "/home/goat/picode" }, "picode"), "/home/goat/picode");
  assert.equal(triggerLabel(null, ""), "");
});
