import assert from "node:assert/strict";
import { test } from "node:test";
import {
	clipEnd,
	clipStart,
	countLines,
	cycleFocus,
	extractToolPath,
	isBinary,
	layout,
	MAX_DIFF_LINES,
	mergeChanges,
	parseCommand,
	parseNumstat,
	parseUnifiedDiff,
	parseUntracked,
	pickFocus,
	renameParts,
	resolveFocusArg,
	shouldRefresh,
	statusText,
	totals,
	unquote,
	untrackedChange,
	type FileChange,
	type Model,
} from "../src/logic.ts";

const tracked = (path: string, added: number, removed: number): FileChange => ({ path, added, removed, untracked: false, binary: false });

test("refresh: only the tools that change the tree; unknown tools never trigger git", () => {
	assert.equal(shouldRefresh("edit"), true);
	assert.equal(shouldRefresh("bash"), true);
	assert.equal(shouldRefresh("read"), false);
	assert.equal(shouldRefresh("some_unknown_tool"), false);
});

test("tool path: edit/write/multiedit name the file, shell tools do not", () => {
	assert.equal(extractToolPath("edit", { path: " src/a.ts " }), "src/a.ts");
	assert.equal(extractToolPath("write", { path: "/abs/b.ts" }), "/abs/b.ts");
	assert.equal(extractToolPath("bash", { command: "touch x" }), undefined);
	assert.equal(extractToolPath("edit", { path: 3 }), undefined);
	assert.equal(extractToolPath("edit", null), undefined);
});

test("numstat: counts, binary rows, quoted paths, CRLF", () => {
	const files = parseNumstat('12\t3\tsrc/a.ts\n-\t-\timg/logo.png\r\n0\t0\t"sp ace.txt"\n\n');
	assert.deepEqual(files, [
		tracked("src/a.ts", 12, 3),
		{ path: "img/logo.png", added: 0, removed: 0, untracked: false, binary: true },
		tracked("sp ace.txt", 0, 0),
	]);
	assert.deepEqual(parseNumstat(""), []);
	assert.deepEqual(parseNumstat("garbage line"), []);
});

test("porcelain: only the ?? rows become untracked paths", () => {
	assert.deepEqual(parseUntracked(" M src/a.ts\n?? new.ts\n?? \"dir/with space.md\"\nA  staged.ts\n"), ["new.ts", "dir/with space.md"]);
	assert.equal(unquote('"a\\tb\\303\\251"'), "a\tbé");
	assert.equal(unquote("plain"), "plain");
});

test("line counting mirrors numstat; NUL bytes mean binary", () => {
	assert.equal(countLines(""), 0);
	assert.equal(countLines("a\nb\n"), 2);
	assert.equal(countLines("a\nb"), 2);
	assert.equal(countLines("\n"), 1);
	assert.equal(isBinary(new Uint8Array([65, 0, 66])), true);
	assert.equal(isBinary(new Uint8Array([65, 66])), false);
	assert.deepEqual(untrackedChange("n.ts", { lines: 4, binary: false }), { path: "n.ts", added: 4, removed: 0, untracked: true, binary: false });
	assert.deepEqual(untrackedChange("n.bin", { lines: 0, binary: true }), { path: "n.bin", added: 0, removed: 0, untracked: true, binary: true });
});

test("merge: one row per path, sorted, tracked wins over untracked", () => {
	const merged = mergeChanges([tracked("b.ts", 1, 1), tracked("a.ts", 2, 0)], [untrackedChange("c.ts", { lines: 3, binary: false }), untrackedChange("a.ts", { lines: 9, binary: false })]);
	assert.deepEqual(
		merged.map((f) => [f.path, f.added, f.untracked]),
		[
			["a.ts", 2, false],
			["b.ts", 1, false],
			["c.ts", 3, true],
		],
	);
	assert.deepEqual(totals(merged), { files: 3, added: 6, removed: 1 });
	assert.equal(statusText({ files: merged, focus: "a.ts", diff: [] }), "diff +6 -1");
	assert.equal(statusText({ files: [], focus: undefined, diff: [] }), "diff clean");
	assert.equal(statusText({ files: [], focus: undefined, diff: [], unavailable: true }), undefined);
	assert.equal(statusText(null), undefined);
});

test("renames: braced and plain forms give the new path and both sides", () => {
	assert.deepEqual(renameParts("src/{old => new}/f.ts"), { display: "src/new/f.ts", paths: ["src/old/f.ts", "src/new/f.ts"] });
	assert.deepEqual(renameParts("old.ts => new.ts"), { display: "new.ts", paths: ["old.ts", "new.ts"] });
	assert.deepEqual(renameParts("plain.ts"), { display: "plain.ts", paths: ["plain.ts"] });
});

test("focus: last touched, else the one shown, else the first; cycling wraps", () => {
	const files = [tracked("a.ts", 1, 0), tracked("b.ts", 1, 0), tracked("src/{x => y}/c.ts", 1, 0)];
	assert.equal(pickFocus(files, "b.ts", "a.ts"), "b.ts");
	assert.equal(pickFocus(files, "gone.ts", "a.ts"), "a.ts");
	assert.equal(pickFocus(files, undefined, undefined), "a.ts");
	assert.equal(pickFocus(files, "src/y/c.ts", undefined), "src/{x => y}/c.ts");
	assert.equal(pickFocus([], "a.ts", "a.ts"), undefined);
	assert.equal(cycleFocus(files, "a.ts", 1), "b.ts");
	assert.equal(cycleFocus(files, "a.ts", -1), "src/{x => y}/c.ts");
	assert.equal(cycleFocus(files, "missing", 1), "a.ts");
	assert.equal(cycleFocus([], "a.ts", 1), undefined);
});

test("focus arg: exact, unique tail, unique substring, else nothing", () => {
	const files = [tracked("src/a.ts", 1, 0), tracked("lib/a.ts", 1, 0), tracked("web/main.tsx", 1, 0)];
	assert.equal(resolveFocusArg(files, "./web/main.tsx"), "web/main.tsx");
	assert.equal(resolveFocusArg(files, "main.tsx"), "web/main.tsx");
	assert.equal(resolveFocusArg(files, "a.ts"), undefined); // two tails match
	assert.equal(resolveFocusArg(files, "lib/"), "lib/a.ts");
	assert.equal(resolveFocusArg(files, ""), undefined);
});

test("command words", () => {
	assert.deepEqual(parseCommand(""), { kind: "toggle" });
	assert.deepEqual(parseCommand("  OFF "), { kind: "close" });
	assert.deepEqual(parseCommand("on"), { kind: "open" });
	assert.deepEqual(parseCommand("next"), { kind: "next" });
	assert.deepEqual(parseCommand("p"), { kind: "prev" });
	assert.deepEqual(parseCommand("top"), { kind: "top" });
	assert.deepEqual(parseCommand("src/a.ts"), { kind: "focus", value: "src/a.ts" });
});

const SAMPLE = [
	"diff --git a/f.ts b/f.ts",
	"index 111..222 100644",
	"--- a/f.ts",
	"+++ b/f.ts",
	"@@ -1,3 +1,4 @@ header",
	" keep",
	"-old",
	"+new",
	"+added",
	" ",
	"\\ No newline at end of file",
	"",
].join("\n");

test("unified diff: numbers follow the hunk header; file header dropped; blank context kept", () => {
	const lines = parseUnifiedDiff(SAMPLE);
	assert.deepEqual(lines, [
		{ kind: "hunk", text: "@@ -1,3 +1,4 @@ header" },
		{ kind: "ctx", text: "keep", oldNo: 1, newNo: 1 },
		{ kind: "del", text: "old", oldNo: 2 },
		{ kind: "add", text: "new", newNo: 2 },
		{ kind: "add", text: "added", newNo: 3 },
		{ kind: "ctx", text: "", oldNo: 3, newNo: 4 },
		{ kind: "meta", text: "No newline at end of file" },
	]);
	assert.deepEqual(parseUnifiedDiff(""), []);
});

test("unified diff: capped with a trailing note", () => {
	const big = "@@ -1 +1 @@\n" + "+x\n".repeat(MAX_DIFF_LINES + 10);
	const lines = parseUnifiedDiff(big);
	assert.equal(lines.length, MAX_DIFF_LINES + 1);
	assert.match(lines[lines.length - 1]!.text, /more lines not shown/);
});

test("clipping: end keeps the head, start keeps the tail, both fit the width", () => {
	assert.equal(clipEnd("abcdef", 4), "abc…");
	assert.equal(clipEnd("abc", 4), "abc");
	assert.equal(clipStart("src/components/site/hero.tsx", 12), "…te/hero.tsx");
	assert.equal(clipStart("short", 12), "short");
	assert.equal(clipEnd("a🎉bcd", 3), "a🎉…"); // code points, not code units
});

function model(files: FileChange[], focus: string | undefined, diff = parseUnifiedDiff(SAMPLE)): Model {
	return { files, focus, diff };
}

test("layout: header, rows with right-aligned counts, separator, numbered hunks; every line fits", () => {
	const files = [tracked("src/a.ts", 12, 3), untrackedChange("new.ts", { lines: 5, binary: false }), { path: "logo.png", added: 0, removed: 0, untracked: false, binary: true }];
	const out = layout({ model: model(files, "src/a.ts"), scroll: 0, width: 40, height: 20 });
	assert.equal(out.lines[0], "3 files changed +17 -3");
	assert.equal(out.lines[1], "");
	assert.equal(out.lines[2], "▸ src/a.ts".padEnd(34) + "+12 -3");
	assert.equal(out.lines[3], "  new.ts".padEnd(34) + "+5 new");
	assert.equal(out.lines[4], "  logo.png".padEnd(37) + "bin");
	assert.equal(out.lines[5], "─".repeat(40));
	assert.equal(out.lines[6], "src/a.ts");
	assert.equal(out.lines[7], "@@ -1,3 +1,4 @@ header");
	assert.equal(out.lines[8], "  1  keep");
	assert.equal(out.lines[9], "  2 -old");
	assert.equal(out.lines[10], "  2 +new");
	assert.equal(out.lines[11], "  3 +added");
	assert.equal(out.lines[12], "  4  ");
	assert.equal(out.lines[13], "No newline at end of file");
	assert.equal(out.lines.length, 14);
	assert.equal(out.maxScroll, 0);
	for (const l of out.lines) assert.ok(Array.from(l).length <= 40, `too wide: ${l}`);
});

test("layout: the diff area scrolls and announces what is below; scroll clamps", () => {
	const files = [tracked("src/a.ts", 12, 3)];
	const short = layout({ model: model(files, "src/a.ts"), scroll: 0, width: 40, height: 8 });
	// header, blank, 1 row, separator, path = 5 lines; area = 3 of 7 diff lines
	assert.equal(short.lines.length, 8);
	assert.equal(short.maxScroll, 4);
	assert.equal(short.below, 4);
	assert.match(short.lines[7]!, /↓ 4 more/);
	const end = layout({ model: model(files, "src/a.ts"), scroll: 99, width: 40, height: 8 });
	assert.equal(end.below, 0);
	assert.equal(end.lines[7], "No newline at end of file");
});

test("layout: a long file list keeps the focused row in view", () => {
	const files = Array.from({ length: 30 }, (_, i) => tracked(`f${String(i).padStart(2, "0")}.ts`, 1, 0));
	const out = layout({ model: model(files, "f25.ts", []), scroll: 0, width: 40, height: 20 });
	// 40% of 20 = 8 rows; the window ends on the focused row
	assert.match(out.lines[2]!, /↑ 18 more above/);
	assert.match(out.lines[10]!, /▸ f25\.ts/);
	assert.match(out.lines[11]!, /↓ 4 more below/);
	assert.equal(out.lines[13], "f25.ts");
	assert.match(out.lines[14]!, /no hunks/);
});

test("layout: long paths clip from the left so the file name stays readable", () => {
	const files = [tracked("apps/web/app/[locale]/(editorial)/blog/[slug]/page.tsx", 17, 54)];
	const out = layout({ model: model(files, files[0]!.path, []), scroll: 0, width: 40, height: 12 });
	assert.match(out.lines[2]!, /^▸ …\S*page\.tsx +\+17 -54$/);
	assert.ok(Array.from(out.lines[2]!).length <= 40);
});

test("layout: empty states are one line each", () => {
	assert.deepEqual(layout({ model: { files: [], focus: undefined, diff: [] }, scroll: 0, width: 40, height: 10 }).lines, ["diff", "No changes against HEAD."]);
	assert.deepEqual(layout({ model: { files: [], focus: undefined, diff: [], unavailable: true }, scroll: 0, width: 60, height: 10 }).lines, ["diff", "Not a git repository — nothing to compare."]);
});

test("layout: painter wraps segments without changing the visible text", () => {
	const files = [tracked("a.ts", 1, 1)];
	const out = layout({ model: model(files, "a.ts"), scroll: 0, width: 40, height: 20 }, (kind, text) => `<${kind}>${text}</${kind}>`, (s) => Array.from(s.replace(/<\/?[a-z]+>/g, "")).length);
	assert.equal(out.lines[0], "<title>1 file changed</title> <add>+1</add> <del>-1</del>");
	assert.match(out.lines[7]!, /^<del>  2 -old<\/del>$/);
	assert.match(out.lines[2]!, /^<accent>▸ <\/accent><accent>a\.ts<\/accent> +<add>\+1<\/add> <del>-1<\/del>$/);
});

test("layout: fill pads to exactly the height so the panel is a full column", () => {
	const files = [tracked("a.ts", 1, 1)];
	const out = layout({ model: model(files, "a.ts"), scroll: 0, width: 40, height: 30, fill: true });
	assert.equal(out.lines.length, 30);
	assert.equal(out.lines[29], "");
	const empty = layout({ model: { files: [], focus: undefined, diff: [] }, scroll: 0, width: 40, height: 12, fill: true });
	assert.equal(empty.lines.length, 12);
	assert.equal(layout({ model: model(files, "a.ts"), scroll: 0, width: 40, height: 30 }).lines.length, 12);
});
