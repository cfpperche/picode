import test from "node:test";
import assert from "node:assert/strict";
import {
  anchorFor, ownerExists, describeAnchor,
  inspectorLayout, maxInspectorWidth, clampInspectorWidth, defaultOpen,
  INSPECTOR_MIN, INSPECTOR_MAX, INSPECTOR_DEFAULT,
  filterRows, blockedMessage,
  readInspectorPrefs, writeInspectorPrefs,
  prTabLabel, prStateLabel, prChecksLabel, prReviewLabel, prBlockedAction,
  shellQuote, gitActionCommand, gitActions, branchChip, runFallbackNote,
  askableAgents, askChannelHint, askGitPrompt, askedNote,
} from "./inspector.js";

const ws = { id: "ws1", name: "picode", path: "/home/u/picode", git: { branch: "main", dirty: 2 }, agents: [{ id: "ag1", name: "Atlas", workPath: "/home/u/picode/.worktrees/x", git: { branch: "feat/x", dirty: 1 } }, { id: "ag2", name: "default" }] };
const ctx = {
  workspaces: [ws],
  freeAgents: [{ id: "free1", name: "Nova" }],
  terminals: [{ id: "t1", name: "QA", cwd: "/home/u/picode", git: { branch: "main", dirty: 0 } }],
  gitOwners: { "g:abc": { kind: "term", id: "t1", name: "QA" } },
  treeOwners: { "d:/home/u/picode": { kind: "workspace", id: "ws1", name: "picode" } },
};

test("anchorFor follows the selected tab's owner", () => {
  assert.deepEqual(anchorFor("ag1", ctx), { kind: "agent", id: "ag1" });
  assert.deepEqual(anchorFor("free1", ctx), { kind: "agent", id: "free1" });
  assert.deepEqual(anchorFor("t:t1", ctx), { kind: "term", id: "t1" });
  assert.deepEqual(anchorFor("f:a:ag1:src%2Fmain.go", ctx), { kind: "agent", id: "ag1" });
  assert.deepEqual(anchorFor("f:w:ws1:README.md", ctx), { kind: "workspace", id: "ws1" });
  assert.deepEqual(anchorFor("g:abc", ctx), { kind: "term", id: "t1" });
  assert.deepEqual(anchorFor("d:/home/u/picode", ctx), { kind: "workspace", id: "ws1" });
});

test("anchorFor keeps the last anchor on tabs without a folder and returns it by identity", () => {
  const last = { kind: "agent", id: "ag1" };
  assert.equal(anchorFor("x:inbox", ctx, last), last);
  assert.equal(anchorFor(null, ctx, last), last);
  assert.equal(anchorFor("ag1", ctx, last), last, "same owner → same object");
  assert.equal(anchorFor("x:inbox", ctx, null), null);
});

test("an app tab that publishes a subject anchors the rail on it", () => {
  const subjects = { "x:canvas": { kind: "term", id: "t1" } };
  const withSubject = { ...ctx, appSubjects: subjects };
  const last = { kind: "agent", id: "ag1" };
  // The published subject wins over the anchor the rail was carrying.
  assert.deepEqual(anchorFor("x:canvas", withSubject, last), { kind: "term", id: "t1" });
  // Only for the tab that published it: another app keeps the last anchor.
  assert.equal(anchorFor("x:inbox", withSubject, last), last);
  // A subject the fleet no longer has clears the rail rather than showing a
  // folder nobody owns — the same rule every other owner obeys.
  assert.equal(anchorFor("x:canvas", { ...ctx, appSubjects: { "x:canvas": { kind: "term", id: "gone" } } }, last), null);
  // Cleared (the app has nothing in focus): back to keeping the last anchor.
  assert.equal(anchorFor("x:canvas", { ...ctx, appSubjects: { "x:canvas": null } }, last), last);
  assert.equal(anchorFor("x:canvas", ctx, last), last, "no map at all is the old behaviour");
});

test("anchorFor clears an owner that no longer exists", () => {
  assert.equal(anchorFor("t:gone", ctx, { kind: "agent", id: "ag1" }), null);
  assert.equal(anchorFor("x:inbox", ctx, { kind: "term", id: "gone" }), null);
  assert.equal(anchorFor("ghost", ctx, { kind: "agent", id: "ghost" }), null);
  assert.equal(anchorFor("g:unknown", ctx, { kind: "agent", id: "ag1" }).id, "ag1", "an unmapped graph tab keeps the last anchor");
  assert.equal(ownerExists({ kind: "workspace", id: "ws1" }, ctx), true);
  assert.equal(ownerExists({ kind: "workspace", id: "nope" }, ctx), false);
});

test("describeAnchor reads the live identity line", () => {
  const agent = describeAnchor({ kind: "agent", id: "ag1" }, ctx);
  assert.equal(agent.name, "Atlas");
  assert.equal(agent.path, "/home/u/picode/.worktrees/x");
  assert.equal(agent.git.branch, "feat/x");
  const named = describeAnchor({ kind: "agent", id: "ag2" }, ctx);
  assert.equal(named.name, "picode", "a default-named agent wears the workspace name");
  assert.equal(named.git.branch, "main");
  const term = describeAnchor({ kind: "term", id: "t1" }, ctx);
  assert.equal(term.name, "QA");
  assert.equal(term.path, "/home/u/picode");
  const wk = describeAnchor({ kind: "workspace", id: "ws1" }, ctx);
  assert.equal(wk.name, "picode");
  assert.equal(describeAnchor({ kind: "term", id: "gone" }, ctx), null);
  assert.equal(describeAnchor(null, ctx), null);
});

test("inspectorLayout protects the conversation column and hides on narrow shells", () => {
  assert.deepEqual(inspectorLayout({ appWidth: 1440, sidebarWidth: 244, inspectorWidth: 320, wantOpen: true }), { shown: true, reason: "", width: 320 });
  assert.deepEqual(inspectorLayout({ appWidth: 1280, sidebarWidth: 244, inspectorWidth: 320, wantOpen: true }), { shown: true, reason: "", width: 320 });
  // 1200 − 244 − 688 = 268: the rail shrinks to fit instead of vanishing.
  assert.deepEqual(inspectorLayout({ appWidth: 1200, sidebarWidth: 244, inspectorWidth: 320, wantOpen: true }), { shown: true, reason: "", width: 268 });
  // 1100 − 244 − 688 = 168 < 260: even the minimum rail would squeeze the chat.
  assert.deepEqual(inspectorLayout({ appWidth: 1100, sidebarWidth: 244, inspectorWidth: 320, wantOpen: true }), { shown: false, reason: "squeezed", width: 320 });
  assert.deepEqual(inspectorLayout({ appWidth: 1440, sidebarWidth: 244, inspectorWidth: 320, wantOpen: false }), { shown: false, reason: "closed", width: 320 });
  assert.deepEqual(inspectorLayout({ appWidth: 1440, sidebarWidth: 244, inspectorWidth: 320, wantOpen: true, narrow: true }), { shown: false, reason: "narrow", width: 320 });
  assert.equal(inspectorLayout({ appWidth: 0, wantOpen: true }).shown, true, "an unmeasured shell does not hide the rail");
  assert.equal(inspectorLayout({ appWidth: 1440, sidebarWidth: 244, inspectorWidth: 9000, wantOpen: true }).width, 508, "clamped to the rail max, then to the room left (1440 − 244 − 688)");
  assert.equal(inspectorLayout({ appWidth: 2000, sidebarWidth: 244, inspectorWidth: 9000, wantOpen: true }).width, INSPECTOR_MAX);
});

test("width helpers clamp to the rail's range", () => {
  assert.equal(maxInspectorWidth(1440, 244), 508);
  assert.equal(maxInspectorWidth(2000, 244), INSPECTOR_MAX);
  assert.equal(maxInspectorWidth(0, 244), INSPECTOR_MAX);
  assert.equal(maxInspectorWidth(900, 244), 0);
  assert.equal(clampInspectorWidth(100), INSPECTOR_MIN);
  assert.equal(clampInspectorWidth(999), INSPECTOR_MAX);
  assert.equal(clampInspectorWidth("abc"), INSPECTOR_DEFAULT);
  assert.equal(clampInspectorWidth(300.4), 300);
  assert.equal(defaultOpen(1440), true);
  assert.equal(defaultOpen(1439), false);
});

test("filterRows keeps matches and the folders above them", () => {
  const rows = [
    { path: "src", name: "src", depth: 0, isDir: true },
    { path: "src/main.go", name: "main.go", depth: 1, isDir: false },
    { path: "src/util.go", name: "util.go", depth: 1, isDir: false },
    { path: "docs", name: "docs", depth: 0, isDir: true },
    { path: "docs/main.md", name: "main.md", depth: 1, isDir: false },
    { path: "README.md", name: "README.md", depth: 0, isDir: false },
  ];
  assert.equal(filterRows(rows, ""), rows);
  assert.deepEqual(filterRows(rows, "main").map((r) => r.path), ["src", "src/main.go", "docs", "docs/main.md"]);
  assert.deepEqual(filterRows(rows, "READ").map((r) => r.path), ["README.md"]);
  assert.deepEqual(filterRows(rows, "nothing"), []);
});

test("blockedMessage names the moved terminal, and only that", () => {
  assert.equal(blockedMessage("term", "/home/goat/picode/web"), "This terminal moved to ~/picode/web.");
  assert.equal(blockedMessage("term", ""), "This folder changed.");
  assert.equal(blockedMessage("agent", "/x"), "This folder changed.");
});

test("inspector prefs round-trip through storage with safe defaults", () => {
  const mem = new Map();
  const storage = { getItem: (k) => (mem.has(k) ? mem.get(k) : null), setItem: (k, v) => mem.set(k, String(v)) };
  assert.deepEqual(readInspectorPrefs(storage), { open: null, width: INSPECTOR_DEFAULT, tab: "changes", run: false });
  writeInspectorPrefs({ open: false, width: 9999, tab: "files" }, storage);
  assert.deepEqual(readInspectorPrefs(storage), { open: false, width: INSPECTOR_MAX, tab: "files", run: false });
  writeInspectorPrefs({ open: true, tab: "bogus", run: true }, storage);
  assert.deepEqual(readInspectorPrefs(storage), { open: true, width: INSPECTOR_MAX, tab: "files", run: true });
  const broken = { getItem: () => { throw new Error("quota"); }, setItem: () => { throw new Error("quota"); } };
  assert.deepEqual(readInspectorPrefs(broken), { open: null, width: INSPECTOR_DEFAULT, tab: "changes", run: false });
  assert.doesNotThrow(() => writeInspectorPrefs({ open: true }, broken));
  assert.deepEqual(readInspectorPrefs(null), { open: null, width: INSPECTOR_DEFAULT, tab: "changes", run: false });
  assert.equal(runFallbackNote("Atlas is mid-turn in this repository."), "Atlas is mid-turn in this repository. The command is ready in the terminal; press Enter to run it.");
  assert.equal(runFallbackNote(""), "The command is ready in the terminal; press Enter to run it.");
});

test("pull request labels read like a person, not a payload", () => {
  assert.equal(prTabLabel(null), "PR");
  assert.equal(prTabLabel({ status: "none" }), "PR");
  assert.equal(prTabLabel({ status: "ok", pr: { number: 3981 } }), "PR #3981");
  assert.equal(prStateLabel({ state: "open", draft: false }), "Open");
  assert.equal(prStateLabel({ state: "open", draft: true }), "Draft");
  assert.equal(prStateLabel({ state: "merged" }), "Merged");
  assert.equal(prStateLabel({ state: "closed" }), "Closed");
  assert.equal(prStateLabel(null), "");
  assert.equal(prChecksLabel({ total: 5, passed: 2, failed: 1, pending: 1, skipped: 1 }), "1 failed · 1 pending · 2 passed · 1 skipped");
  assert.equal(prChecksLabel({ total: 3, passed: 3 }), "3 passed");
  assert.equal(prChecksLabel({ total: 0 }), "No checks");
  assert.equal(prChecksLabel(null), "No checks");
  assert.equal(prReviewLabel("APPROVED"), "Approved");
  assert.equal(prReviewLabel("CHANGES_REQUESTED"), "Changes requested");
  assert.equal(prReviewLabel("REVIEW_REQUIRED"), "Review required");
  assert.equal(prReviewLabel(""), "No review yet");
  assert.equal(prBlockedAction("gh-missing"), "install");
  assert.equal(prBlockedAction("gh-unauth"), "login");
  assert.equal(prBlockedAction("no-remote"), "");
  assert.equal(prBlockedAction("gh-error"), "retry");
});

test("inspector prefs accept the pr tab", () => {
  const mem = new Map();
  const storage = { getItem: (k) => (mem.has(k) ? mem.get(k) : null), setItem: (k, v) => mem.set(k, String(v)) };
  writeInspectorPrefs({ tab: "pr" }, storage);
  assert.equal(readInspectorPrefs(storage).tab, "pr");
});

test("git actions type the exact command a person would, and quote what a shell could read", () => {
  assert.equal(shellQuote("fix: it's done"), "'fix: it'\\''s done'");
  assert.equal(gitActionCommand("fetch"), "git fetch --prune");
  assert.equal(gitActionCommand("pull"), "git pull --ff-only");
  assert.equal(gitActionCommand("push", { branch: "feat/x", upstream: "origin/feat/x" }), "git push");
  assert.equal(gitActionCommand("push", { branch: "feat/x", upstream: "" }), "git push -u origin feat/x");
  assert.equal(gitActionCommand("push", { branch: "odd$name", upstream: "" }), "git push -u origin 'odd$name'");
  assert.equal(gitActionCommand("commit", { message: "web: add rail" }), "git add -A && git commit -m 'web: add rail'");
  assert.equal(gitActionCommand("commit-push", { message: "it's", branch: "main", upstream: "" }), "git add -A && git commit -m 'it'\\''s' && git push -u origin main");
  assert.equal(gitActionCommand("pr"), "gh pr create --fill");
  assert.equal(gitActionCommand("nope"), "");
  assert.deepEqual(gitActions({ git: true, branch: "main", upstream: "origin/main" }), ["fetch", "pull", "push", "commit", "commit-push"]);
  assert.deepEqual(gitActions({ git: true, branch: "abc123", detached: true }), ["fetch", "commit"]);
  assert.deepEqual(gitActions({ git: false }), []);
});

test("branchChip names the checkout and its distance from upstream", () => {
  assert.equal(branchChip({ git: false }), null);
  const level = branchChip({ git: true, branch: "main", upstream: "origin/main", ahead: 0, behind: 0 });
  assert.equal(level.unpublished, false);
  assert.match(level.title, /level with origin\/main/);
  const far = branchChip({ git: true, branch: "feat/x", upstream: "origin/feat/x", ahead: 2, behind: 1, worktree: "x" });
  assert.equal(far.ahead, 2);
  assert.equal(far.behind, 1);
  assert.equal(far.worktree, "x");
  assert.match(far.title, /2 ahead and 1 behind origin\/feat\/x/);
  const fresh = branchChip({ git: true, branch: "feat/new", ahead: 0, behind: 0 });
  assert.equal(fresh.unpublished, true);
  assert.match(fresh.title, /no upstream/);
  const det = branchChip({ git: true, branch: "abc1234", detached: true });
  assert.equal(det.detached, true);
  assert.equal(det.unpublished, false);
});

test("askableAgents offers running agents of this repository, anchored first, at most three", () => {
  const top = "/w/app";
  const ws = {
    id: "ws1", name: "app", path: top,
    agents: [
      { id: "a-stopped", name: "Off", running: false, mode: "stopped" },
      { id: "a-tui", name: "Zed", running: true, mode: "interactive" },
      { id: "a-rpc", name: "Atlas", running: true, mode: "managed", streaming: true },
      { id: "a-sub", name: "Sub", running: true, mode: "managed", workPath: top + "/pkg" },
      { id: "a-else", name: "Elsewhere", running: true, mode: "managed", workPath: "/w/other" },
      { id: "a-4th", name: "Zulu", running: true, mode: "managed" },
    ],
  };
  const free = [{ id: "f1", name: "Nova", running: true, mode: "managed", workPath: top + "/tools" }, { id: "f2", name: "Rigel", running: true, mode: "managed" }];
  const none = askableAgents({ workspaces: [ws], freeAgents: free }, { root: "" });
  assert.deepEqual(none, []);
  const got = askableAgents({ workspaces: [ws], freeAgents: free }, { root: top + "/pkg", repoRoot: top, anchor: { kind: "agent", id: "a-sub" } });
  assert.deepEqual(got.map((a) => a.id), ["a-sub", "a-rpc", "f1"]);
  assert.equal(got[1].streaming, true);
  assert.equal(got[0].path, top + "/pkg");
  const term = askableAgents({ workspaces: [ws], freeAgents: [] }, { root: top, anchor: { kind: "term", id: "t1" } });
  assert.deepEqual(term.map((a) => a.name), ["Atlas", "Sub", "Zed"]);
  assert.equal(term.find((a) => a.name === "Zed").mode, "interactive");
  assert.equal(askChannelHint(term[2]), "in its terminal");
  assert.equal(askChannelHint(term[0]), "after its turn");
  assert.equal(askChannelHint(term[1]), "");
  // A trailing slash or a Windows separator does not break containment.
  assert.equal(askableAgents({ workspaces: [{ ...ws, path: top + "/" }] }, { root: top }).length, 4 - 1);
});

test("askGitPrompt names the folder, the branch, the action and its rules", () => {
  const ctx = { root: "/w/app", branch: "main", upstream: "origin/main" };
  assert.equal(askGitPrompt("fetch", ctx), "In /w/app, run git fetch --prune and tell me how the branch main stands against its upstream.");
  assert.match(askGitPrompt("pull", ctx), /^In \/w\/app, bring the branch main up to date .*fast-forward only .*stop and tell me why/);
  assert.equal(askGitPrompt("push", ctx), "In /w/app, push the branch main to its upstream (origin/main). Never force-push; if the push is rejected, tell me why.");
  assert.equal(askGitPrompt("push", { root: "/w/app", branch: "feat/x" }), "In /w/app, push the branch feat/x and set its upstream (origin/feat/x). Never force-push; if the push is rejected, tell me why.");
  assert.equal(askGitPrompt("commit", { ...ctx, message: "fix: typo" }), 'In /w/app, commit the current changes on the branch main with this message: "fix: typo". Do not push.');
  assert.equal(askGitPrompt("commit", ctx), "In /w/app, review the uncommitted changes on the branch main and commit them with a fitting message. Do not push.");
  assert.match(askGitPrompt("commit-push", { ...ctx, message: "m" }), /with this message: "m"\. Then push the branch main to its upstream \(origin\/main\), never with force\.$/);
  assert.match(askGitPrompt("pr", ctx), /create a pull request for the branch main with gh pr create.*give me its URL\.$/);
  assert.equal(askGitPrompt("fetch", {}), "In this folder, run git fetch --prune and tell me how the current branch stands against its upstream.");
  assert.equal(askGitPrompt("nope", ctx), "");
});

test("askedNote says when the agent acts", () => {
  assert.equal(askedNote("Atlas", "push", { mode: "managed", via: "queue", busy: false }), "Asked Atlas to push.");
  assert.equal(askedNote("Atlas", "commit-push", { mode: "managed", via: "queue", busy: true }), "Queued for Atlas: commit and push after its current turn.");
  assert.equal(askedNote("Zed", "pr", { mode: "interactive", via: "paste" }), "Asked Zed to open a pull request in its terminal.");
  assert.equal(askedNote("", "fetch", null), "Asked the agent to fetch.");
});
