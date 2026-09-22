import { test } from "node:test";
import assert from "node:assert/strict";
import { newRequestURL, wantsPullRequest, workspaceRowMenu } from "./workspaceRowMenu.js";

const ids = (rows) => rows.map((r) => (r.sep ? "|" : r.id));
const row = (rows, id) => rows.find((r) => r.id === id);
const GH = { name: "origin", url: "https://github.com/o/r", host: "GitHub", kind: "github", defaultBranch: "main" };
const FOLDER = { id: "w1", path: "/home/x/notes" };
const LOCAL = { id: "w2", path: "/home/x/lab", git: { branch: "main" } };
const ON_MAIN = { id: "w3", path: "/home/x/app", git: { branch: "main" }, remote: GH };
const ON_FEATURE = { id: "w4", path: "/home/x/app", git: { branch: "feat/x" }, remote: GH };

// Decision table: the folder's kind decides the "out of PiCode" group.
// | folder                         | reveal | remote | pull request                 |
// | ------------------------------ | ------ | ------ | ---------------------------- |
// | plain folder                   | yes    | —      | —                            |
// | repository, no web remote      | yes    | —      | —                            |
// | GitHub, on the default branch  | yes    | yes    | — (nothing to propose)       |
// | GitHub, other branch, asking   | yes    | yes    | "Checking pull request…"     |
// | GitHub, other branch, PR found | yes    | yes    | Open pull request #N         |
// | GitHub, no PR / gh blocked     | yes    | yes    | Create pull request          |
// | GitLab / Bitbucket, other br.  | yes    | yes    | Create merge/pull request    |
// | other host                     | yes    | yes    | —                            |
test("a plain folder can be shown and copied, and has no web page", () => {
  assert.deepEqual(ids(workspaceRowMenu(FOLDER)), ["communication", "files", "|", "reveal", "copy-path", "|", "settings", "|", "remove"]);
});

test("a repository without a web remote adds only its git graph", () => {
  assert.deepEqual(ids(workspaceRowMenu(LOCAL)), ["communication", "files", "git-graph", "|", "reveal", "copy-path", "|", "settings", "|", "remove"]);
});

test("the default branch opens the repository and proposes nothing", () => {
  const rows = workspaceRowMenu(ON_MAIN);
  assert.deepEqual(ids(rows), ["communication", "files", "git-graph", "|", "reveal", "remote", "copy-path", "|", "settings", "|", "remove"]);
  assert.equal(row(rows, "remote").label, "Open on GitHub");
  assert.equal(row(rows, "remote").url, "https://github.com/o/r");
  assert.equal(wantsPullRequest(ON_MAIN), false);
});

test("a GitHub branch says it is asking, then opens or creates the pull request", () => {
  assert.equal(wantsPullRequest(ON_FEATURE), true);
  const asking = row(workspaceRowMenu(ON_FEATURE), "pr");
  assert.equal(asking.label, "Checking pull request…");
  assert.equal(asking.pending, true);

  const open = row(workspaceRowMenu(ON_FEATURE, { pr: { status: "ok", pr: { number: 12, url: "https://github.com/o/r/pull/12", state: "open" } } }), "pr");
  assert.equal(open.label, "Open pull request #12");
  assert.equal(open.url, "https://github.com/o/r/pull/12");
  const merged = row(workspaceRowMenu(ON_FEATURE, { pr: { status: "ok", pr: { number: 9, url: "u", state: "merged" } } }), "pr");
  assert.equal(merged.label, "Open pull request #9 · merged");

  for (const pr of [{ status: "none" }, { status: "blocked", reason: "gh-missing" }, null]) {
    const create = row(workspaceRowMenu(ON_FEATURE, { pr }), "pr");
    assert.equal(create.label, "Create pull request");
    assert.equal(create.url, "https://github.com/o/r/compare/feat%2Fx?expand=1");
  }
});

test("other hosts link their own new-request page, or none", () => {
  const gl = { ...ON_FEATURE, remote: { url: "https://gitlab.com/g/r", host: "GitLab", kind: "gitlab" } };
  assert.equal(row(workspaceRowMenu(gl), "pr").label, "Create merge request");
  assert.equal(wantsPullRequest(gl), false);
  const bb = { ...ON_FEATURE, remote: { url: "https://bitbucket.org/t/r", host: "Bitbucket", kind: "bitbucket" } };
  assert.equal(row(workspaceRowMenu(bb), "pr").url, "https://bitbucket.org/t/r/pull-requests/new?source=feat%2Fx");
  const self = { ...ON_FEATURE, remote: { url: "https://git.example.com/t/r", host: "git.example.com", kind: "" } };
  const rows = workspaceRowMenu(self);
  assert.equal(row(rows, "remote").label, "Open on git.example.com");
  assert.equal(row(rows, "pr"), undefined);
  assert.equal(newRequestURL(self.remote, "x"), "");
});

test("under WSL the file manager is Explorer and the path comes in both forms", () => {
  const rows = workspaceRowMenu({ ...FOLDER, winPath: "\\\\wsl.localhost\\Ubuntu\\home\\x\\notes" });
  assert.equal(row(rows, "reveal").label, "Show in Explorer");
  const copy = row(rows, "copy-path");
  assert.deepEqual(copy.sub.map((s) => [s.label, s.value]), [
    ["Linux path", "/home/x/notes"],
    ["Windows path", "\\\\wsl.localhost\\Ubuntu\\home\\x\\notes"],
  ]);
  assert.equal(row(workspaceRowMenu(FOLDER), "copy-path").value, "/home/x/notes");
});

test("order moves sit above Remove, only the moves that exist", () => {
  assert.deepEqual(ids(workspaceRowMenu(FOLDER, { hasAgents: true, canMoveUp: true, canMoveDown: true })),
    ["communication", "files", "sessions", "|", "reveal", "copy-path", "|", "settings", "|", "move-up", "move-down", "|", "remove"]);
  assert.deepEqual(ids(workspaceRowMenu(FOLDER, { canMoveDown: true })).slice(-4), ["|", "move-down", "|", "remove"]);
  assert.equal(workspaceRowMenu(FOLDER).at(-1).danger, true);
});
