import assert from "node:assert/strict";
import { test } from "node:test";
import { pickerOptions, currentWorkspaceId, triggerLabel, openRepoKeys, pickAction } from "./gitWorkspacePicker.js";

const wsRepo = (name, path, branch, id) => ({
  id: id || "w-" + name,
  name,
  path,
  git: branch ? { branch } : null,
  agents: [],
});

const desktop = wsRepo("desktop", "/home/goat/picode", "main");
const comms = wsRepo("communication", "/home/goat/picode/.worktrees/comms", "feat/comms");
const notes = wsRepo("notes", "/home/goat/notes", "");           // folder, not a repository
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

test("currentWorkspaceId answers for every owner kind", () => {
  const terminals = [
    { id: "t-desktop", workspaceId: "w-desktop" },
    { id: "t-free" },
  ];
  const freeAgents = [{ id: "a-free" }];
  const withAgents = [
    { ...desktop, agents: [{ id: "a-desktop" }] },
    { ...comms, agents: [] },
  ];
  const lists = { workspaces: withAgents, freeAgents, terminals };
  assert.equal(currentWorkspaceId({ kind: "workspace", id: "w-desktop" }, lists), "w-desktop");
  assert.equal(currentWorkspaceId({ kind: "agent", id: "a-desktop" }, lists), "w-desktop");
  assert.equal(currentWorkspaceId({ kind: "term", id: "t-desktop" }, lists), "w-desktop");
  // Neither a free agent nor a free terminal belongs to a workspace.
  assert.equal(currentWorkspaceId({ kind: "agent", id: "a-free" }, lists), "");
  assert.equal(currentWorkspaceId({ kind: "term", id: "t-free" }, lists), "");
  // A workspace that was removed while the tab stayed open names nothing.
  assert.equal(currentWorkspaceId({ kind: "workspace", id: "w-gone" }, lists), "");
  assert.equal(currentWorkspaceId(null, lists), "");
});

test("triggerLabel prefers the workspace, falls back to the repository name", () => {
  assert.equal(triggerLabel(desktop, "picode"), "desktop");
  assert.equal(triggerLabel(null, "picode"), "picode");
  // An unnamed workspace still names something: its folder.
  assert.equal(triggerLabel({ id: "w", path: "/home/goat/picode" }, "picode"), "/home/goat/picode");
  assert.equal(triggerLabel(null, ""), "");
});

test("openRepoKeys drops provisional tabs and everything that is not a graph", () => {
  assert.deepEqual(
    openRepoKeys(["g:/home/goat/picode/.git", "g:@t:desktop-51c42d", "t:desktop-51c42d", "d:/home/goat/picode", ""]),
    ["/home/goat/picode/.git"],
  );
  assert.deepEqual(openRepoKeys(null), []);
});

test("pickAction: same repository keeps the tab, a new one renames it, an open one adopts it", () => {
  const key = "/home/goat/picode/.git";
  // Sibling worktrees of one repository: the tab stays, the owner moves.
  assert.equal(pickAction(key, key, [key]), "same");
  // Another repository with no tab: this tab becomes it (one tab per repo).
  assert.equal(pickAction("/home/goat/orikami/.git", key, [key]), "rename");
  // Another repository already on the strip: select it, hand it the pick.
  assert.equal(pickAction("/home/goat/orikami/.git", key, [key, "/home/goat/orikami/.git"]), "adopt");
  // Nothing resolved is nothing to do — never a retarget.
  assert.equal(pickAction("", key, [key]), "none");
});
