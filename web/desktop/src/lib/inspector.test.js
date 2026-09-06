import test from "node:test";
import assert from "node:assert/strict";
import {
  anchorFor, ownerExists, describeAnchor,
  inspectorLayout, maxInspectorWidth, clampInspectorWidth, defaultOpen,
  INSPECTOR_MIN, INSPECTOR_MAX, INSPECTOR_DEFAULT,
  groupChanges, changeTotals, scopeChanges, normalizeTouched,
  compactCount, totalsLabel, filterRows, blockedMessage,
  readInspectorPrefs, writeInspectorPrefs,
  prTabLabel, prStateLabel, prChecksLabel, prReviewLabel, prBlockedAction,
} from "./inspector.js";
import { flattenTree } from "./fileTree.js";

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

test("groupChanges builds a folder tree with summed counts", () => {
  const changes = [
    { path: "web/src/b.js", kind: "modified", add: 10, del: 2 },
    { path: "web/src/a.js", kind: "added", add: 5, del: 0 },
    { path: "web/img/logo.png", kind: "untracked", add: 0, del: 0, binary: true },
    { path: "README.md", kind: "modified", add: 1, del: 1 },
    { path: "cmd/main.go", kind: "deleted", add: 0, del: 40 },
  ];
  const { levels, dirs, stats } = groupChanges(changes);
  assert.deepEqual([...dirs].sort(), ["cmd", "web", "web/img", "web/src"]);
  assert.deepEqual(levels[""].dirs.map((d) => d.name), ["cmd", "web"]);
  assert.deepEqual(levels[""].files.map((f) => f.name), ["README.md"]);
  assert.deepEqual(levels["web"].dirs.map((d) => d.path), ["web/img", "web/src"]);
  assert.deepEqual(levels["web/src"].files.map((f) => f.name), ["a.js", "b.js"]);
  assert.deepEqual(stats.get("web"), { add: 15, del: 2, files: 3, binary: false });
  assert.deepEqual(stats.get("web/src"), { add: 15, del: 2, files: 2, binary: false });
  assert.deepEqual(stats.get("web/img"), { add: 0, del: 0, files: 1, binary: false });
  assert.equal(stats.get("web/img/logo.png").binary, true);
  assert.equal(stats.get("cmd/main.go").kind, "deleted");
  // The levels feed flattenTree unchanged: every folder expanded lists all five files.
  const rows = flattenTree(levels, dirs);
  assert.deepEqual(rows.map((r) => r.path), ["cmd", "cmd/main.go", "web", "web/img", "web/img/logo.png", "web/src", "web/src/a.js", "web/src/b.js", "README.md"]);
  assert.deepEqual(groupChanges([]).levels, { "": { dirs: [], files: [] } });
});

test("changeTotals, scopeChanges and normalizeTouched agree on root-relative paths", () => {
  const changes = [{ path: "a.js", add: 3, del: 1 }, { path: "src/b.js", add: 0, del: 2 }, { path: "bin.png", binary: true }];
  assert.deepEqual(changeTotals(changes), { add: 3, del: 3, files: 3 });
  assert.equal(scopeChanges(changes, null), changes);
  assert.deepEqual(scopeChanges(changes, new Set(["src/b.js"])).map((c) => c.path), ["src/b.js"]);
  assert.deepEqual(scopeChanges(changes, new Set()), []);
  const touched = normalizeTouched(["./a.js", "/home/u/repo/src/b.js", "/elsewhere/c.js", "src/", "", "../up.js", "/home/u/repo"], "/home/u/repo/");
  assert.deepEqual([...touched].sort(), ["a.js", "src", "src/b.js"]);
  assert.deepEqual([...normalizeTouched(["/abs/x.js"], "")], [], "absolute paths need a root to be relative to");
});

test("compactCount and totalsLabel print the benchmark shape", () => {
  assert.equal(compactCount(684), "684");
  assert.equal(compactCount(1900), "1.9k");
  assert.equal(compactCount(1000), "1k");
  assert.equal(compactCount(12345), "12k");
  assert.equal(compactCount(1250000), "1.3M");
  assert.equal(compactCount(-3), "0");
  assert.equal(totalsLabel({ add: 1900, del: 684 }), "+1.9k −684");
  assert.equal(totalsLabel({ add: 0, del: 0 }), "");
  assert.equal(totalsLabel({ add: 4, del: 0 }), "+4");
  assert.equal(totalsLabel(null), "");
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
  assert.deepEqual(readInspectorPrefs(storage), { open: null, width: INSPECTOR_DEFAULT, tab: "changes" });
  writeInspectorPrefs({ open: false, width: 9999, tab: "files" }, storage);
  assert.deepEqual(readInspectorPrefs(storage), { open: false, width: INSPECTOR_MAX, tab: "files" });
  writeInspectorPrefs({ open: true, tab: "bogus" }, storage);
  assert.deepEqual(readInspectorPrefs(storage), { open: true, width: INSPECTOR_MAX, tab: "files" });
  const broken = { getItem: () => { throw new Error("quota"); }, setItem: () => { throw new Error("quota"); } };
  assert.deepEqual(readInspectorPrefs(broken), { open: null, width: INSPECTOR_DEFAULT, tab: "changes" });
  assert.doesNotThrow(() => writeInspectorPrefs({ open: true }, broken));
  assert.deepEqual(readInspectorPrefs(null), { open: null, width: INSPECTOR_DEFAULT, tab: "changes" });
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
