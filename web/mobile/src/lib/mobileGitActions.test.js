import test from "node:test";
import assert from "node:assert/strict";
import { commitMessageSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { gitActions, gitActionCommand, askableAgents, askGitPrompt, askedNote } from "./git/actions.js";

// Decision table: repository state -> actions offered before delivery. Run
// readiness is checked by the host/server, never inferred from this snapshot.
for (const [name, status, offered] of [
  ["unloaded", null, []],
  ["plain folder", { git: false }, []],
  ["missing branch", { git: true }, []],
  ["published branch", { git: true, branch: "main", upstream: "origin/main" }, ["fetch", "pull", "push", "commit", "commit-push"]],
  ["unpublished branch", { git: true, branch: "feat/mobile" }, ["fetch", "pull", "push", "commit", "commit-push"]],
  ["detached head", { git: true, branch: "a".repeat(40), detached: true }, ["fetch", "commit"]],
]) {
  test(`Git action availability: ${name}`, () => assert.deepEqual(gitActions(status), offered));
}

for (const [action, options, command] of [
  ["fetch", {}, "git fetch --prune"],
  ["pull", {}, "git pull --ff-only"],
  ["push", { branch: "main", upstream: "upstream/main" }, "git push"],
  ["push", { branch: "feat/mobile" }, "git push -u origin feat/mobile"],
  ["commit", { message: "mobile: save edits" }, "git add -A && git commit -m 'mobile: save edits'"],
  ["commit-push", { message: "save", branch: "feat/mobile" }, "git add -A && git commit -m 'save' && git push -u origin feat/mobile"],
  ["commit-push", { message: "save", branch: "main", upstream: "upstream/main" }, "git add -A && git commit -m 'save' && git push"],
  ["pr", {}, "gh pr create --fill"],
  ["reset", {}, ""],
]) {
  test(`prepare/run command policy: ${action} ${options.upstream || options.branch || ""}`, () => {
    const actual = gitActionCommand(action, options);
    assert.equal(actual, command);
    assert.doesNotMatch(actual, /[\r\n\0]|--force|--hard|--rebase/);
  });
}

for (const [branch, argument] of [
  ["release+docs", "release+docs"],
  ["feat/$HOME", "'feat/$HOME'"],
  ["feat/$(printf-oops)", "'feat/$(printf-oops)'"],
  ["feat/one;printf-oops", "'feat/one;printf-oops'"],
  ["feat/it's", "'feat/it'\\''s'"],
]) {
  test(`publishing quotes the literal branch ${branch}`, () => {
    assert.equal(gitActionCommand("push", { branch }), `git push -u origin ${argument}`);
  });
}

test("commit shell syntax remains one literal message in both command paths", () => {
  const message = "it's $HOME; $(printf-oops) `printf-oops` && false";
  const quoted = "'it'\\''s $HOME; $(printf-oops) `printf-oops` && false'";
  const parsed = parseForm(commitMessageSchema, { message });
  assert.equal(parsed.ok, true);
  assert.equal(gitActionCommand("commit", parsed.value), `git add -A && git commit -m ${quoted}`);
  assert.equal(gitActionCommand("commit-push", { ...parsed.value, branch: "main", upstream: "origin/main" }), `git add -A && git commit -m ${quoted} && git push`);
});

for (const [name, message, accepted] of [
  ["trimmed message", "  mobile: save edits  ", true],
  ["200 characters", "a".repeat(200), true],
  ["201 characters", "a".repeat(201), false],
  ["empty", "", false],
  ["only whitespace", "   ", false],
  ["leading option", "  --amend", false],
  ["newline", "first\nsecond", false],
  ["carriage return", "first\rsecond", false],
  ["tab", "first\tsecond", false],
  ["NUL", "first\0second", false],
  ["DEL", "first\x7fsecond", false],
]) {
  test(`terminal commit validation: ${name}`, () => {
    const parsed = parseForm(commitMessageSchema, { message });
    assert.equal(parsed.ok, accepted);
    if (accepted) assert.equal(parsed.value.message, message.trim());
    else { assert.equal(parsed.value, null); assert.ok(parsed.error); }
  });
}

// Decision table: only a running Pi channel inside this repository may be
// offered. Sharing a pathname prefix is not repository membership.
for (const [name, agent, path, expected] of [
  ["managed idle", { mode: "managed", running: true }, "/repo", true],
  ["managed busy", { mode: "managed", running: true, streaming: true }, "/repo/src", true],
  ["interactive", { mode: "interactive", running: true }, "/repo", true],
  ["stopped managed", { mode: "managed", running: false }, "/repo", false],
  ["stopped mode", { mode: "stopped", running: true }, "/repo", false],
  ["Agent CLI", { mode: "terminal", running: true }, "/repo", false],
  ["missing mode", { running: true }, "/repo", false],
  ["outside repository", { mode: "managed", running: true }, "/elsewhere", false],
  ["pathname prefix only", { mode: "managed", running: true }, "/repo-other", false],
  ["free agent without a folder", { mode: "managed", running: true }, "", false],
]) {
  test(`Ask eligibility: ${name}`, () => {
    const available = askableAgents({ freeAgents: [{ id: "a", name: "Agent", ...agent, workPath: path }] }, { root: "/repo" });
    assert.deepEqual(available.map(a => a.id), expected ? ["a"] : []);
  });
}

test("Ask uses repository scope, workspace fallback and an explicit agent work folder", () => {
  const workspaces = [{ name: "Project", path: "/repo", agents: [
    { id: "default", name: "default", running: true, mode: "managed" },
    { id: "sub", name: "Sub", running: true, mode: "managed", workPath: "/repo/pkg" },
    { id: "moved", running: true, mode: "managed", workPath: "/other" },
  ] }];
  const available = askableAgents({ workspaces }, { root: "/repo/pkg", repoRoot: "/repo" });
  assert.deepEqual(available.map(a => a.id), ["default", "sub"]);
  assert.equal(available[0].name, "Project");
  assert.equal(available[0].path, "/repo");
  assert.deepEqual(askableAgents({ workspaces }, {}), []);
});

test("Ask keeps the anchored agent visible within the three-agent limit", () => {
  const freeAgents = ["Alpha", "Beta", "Gamma", "Zulu"].map(name => ({ id: name, name, mode: "managed", running: true, workPath: "/repo" }));
  assert.deepEqual(askableAgents({ freeAgents }, { root: "/repo", anchor: { kind: "agent", id: "Zulu" } }).map(a => a.id), ["Zulu", "Alpha", "Beta"]);
});

test("Ask containment normalizes Windows separators and trailing slashes", () => {
  const freeAgents = [{ id: "a", mode: "managed", running: true, workPath: "C:\\repo\\src\\" }, { id: "b", mode: "managed", running: true, workPath: "C:\\repo-other" }];
  assert.deepEqual(askableAgents({ freeAgents }, { root: "C:/repo/" }).map(a => a.id), ["a"]);
});

test("Ask preserves explicit folder and branch plus each action's safety policy", () => {
  const options = { root: "/repo/mobile", branch: "feat/mobile", upstream: "origin/feat/mobile" };
  for (const action of ["fetch", "pull", "push", "commit", "commit-push", "pr"]) {
    const prompt = askGitPrompt(action, options);
    assert.ok(prompt.startsWith("In /repo/mobile,"));
    assert.ok(prompt.includes("the branch feat/mobile"));
  }
  assert.match(askGitPrompt("pull", options), /fast-forward only.*stop.*instead of merging or rebasing/);
  assert.match(askGitPrompt("push", options), /upstream \(origin\/feat\/mobile\).*Never force-push/);
  assert.match(askGitPrompt("push", { root: "/repo", branch: "new" }), /set its upstream \(origin\/new\)/);
  assert.match(askGitPrompt("commit", options), /review the uncommitted changes.*fitting message.*Do not push/);
  assert.match(askGitPrompt("commit", { ...options, message: "  reviewed message  " }), /with this message: "reviewed message".*Do not push/);
  assert.match(askGitPrompt("commit-push", options), /Then push.*never with force/);
  assert.match(askGitPrompt("pr", options), /gh pr create.*give me its URL/);
  assert.equal(askGitPrompt("reset", options), "");
});

for (const [name, result, expected] of [
  ["managed idle", { mode: "managed", busy: false }, "Asked Atlas to commit."],
  ["managed busy", { mode: "managed", busy: true }, "Queued for Atlas: commit after its current turn."],
  ["interactive", { mode: "interactive", busy: true }, "Asked Atlas to commit in its terminal."],
]) {
  test(`Ask receipt reports delivery, not Git completion: ${name}`, () => assert.equal(askedNote("Atlas", "commit", result), expected));
}
