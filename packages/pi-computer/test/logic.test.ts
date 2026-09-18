import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import {
	ACTIONS,
	captureLine,
	imageBlock,
	parseAnswer,
	parseServerJson,
	previewDetails,
	rejectUnauthorizedFor,
	resolveDataDir,
	resolveServerUrl,
	resolveToken,
	summarizeAct,
	summarizeSnapshot,
	summarizeWindows,
} from "../src/logic.ts";

test("the daemon is found through PICODE_URL or server.json", () => {
	assert.deepEqual(resolveServerUrl({ PICODE_URL: "https://box:8445/" }, null), { ok: true, url: "https://box:8445" });
	assert.equal(resolveServerUrl({ PICODE_URL: "box" }, null).ok, false);
	assert.deepEqual(resolveServerUrl({}, '{"url":"https://localhost:8445"}'), { ok: true, url: "https://localhost:8445" });
	assert.equal(resolveServerUrl({}, null).ok, false);
	assert.equal(parseServerJson("{").ok, false);
	assert.equal(resolveDataDir({}, "/home/x/"), "/home/x/.picode");
	assert.equal(resolveDataDir({ PICODE_DATA: "/d" }, "/home/x"), "/d");
	assert.equal(resolveToken({ PICODE_TOKEN: "ab".repeat(16) }, "zz"), "ab".repeat(16));
	assert.equal(resolveToken({}, "not a token"), "");
	assert.equal(rejectUnauthorizedFor("https://localhost:8445"), false);
	assert.equal(rejectUnauthorizedFor("https://box.example:8445"), true);
});

test("parseAnswer keeps the daemon's words on a refusal", () => {
	assert.deepEqual(parseAnswer(200, '{"action":"screenshot","output":{"ok":true}}'), { ok: true, answer: { action: "screenshot", output: { ok: true } } });
	assert.deepEqual(parseAnswer(403, '{"error":"this agent may not use the computer — turn it on in Settings ▸ Computer"}'), {
		ok: false,
		error: "this agent may not use the computer — turn it on in Settings ▸ Computer",
	});
	assert.equal(parseAnswer(502, "nope").ok, false);
});

test("a capture becomes an image block plus a line that names its pixel space", () => {
	const output = {
		ok: true,
		image: "iVBORw0KGgo=",
		mime: "image/png",
		meta: { display: 1, window: null, width: 1280, height: 800, scale: 2, seq: 3, cursor: [236, 220], preview: "data:image/jpeg;base64,/9j/" },
	};
	assert.deepEqual(imageBlock(output), { type: "image", data: "iVBORw0KGgo=", mimeType: "image/png" });
	assert.equal(imageBlock({ ok: true }), null);
	assert.equal(captureLine(output), "Screenshot of display 1 (1280×800 image, scale 2) — coordinates you send are pixels of this image; cursor at [236, 220].");
	const win = { ...output, meta: { ...output.meta, window: { exe: "notepad.exe", title: "Untitled - Notepad" }, cursor: null } };
	assert.match(captureLine(win), /^Screenshot of window notepad\.exe "Untitled - Notepad"/);
	const details = previewDetails(output, undefined, 1700000000000);
	assert.equal(details?.preview.source, "computer");
	assert.equal(details?.preview.ts, 1700000000000);
	assert.equal(details?.preview.image, "data:image/jpeg;base64,/9j/");
	assert.equal(details?.preview.title, "display 1");
	assert.equal(details?.preview.seq, 3);
	assert.equal(previewDetails({ ok: true }), undefined);
});

test("snapshots and window lists render as lines the model can act on", () => {
	const snap = { window: { exe: "notepad.exe", title: "Untitled - Notepad" }, lines: ['e1 Window "Untitled - Notepad" @center(640,400) depth=0', 'e3 Edit "Text Editor" @center(640,420) depth=2 value="hello"'], dropped: 2 };
	const text = summarizeSnapshot(snap);
	assert.match(text, /^Accessibility tree of notepad\.exe "Untitled - Notepad"/);
	assert.match(text, /e3 Edit "Text Editor"/);
	assert.match(text, /… 2 more node\(s\) not shown$/);
	assert.match(summarizeSnapshot({ lines: [] }), /nothing is exposed/);
	const wins = summarizeWindows({
		displays: [{ index: 1, primary: true, left: 0, top: 0, right: 2560, bottom: 1600 }],
		windows: [{ id: 132458, exe: "notepad.exe", title: "Untitled - Notepad", display: 1, foreground: true, minimized: false }],
	});
	assert.equal(wins, 'display 1 (primary): 2560×1600 at 0,0\nwindow 132458 notepad.exe "Untitled - Notepad" display=1 [front]');
	assert.equal(summarizeWindows({}), "No windows are open.");
});

test("actions without an image answer in one line", () => {
	assert.equal(summarizeAct("cursor_position", { coordinate: [4, 5] }), "Cursor at [4, 5] in the last image.");
	assert.match(summarizeAct("cursor_position", { coordinate: null, screen: [9, 9] }), /outside the last image/);
	assert.equal(summarizeAct("clipboard_read", { text: "hi", dropped: 0 }), "Clipboard text:\nhi");
	assert.equal(summarizeAct("clipboard_read", { text: "" }), "The clipboard has no text.");
	assert.equal(summarizeAct("clipboard_write", { chars: 3 }), "Clipboard set (3 characters).");
	assert.equal(summarizeAct("focus", { window: { exe: "notepad.exe" } }), "Focused notepad.exe.");
	assert.equal(summarizeAct("open", { target: "notepad.exe" }), "Asked Windows to open notepad.exe.");
});

test("the tool's actions are the daemon's 23, no more, no less", () => {
	assert.equal(ACTIONS.length, 23);
	const source = readFileSync(new URL("../extensions/computer.ts", import.meta.url), "utf8");
	assert.match(source, /ACTIONS\.map\(\(a\) => Type\.Literal\(a\)\)/);
	for (const a of ["screenshot", "left_click", "open", "clipboard_write"]) assert.ok(ACTIONS.includes(a as never), a);
});
