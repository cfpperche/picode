import assert from "node:assert/strict";
import { test } from "node:test";
import { pickerOptions, workspaceForOwner, triggerLabel, hintPath } from "./workspacePicker.js";

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

test("pickerOptions offers every folder by default and repositories only on request", () => {
  // The file tree draws any folder (ADR-0030: no repository is a state, not an
  // error), so the default filter is "has a folder" and nothing else.
  const all = pickerOptions(workspaces);
  assert.deepEqual(all.map((o) => o.id), ["w-communication", "w-desktop", "w-notes", "w-ORIKAMI"]);
  // A Git surface may not: a folder that is not a repository has no history,
  // and the graph route would answer 404.
  const repos = pickerOptions(workspaces, { reposOnly: true });
  assert.deepEqual(repos.map((o) => o.id), ["w-communication", "w-desktop", "w-ORIKAMI"]);
  // A workspace with no folder at all is never offered: there is nothing to read.
  assert.deepEqual(pickerOptions([...workspaces, { id: "w-nofolder", name: "ghost" }]).some((o) => o.id === "w-nofolder"), false);
});

test("pickerOptions: the hint is the branch by default and the folder when asked", () => {
  const branch = pickerOptions(workspaces, { reposOnly: true });
  assert.equal(branch[0].hint, "feat/comms");
  assert.equal(branch[1].hint, "main");
  // The tree asks "which folder am I browsing?", so the folder is its hint.
  const path = pickerOptions(workspaces, { hint: "path" });
  assert.deepEqual(path.map((o) => o.hint), ["~/picode/.worktrees/comms", "~/picode", "~/notes", "~/orikami"]);
  assert.deepEqual(path.map((o) => o.label), ["communication", "desktop", "notes", "ORIKAMI"]);
  // The folder is the row's title: a deep worktree path elides to
  // `~/picode/.worktrees/gg-wo…`, which disambiguates nothing.
  assert.equal(path[0].title, "/home/goat/picode/.worktrees/comms");
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

test("hintPath keeps a path whole when it fits and wears its tail when it does not", () => {
  assert.equal(hintPath("/home/goat/picode"), "~/picode");
  assert.equal(hintPath(""), "");
  // The tail names the folder; the head names nothing (`~/picode/.worktrees/tree-w…`).
  const deep = "/home/goat/picode/.worktrees/tree-workspace-picker/var/qa/twp/repos/alpha";
  assert.equal(hintPath(deep), "…/var/qa/twp/repos/alpha");
  assert.equal(hintPath(deep).length <= 30, true);
  assert.equal(hintPath(deep, 18), "…/twp/repos/alpha");
  // A single very long segment has no tail to keep: nothing is dropped, so it
  // comes back whole and the row's ellipsis takes it from there.
  assert.equal(hintPath("/home/goat/" + "x".repeat(60)), "…/" + "x".repeat(60));
});
